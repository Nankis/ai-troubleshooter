# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | test | Web enabled runtime advisor | Web 启用 Codex 后无需重启即可作为决策 advisor | pass |
| EV-T2-002 | test | 单活 provider | 同一 runtime 只保持一个本地决策 provider enabled | pass |
| EV-T3-001 | command | Codex CLI | 真实 Codex CLI 非交互 JSON 调用可用 | pass |
| EV-T4-001 | web | Web 本地决策 Agent | 页面只突出可做决策层的本地 agent，并能点击启用 Codex | pass |
| EV-T5-001 | web/mysql | 完整流程 | Web 提交 case，Codex 作为 `llm_decision_agent`，Gateway 只读工具执行并落库 | pass |
| EV-T5-002 | mysql | Runtime 状态 | MySQL 中 Codex enabled=true、Claude false | pass |
| EV-T5-003 | mysql | Tool audit | Gateway 工具审计记录 6 次 allowed 只读调用 | pass |
| EV-T6-001 | docs | 文档同步 | README / runbook / onboarding 已说明 Web 动态启用本地决策 agent | pass |
| EV-T7-001 | command | 收口验证 | `make test`、`make secret-scan`、`git diff --check` 全部通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | 2026-05-25 | `python3.13 -m py_compile apps/agent-platform/agent_platform/decision_advisor.py apps/agent-platform/agent_platform/service.py apps/agent-platform/agent_platform/local_agents.py` | pass | 语法检查通过。 |
| EV-T2-001 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_agent_platform_fastapi.py'` | pass | 27 tests OK。 |
| EV-T3-001 | 2026-05-25 | `complete_json_with_local_agent(... provider_id='codex' ...)` | pass | 返回 `({'ok': True, 'provider': 'openai', 'selected_tools': ['get_health_food_ai_quota']}, 'local_agent', 'codex')`。 |
| EV-T5-001 | 2026-05-25 | `MYSQL_DATABASE=ai_troubleshooter make migrate-mysql` | pass | 迁移脚本确认 001-008 已应用到 canonical schema。 |
| EV-T5-001 | 2026-05-25 | `DB_DRIVER=mysql ... HTTP_PORT=18148 CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false go run ./cmd/investigation-gateway` | pass | Gateway 启动在 `http://localhost:18148`。 |
| EV-T5-001 | 2026-05-25 | `DB_DRIVER=mysql ... AGENT_PLATFORM_PORT=19148 GATEWAY_ENDPOINT=http://127.0.0.1:18148 ... .venv/bin/python -m agent_platform` | pass | Agent Platform 启动在 `http://127.0.0.1:19148`。 |
| EV-T5-001 | 2026-05-25 | `SELECT ... FROM tb_troubleshoot_agent_run WHERE case_id=54` | pass | `llm_decision_agent` 为 `model_provider=local_agent`、`model_name=codex`。 |
| EV-T5-002 | 2026-05-25 | `SELECT JSON_EXTRACT(provider_list_json, '$[*].provider_id'), JSON_EXTRACT(provider_list_json, '$[*].enabled') ...` | pass | providers=`["claude_code","codex","cursor","cursor_agent"]`，enabled=`[false,true,false,false]`。 |
| EV-T5-003 | 2026-05-25 | `SELECT tool_name, policy_decision ... FROM tb_troubleshoot_tool_call_audit WHERE case_ref='case_20260525_000054'` | pass | 6 个工具均 `allowed`，包含 user profile、AI quota、meal records、recommendation status、logs、similar cases。 |
| EV-T7-001 | 2026-05-25 | `make test` | pass | Go tests、Decision Engine 19 tests、Agent Platform 38 tests、root tests 4 tests 通过。 |
| EV-T7-001 | 2026-05-25 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-T7-001 | 2026-05-25 | `git diff --check` | pass | 无输出。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-25 | 打开 `http://127.0.0.1:19148/web`，右侧“本地决策 Agent”发现 Claude Code 和 Codex，隐藏不可做决策层的 Cursor 项，点击 Codex 启用。 | `programs/P-2026-048-web-enabled-codex-decision-agent/web-codex-enabled.png` | pass |
| EV-T5-001 | 2026-05-25 | Web 输入并提交 `health-food uid hf-codex-provider-048 today ai quota exhausted, use Codex decision agent`。 | `programs/P-2026-048-web-enabled-codex-decision-agent/web-codex-provider-result.png` | pass |
| EV-T5-001 | 2026-05-25 | 页面显示 `llm_decision_agent · decision_advisor`，进度完成到 Verifier，并返回 Gateway 证据摘要。 | `case_20260525_000054` | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| Web 启用 Codex 后动态进入 Python 决策层 | Task 2 | EV-T2-001, EV-T5-001 | pass |
| Web 只突出可作为决策层的本地 agent | Task 3 | EV-T4-001 | pass |
| Codex CLI 真实非交互调用可用 | Task 4 | EV-T3-001 | pass |
| 单测覆盖无需重启、单活 provider | Task 5 | EV-T2-001, EV-T2-002 | pass |
| 完整本地流程跑通 | Task 6 | EV-T5-001, EV-T5-002, EV-T5-003 | pass |
| 文档同步和最终检查 | Task 7 | EV-T6-001, EV-T7-001 | pass |

## 未验证项

- 本轮业务证据侧 Gateway 使用 `CONNECTOR_MODE=mock`，因此只能证明平台、Web、真实 Codex advisor、MySQL、Gateway 审计和只读工具链路；不能声称 health-food 生产真实数据已验证。
- 本轮没有接 Lark/飞书真实回调，也没有验证 Vision provider。

## 已知噪音

- Browser 自动化逐字符按键时出现 Statsig 初始化频率 warning，不影响本地 Web 交互和平台服务结果。
- Browser `locator.type/fill` 受虚拟剪贴板限制，最终用真实页面 click + key press 提交 Web case。
