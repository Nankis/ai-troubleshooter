# Handoff Index

当前活跃 Program：

- `programs/P-2026-051-direct-agent-chat-routing/HANDOFF.md`

当前状态：

- P-2026-050 已完成并推送 main：无真实决策 Agent 时阻断排障。
- P-2026-051 已完成代码修复、本地 Web/MySQL 验证和最终检查，进入 main 提交推送。
- 核心结果：同一 case 内最新消息如果是模型/Agent/平台咨询或用户纠错，平台走真实 `llm_decision_agent` direct chat；没有真实决策 Agent 时阻断，不再继承旧业务上下文查 Gateway。
- 证据：`case_20260525_000063` 先完成 health-food mock Gateway 排障，随后 `my claude code cannot work` follow-up 只新增 `decision_agent_direct_answer`，无额外 Gateway/Knowledge/Tool 记录。
- 补充证据：`case_20260525_000064` 中 `现在是用什么模型` 由 `llm_decision_agent / local_agent / codex` 回答，`tool_invocation=0`。
- 最终话术证据：`case_20260525_000065` 明确“本次真实决策 Agent=codex/codex，local_rules 只是平台主 LLM profile”，`tool_invocation=0`。

接手规则：

- 先读 `AGENTS.md`、`programs/README.md`，再读当前 Program 的 `HANDOFF.md`。
- Program 暂停、完成里程碑、切换方向或上下文压缩前，必须更新对应 Program 的 `HANDOFF.md`。
