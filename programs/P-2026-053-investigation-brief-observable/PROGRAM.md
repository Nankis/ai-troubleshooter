# P-2026-053 Investigation Brief Observable

## Objective

让每个 case 在进入决策层前形成可观察的 `InvestigationBrief`，并落入平台 MySQL / Context Ledger / Web payload。

## Scope

- Python Decision models 增加 InvestigationBrief。
- Agent Platform 构建 brief、记录 Context Ledger、传给 DecisionRequest。
- Web 右侧展示当前目标、假设、约束和可用证据摘要。

## Acceptance

- case API 返回 `investigation_brief`。
- MySQL `tb_troubleshoot_context_ledger` 有 `investigation_brief` 记录。
- Web 页面能看到 brief 内容。
