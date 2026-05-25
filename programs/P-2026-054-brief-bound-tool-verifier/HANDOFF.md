# Handoff

## Current Goal

将工具调用计划绑定到明确假设和预期证据，避免“为了查而查”。

## Current State

- 已完成。
- `ToolPlan` 增加 `hypothesis_id`、`expected_evidence`。
- Supervisor 和 Runtime LLM advisor 均补齐工具计划绑定。
- Verifier 校验 reason/hypothesis/expected evidence。
- Agent Platform 工具调用决策日志记录绑定字段。

## Evidence

- `make test`：pass。
- `case_20260525_000068` MySQL tool invocation JSON_EXTRACT 显示 5 个工具都有 hypothesis。

## Commands

- `make test`
- `MYSQL_PWD=<local-password> mysql ... JSON_EXTRACT(input_snapshot_json,'$.hypothesis_id') ...`

## Next Steps

1. 进入 P-2026-055：Brief 驱动工具排序和决策规划。

## Risks

- LLM advisor 输出仍是建议，最终只以 Verifier 后的 plan 为准。
