# Handoff Index

当前活跃 Program：

- `programs/P-2026-044-decision-engine-runtime-clarity/HANDOFF.md`

当前状态：

- P-2026-044 已完成：Web Chat 不需要单独启动外部 Decision Engine 服务，但排查主路径必须在 Agent Platform 进程内调用 `DecisionEngine.plan()`；已加回归测试防止绕过。

接手规则：

- 先读 `AGENTS.md`、`programs/README.md`，再读当前 Program 的 `HANDOFF.md`。
- Program 暂停、完成里程碑、切换方向或上下文压缩前，必须更新对应 Program 的 `HANDOFF.md`。
