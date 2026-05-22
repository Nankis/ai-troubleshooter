# RESULT

已完成架构边界和决策层归属调整：

- README 架构图已将平台数据、知识沉淀、LLM/Vision provider 收回 Agent 平台内部。
- 文档明确业务方只需要提供 readonly business APIs/adapters。
- Python `apps/decision-engine` 被明确为目标 Agent Orchestrator。
- Go `orchestrator` 已改名为 `decisionbaseline`，只作为本地 fallback。
- worker 已改为依赖 `CaseProcessor` 接口。
- 变更已提交并推送到 `main`：`463cb00 Clarify platform architecture boundaries`。
- GitHub Actions CI 已通过。
- 已新增 `AGENTS.md`、`programs/README.md` 和 `docs/LESSONS.md`，把本次回写旧 Program 的错误记录为 `program-history-rewrite`。
