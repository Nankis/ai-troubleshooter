# RESULT

## 结果摘要

- 每个只读工具调用计划都必须绑定 `hypothesis_id`、`reason` 和 `expected_evidence`，并由 Verifier 校验。

## 变更范围

- `apps/decision-engine/decision_engine/models.py`
- `apps/decision-engine/decision_engine/agent_team.py`
- `apps/agent-platform/agent_platform/decision_advisor.py`
- `apps/agent-platform/agent_platform/service.py`

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| ToolPlan 增加 hypothesis/expected_evidence | 完成 | EV-P054-001 |
| Supervisor 计划绑定 brief hypothesis | 完成 | EV-P054-001 |
| Runtime LLM advisor 计划绑定 brief hypothesis | 完成 | EV-P054-001 |
| Verifier 增加校验 | 完成 | EV-P054-002 |
| 工具调用决策日志写入绑定信息 | 完成 | EV-P054-003 |
| 单测与 MySQL 验证 | 完成 | EV-P054-002 / EV-P054-003 |

## 验证摘要

- `make test`：pass。
- L3：`case_20260525_000068` MySQL tool invocation 日志包含 hypothesis 和 expected evidence。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| 缺少 hypothesis/reason 的计划会被 Verifier 拦截或降级 | pass | EV-P054-002 |
| MySQL 决策日志中工具调用 input 包含 hypothesis/expected_evidence | pass | EV-P054-003 |

## Commit

- 待最终统一提交。

## 残留风险

- LLM advisor 仍可能给出不佳排序，但 Verifier 会拦截不可用工具和缺绑定计划。
