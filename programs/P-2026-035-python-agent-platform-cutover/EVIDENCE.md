# Evidence

| ID | 等级 | 状态 | 说明 |
| --- | --- | --- | --- |
| EV-001 | L0 | pass | README、architecture decisions、local runbook、business onboarding 已更新为 Python Agent Platform + Go Investigation Gateway 主路径。 |
| EV-002 | L1 | pass | Python Agent Platform 单测 8 条通过，Decision Engine 单测 16 条通过。 |
| EV-003 | L1 | pass | Go Gateway 关键包测试通过。 |
| EV-004 | L1 | pass | `compileall`、`make secret-scan`、`git diff --check` 通过。 |
| EV-005 | L2/L3 | pass | 本地启动 Go Gateway + Python Agent Platform，使用 MySQL 和 mock readonly connector 完成 API 排障，case/decision/tool audit 均落库。 |
| EV-006 | L2/L3 | pass | Web UI 通过浏览器实际输入并提交问题，页面显示 Agent 结论和 11 条决策日志；截图见 artifacts。 |
| EV-007 | L2 | pass | 本地 HTTP 验证 Lark encrypted callback challenge、明文降级拒绝、消息幂等和图片未配置提示。 |

## 命令记录

| 时间 | 命令 | 结果 |
| --- | --- | --- |
| 2026-05-24 | `.venv/bin/python -m pip install -e apps/agent-platform` | pass，安装 FastAPI/PyMySQL/cryptography 等依赖。 |
| 2026-05-24 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m compileall apps/agent-platform/agent_platform apps/decision-engine/decision_engine` | pass。 |
| 2026-05-24 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_*.py'` | pass，8 tests。 |
| 2026-05-24 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'` | pass，16 tests。 |
| 2026-05-24 | `go test ./cmd/investigation-gateway ./internal/gateway ./internal/connectors ./internal/storage ./internal/storage/mysql ./internal/capability ./internal/policy ./internal/httpauth ./internal/masking ./internal/tool` | pass。 |
| 2026-05-24 | `make test` | pass。 |
| 2026-05-24 | `make secret-scan` | pass。 |
| 2026-05-24 | `git diff --check` | pass。 |

## 本地服务验证

启动服务，敏感 DSN 已脱敏：

```bash
DB_DRIVER=mysql DB_DSN='<redacted>' CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false HTTP_PORT=18080 go run ./cmd/investigation-gateway
PYTHONPATH=apps/agent-platform:apps/decision-engine DB_DRIVER=mysql DB_DSN='<redacted>' GATEWAY_ENDPOINT=http://127.0.0.1:18080 AGENT_PLATFORM_PORT=19091 AI_MODEL_PROFILE=local_rules LLM_ALLOW_RULE_FALLBACK=false .venv/bin/python -m agent_platform
```

健康检查：

```bash
curl -sS http://127.0.0.1:18080/healthz
# {"ok":true}

curl -sS http://127.0.0.1:19091/healthz
# service=agent-platform, decision_layer=python, llm_provider=local_rules
```

API 同步排障：

```bash
curl -sS -X POST http://127.0.0.1:19091/api/v1/chat \
  -F 'message=health-food uid hf-user-fastapi 今日没有每日推荐' \
  -F 'async=0'
```

结果：`case_20260524_000015`，状态 `NEED_HUMAN_CONFIRMATION`，Gateway 调用 6 个只读工具并写入 AI 决策日志。

MySQL 现场查询：

```text
case_20260524_000015 uid=hf-user-fastapi issue_domain=health_food issue_type=每日推荐缺失 status=NEED_HUMAN_CONFIRMATION decision_logs=11
decision logs: classify_extract=1, gateway_tool_discovery=1, knowledge_retrieval=1, orchestrator_plan=1, tool_invocation=6, summarize_findings=1
tool audit: get_health_food_user_profile/get_health_food_ai_quota/get_health_food_meal_records/get_health_food_recommendation_status/search_logs_by_service/get_similar_cases all policy_decision=allowed
```

缺少 uid 反问：

```bash
curl -sS -X POST http://127.0.0.1:19091/api/v1/chat \
  -F 'message=health-food 今日没有每日推荐' \
  -F 'async=0'
```

结果：`case_20260524_000016`，状态 `WAITING_USER_REPLY`，回复要求补充“业务 uid”，不会要求用户写内部字段格式或 timezone 字符串。

Lark encrypted callback 本地验证：

```text
LARK_VERIFICATION_TOKEN=token_api
LARK_ENCRYPT_KEY=encrypt_key_api
LARK_ALLOWED_CHAT_IDS=oc_api
POST /lark/events encrypted challenge -> 200 {"challenge":"challenge_api"}
POST /lark/events plain challenge while encrypt key configured -> 400
POST /lark/events encrypted message twice -> first creates case_20260524_000017, second duplicate=true
```

Web UI 浏览器验证：

- 使用 Browser 打开 `http://127.0.0.1:19091/web`。
- 通过虚拟键盘实际输入 `health-food uid hf-user-webfix today token quota wrong` 并点击发送。
- 页面显示 Agent 回复：`case_20260524_000020`，`已通过 Gateway 查询 6 个只读工具`，并显示 `已记录 11 条决策 · 最新：summarize_findings / success`。
- 截图：`programs/P-2026-035-python-agent-platform-cutover/artifacts/web-ui-case-000020.png`

MySQL 现场查询：

```text
case_20260524_000020 uid=hf-user-webfix issue_domain=health_food issue_type=AI配额异常 status=NEED_HUMAN_CONFIRMATION
decision logs: classify_extract=1, gateway_tool_discovery=1, knowledge_retrieval=1, orchestrator_plan=1, tool_invocation=6, summarize_findings=1
tool audit: 6 rows, all policy_decision=allowed
```

## 发现并修复的问题

- Web Chat async 竞态：提交后第一次轮询可能在后台任务启动前读到 `NEW`，页面停止轮询，只显示用户消息。修复：异步提交返回前先将 case 标为 `READY_TO_INVESTIGATE`，确保轮询持续到 Agent 结果落库。

## 未验证项

- 本轮使用 `CONNECTOR_MODE=mock` 和 `AI_MODEL_PROFILE=local_rules`，只证明本地链路、MySQL 落库、Gateway 契约和 Web UI 可用；不宣称真实业务或真实大模型验收。
- 未接真实 Lark/飞书外部平台回调；本轮只做本地 encrypted HTTP 验证。
- 未接真实 Vision provider；Lark/Web 图片入口已到 Python，但未用 Qwen-VL 等真实视觉模型验收。
- 未删除 legacy Go dev-server/worker/baseline；本轮只把主路径、文档和新增能力切到 Python。
