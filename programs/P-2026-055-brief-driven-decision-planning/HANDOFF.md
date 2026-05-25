# Handoff

## Current Goal

让决策层按 Brief 聚焦问题目标，减少无关工具查询。

## Current State

- 已完成。
- Supervisor report 增加 brief goal/hypotheses 摘要。
- `select_tools` 按 Brief candidate tools 和 health-food issue type 排序。
- LLM advisor payload 通过 `asdict(request)` 包含 `investigation_brief`。

## Evidence

- `make test`：pass。
- `case_20260525_000068` 真实链路工具顺序符合推荐问题排查路径。

## Commands

- `make test`
- `curl -X POST http://127.0.0.1:19191/web/api/chat ...`

## Next Steps

1. 进入 P-2026-056：case scheduler 状态机和全链路 UI 证据。

## Risks

- 排序优化不能突破工具预算和 Gateway-only 边界。
