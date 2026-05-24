# Evidence

| ID | Type | Status | Evidence |
| --- | --- | --- | --- |
| EV-001 | audit | pass | 当前默认 `LLM_PROVIDER=local_rules`，此前 Web 结论不是大模型推理。 |
| EV-002 | audit | pass | `worker` 当前直接使用 `llm.NewRuleBasedClient()`，即使配置真实 LLM 也不会生效。 |
| EV-003 | real-llm | pass | 已从 health-food `application-local.yml` 本地读取 Qwen 配置并实际调用 DashScope OpenAI-compatible `/chat/completions` 成功；密钥未打印、未写文件。 |
| EV-004 | bug | pass | `openai_compatible` 分类 prompt 漏了 `health_food`，会迫使 health-food 场景回退规则。 |
| EV-005 | code-test | pass | `go test ./internal/config ./internal/llm ./internal/vision ./cmd/worker` 通过，覆盖 profile 加载、strict fallback、health-food 分类和 vision provider 继承。 |
| EV-006 | real-service | pass | 本地启动 health-food `feature/P-2026-009-health-food-readonly`，`GET /food-health/v1/readonly/health-food/healthz` 返回 `source=health-food-readonly-api/local-mysql`、`database=meow_pas`、`readonly=true`。 |
| EV-007 | real-gateway | pass | 通过排障平台 Gateway 直接调用 `get_health_food_recommendation_status`，uid `2054603630081875968`、date `2026-05-23` 返回 `job_status=source_date_mismatch`。 |
| EV-008 | web-real-llm | pass | Web UI 通过真实 Qwen 创建 `case_20260524_000013`，最终摘要指出 `source_date_mismatch`；截图：`programs/P-2026-034-real-llm-model-profile-validation/evidence/web-real-qwen-health-food-result.png`。 |
| EV-009 | mysql | pass | MySQL `tb_troubleshoot_investigation` 记录 `case_id=13`、`model_provider=qwen`、`model_name=qwen-plus`、`investigation_status=finished`、`confidence=0.8000`。 |
| EV-010 | mysql | pass | MySQL `tb_troubleshoot_ai_decision_log` 记录 classify/extract/decide/tool/summarize；`decide_next_action` 的 `selected_tools` 包含 profile、meal、recommendation、logs、similar，且 `augmented=true`。 |
| EV-011 | mysql | pass | MySQL `tb_troubleshoot_tool_call_audit` 记录 5 个 Gateway tool `allowed`，其中 recommendation args 包含 `recommendation_date=2026-05-23`。 |
| EV-012 | full-test | pass | `make test` 通过：Go 全量、Python decision-engine 14 条、根目录 Python 4 条。 |
| EV-013 | secret-scan | pass | `make secret-scan` 通过，未发现密钥写入仓库。 |
| EV-014 | diff-check | pass | `git diff --check` 通过。 |

## 命令摘录

- `MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=*** MYSQL_DATABASE=ai_troubleshooter make migrate-mysql`
- `mvn -pl health-food-srv spring-boot:run -Dspring-boot.run.profiles=local -Dspring-boot.run.arguments='--server.port=18080 --troubleshooter.readonly.api-key=***'`
- `AI_MODEL_PROFILE=qwen AI_MODEL_CONFIG_FILE=/Users/ginseng/IdeaProjects/health-workspace/repos/health-food/health-food-srv/src/main/resources/application-local.yml LLM_MODEL=qwen-plus LLM_ALLOW_RULE_FALLBACK=false CONNECTOR_MODE=http go run ./cmd/dev-server`
- `go test ./internal/decisionbaseline ./internal/llm ./internal/config ./cmd/dev-server`
- `make test`
- `make secret-scan`
- `git diff --check`

## 发现与修复

- 第一次真实 Web LLM 验证失败在 `classify_issue`：Qwen 返回字段别名，严格模式正确暴露错误。已修复为兼容中英文字段别名、缺省 confidence 归一，并把 raw 摘要写入错误。
- 第二次真实 Web LLM 验证只查了用户资料，结论证据不足。已增加 health-food 推荐/token 场景的最低证据工具 guardrail，避免模型选择过少导致排查无效。
