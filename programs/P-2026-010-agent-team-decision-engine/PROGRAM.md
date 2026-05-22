# P-2026-010 Agent Team Decision Engine

## 背景

用户确认下一步先在 Python 决策层实现 agents team：Supervisor、Kline Agent、Asset Agent、Knowledge Agent、Verifier，并保持 Go Gateway 不动。

## 目标

- 在 `apps/decision-engine` 内实现轻量 Agent Team 编排。
- 保留现有 `/v1/decisions/plan` 外部契约，增加可观测的 agent reports 和 verifier 信息。
- Kline / Asset 专家负责领域必要字段和工具计划。
- Knowledge Agent 负责平台经验优先判断，但需要遵守实时校验约束。
- Verifier 负责预算、可用工具、去重、停止条件和最终响应收敛。
- 更新 Python 决策层 README、OpenAPI 和 Program 证据。

## 非目标

- 不改 Go Gateway、Go worker 或 Go baseline。
- 不引入 LangChain / LangGraph 作为本轮运行时依赖。
- 不调用真实 LLM、真实生产服务或真实业务 DB。
- 不提交 API key、token、密码或本地私有配置。
