# RESULT

## 结果摘要

- 已澄清 Decision Engine 运行形态：Web Chat/Lark/飞书不需要业务方单独启动外部 Decision Engine 服务，但排查主路径必须在 Agent Platform 进程内调用 `DecisionEngine.plan()`。
- 已增加回归测试，防止未来绕过 Decision Engine。

## 变更范围

- `docs/business-onboarding-quickstart.md`：重写 6.3，给出主路径流程和 MySQL 决策日志验证方式。
- `apps/decision-engine/README.md`：明确嵌入式运行形态与 standalone 调试入口区别。
- `README.md`、`docs/agent-framework-selection.md`：同步架构边界，移除 Go 决策 fallback 的过期描述。
- `apps/agent-platform/tests/test_agent_platform_fastapi.py`：新增 `RecordingDecisionEngine` 回归测试。

## 验证摘要

- `apps/agent-platform/tests/test_agent_platform_fastapi.py`：16 tests OK。
- `apps/decision-engine/tests/test_engine.py`：18 tests OK。
- 回归：`make test`、`make secret-scan`、`git diff --check` 均通过。

## Commit

- `P-2026-044 clarify decision engine runtime`

## 残留风险

- 本轮只澄清运行形态和增加主路径回归测试，不改变 Decision Engine 部署拓扑。
