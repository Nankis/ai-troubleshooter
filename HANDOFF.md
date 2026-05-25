# Handoff Index

当前活跃 Program：

- `programs/P-2026-050-require-real-decision-agent/HANDOFF.md`

当前状态：

- P-2026-050 已完成代码修复并通过本地验证，待 commit/push。
- 核心结果：无真实决策 Agent 时阻断排障；启用 Codex 本地 Agent 后才允许进入 Gateway 只读工具排查。
- 证据：`case_20260525_000062` 无 Agent 阻断且 Gateway/Knowledge/Tool 日志为 0；`case_20260525_000061` 启用 Codex 后 `llm_decision_agent=local_agent/codex` 且工具调用成功。

接手规则：

- 先读 `AGENTS.md`、`programs/README.md`，再读当前 Program 的 `HANDOFF.md`。
- Program 暂停、完成里程碑、切换方向或上下文压缩前，必须更新对应 Program 的 `HANDOFF.md`。
