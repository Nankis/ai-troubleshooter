# P-2026-054 Brief Bound Tool Verifier

## Objective

让每次工具调用计划都绑定 brief hypothesis/reason/expected evidence，并由 Verifier 校验。

## Scope

- ToolPlan 增加 hypothesis/expected_evidence 字段。
- Supervisor 和 LLM advisor 生成工具计划时补齐 brief 绑定信息。
- Verifier 拒绝没有 reason 或 hypothesis 的工具计划。
- Agent Platform 记录工具调用时保留这些决策字段。

## Acceptance

- 单测证明缺少 hypothesis/reason 的计划会被 Verifier 拦截或降级。
- MySQL 决策日志中工具调用 input 包含 hypothesis/expected_evidence。
