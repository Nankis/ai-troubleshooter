# ERRORS

## E1：上一轮回写旧 Program

- 现象：为了全仓命名一致，上一轮把旧 Program 中的 `orchestrator` 表述同步改成了 `decisionbaseline`。
- 影响：旧 Program 的历史上下文不够纯粹。
- 根因：没有先建立类似 `game` 仓库的入口规则和反复错误计数器，导致把“文档一致性”优先级放到了“Program 历史可追溯性”前面。
- 处理：用户明确旧的不需要恢复；从本 Program 开始，后续独立变更新增 Program，不再回写旧 Program。
- 防复发：新增 `AGENTS.md`、`programs/README.md` 和 `docs/LESSONS.md`；`program-history-rewrite` 计数为 1，后续命中同类场景必须先读复盘。
