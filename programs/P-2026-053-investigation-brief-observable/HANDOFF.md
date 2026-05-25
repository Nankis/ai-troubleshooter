# Handoff

## Current Goal

让 case 在排障前生成可观察、可落库、可展示的 InvestigationBrief。

## Current State

- 已完成。
- `DecisionRequest` 增加 `InvestigationBrief`。
- Agent Platform 在 Gateway tools/knowledge 加载后生成 brief，写入 `tb_troubleshoot_context_ledger`，并传给 Decision Engine。
- Web 右侧增加 Brief 展示。

## Evidence

- `make test`：pass。
- `case_20260525_000068`：MySQL ledger 写入 `investigation_brief`。
- Web 截图：`programs/P-2026-056-case-scheduler-state-machine/artifacts/web-case-000068-brief.png`。

## Commands

- `make test`
- `MYSQL_PWD=<local-password> mysql ... tb_troubleshoot_context_ledger ... case_20260525_000068`

## Next Steps

1. 进入 P-2026-054：工具计划绑定 brief hypothesis/reason/expected evidence。

## Risks

- brief 只能提供高层目标和假设，不替代 Verifier、安全边界或真实 Gateway 证据。
