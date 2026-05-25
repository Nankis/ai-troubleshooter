# Handoff

## Current Goal

建立 Brief 驱动排障和 case scheduler 的最小工作流契约。

## Current State

- 已完成。
- 新增 `WORKFLOW.md`、`.workflow/task.schema.json`、`.workflow/tasks/README.md`。
- 新增 `docs/investigation-brief.md` 和 `docs/case-scheduler-design.md`。
- 新增 `scripts/validate_program.py` 并验证 P-052 到 P-056 Program 骨架。

## Evidence

- `python3.13 scripts/validate_program.py ...`：pass，输出 `validated 5 program(s)`。
- `python3.13 -m json.tool .workflow/task.schema.json`：pass。

## Commands

- `python3.13 scripts/validate_program.py programs/P-2026-052-mini-symphony-workflow-and-case-scheduler programs/P-2026-053-investigation-brief-observable programs/P-2026-054-brief-bound-tool-verifier programs/P-2026-055-brief-driven-decision-planning programs/P-2026-056-case-scheduler-state-machine`
- `python3.13 -m json.tool .workflow/task.schema.json`

## Next Steps

1. 进入 P-2026-053：实现 InvestigationBrief 生成、落库、API/Web 展示。

## Risks

- 仅文档/schema 不代表完整排障流程已运行；最终验收必须在后续 Program 走真实 MySQL 和真实 HTTP 链路。
