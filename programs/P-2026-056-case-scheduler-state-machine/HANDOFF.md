# Handoff

## Current Goal

实现最小 case scheduler 状态机并接入真实排障流程。

## Current State

- 已完成。
- 新增 `apps/agent-platform/agent_platform/case_scheduler.py`。
- `process_case` 接入 scheduler claim；orchestrator run finish 时记录 scheduler finish。
- 完成 MySQL + real health-food adapter + Go Gateway + Python Web + Codex local agent L3 验证。

## Evidence

- `make test`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。
- `case_20260525_000068`：查到真实 health-food 推荐/用户数据，5 个只读工具成功。
- `case_20260525_000069`：查不到用户/推荐数据，5 个只读工具成功返回证据。
- Web 截图：`programs/P-2026-056-case-scheduler-state-machine/artifacts/web-case-000068-brief.png`。

## Commands

- `REAL_HEALTH_FOOD_ADAPTER_PORT=19086 ... .venv/bin/python scripts/real-health-food-readonly-adapter.py`
- `HTTP_PORT=18181 CONNECTOR_MODE=http ... go run ./cmd/investigation-gateway`
- `AGENT_PLATFORM_PORT=19191 ... .venv/bin/python -m agent_platform`
- `curl -X POST http://127.0.0.1:19191/web/api/chat ...`
- Playwright Web check via bundled Node runtime.

## Next Steps

1. 最终统一提交并推送 main。

## Risks

- 本阶段只做最小状态机，不引入复杂 worker；后续如要多 worker 并发 claim，需要新增 DB claim/heartbeat 表或锁策略。
