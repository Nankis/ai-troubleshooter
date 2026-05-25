# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | test | 低信号输入 | 问候语不命中知识库、不调用工具 | pass |
| EV-T3-001 | test | 来源透明 | mock adapter 和 local_rules 在回复中显式暴露 | pass |
| EV-T4-001 | web | Enter 发送 | Web textarea Enter 直接发送，低信号输入转追问 | pass |
| EV-T5-001 | web/mysql | 关闭本地 Agent 后的有效问题 | 仍可规则编排，但回复说明未启用本地决策 Agent 和 mock adapter | pass |
| EV-T6-001 | command | 收口验证 | `make test`、`make secret-scan`、`git diff --check` 全部通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest discover -s apps/decision-engine/tests -p 'test_engine.py'` | pass | 20 tests OK，新增低信号输入测试。 |
| EV-T3-001 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_agent_platform_fastapi.py'` | pass | 29 tests OK，新增 mock/local_rules 来源披露和问候语追问测试。 |
| EV-T4-001 | 2026-05-25 | Browser 打开 `http://127.0.0.1:19148/web`，输入 `hi` 后按 Enter | pass | 生成 case `case_20260525_000056`，状态 `WAITING_USER_REPLY`。 |
| EV-T5-001 | 2026-05-25 | Browser 新建对话，输入 `health-food uid hf-boundary-049 today token quota wrong` 后按 Enter | pass | 生成 case `case_20260525_000057`，回复包含 mock adapter 和 local_rules 说明。 |
| EV-T5-001 | 2026-05-25 | `SELECT ... FROM tb_troubleshoot_case WHERE case_no IN (...)` | pass | case 56 为空 domain 且 `WAITING_USER_REPLY`；case 57 为 health_food 且 `NEED_HUMAN_CONFIRMATION`。 |
| EV-T5-001 | 2026-05-25 | `SELECT case_ref, COUNT(*) FROM tb_troubleshoot_tool_call_audit ...` | pass | case 56 无工具审计；case 57 有 6 次工具调用。 |
| EV-T6-001 | 2026-05-25 | `make test` | pass | Go tests、Decision Engine 20 tests、Agent Platform 40 tests、root tests 4 tests 通过。 |
| EV-T6-001 | 2026-05-25 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-T6-001 | 2026-05-25 | `git diff --check` | pass | 无输出。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-25 | 本地决策 Agent 均关闭，Web textarea 输入 `hi` 后按 Enter。 | `programs/P-2026-049-web-chat-trust-boundaries/web-enter-low-signal.png` | pass |
| EV-T5-001 | 2026-05-25 | 本地决策 Agent 均关闭，Web 提交有效 health-food 问题。 | `programs/P-2026-049-web-chat-trust-boundaries/web-mock-localrules-disclosure.png` | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 直接问候不再命中平台经验 | Task 2 | EV-T2-001, EV-T4-001 | pass |
| 关闭本地决策 Agent 后不假装走 Agent/LLM | Task 3 | EV-T3-001, EV-T5-001 | pass |
| mock data 不冒充真实业务结论 | Task 3 | EV-T3-001, EV-T5-001 | pass |
| Enter 直接发送 | Task 4 | EV-T4-001 | pass |
| 全量检查通过 | Task 6 | EV-T6-001 | pass |

## 未验证项

- 本轮 Gateway 仍为 `CONNECTOR_MODE=mock`，只验证来源披露和链路，不代表 health-food 生产真实 adapter。
- 未验证 Lark/飞书入口。

## 已知噪音

- Browser 自动化逐键输入时出现 Statsig 初始化频率 warning，不影响 Web 表单提交和服务端结果。
