# Decisions

## D1: `local_rules` 只能做 intake，不能做排障

允许 deterministic/rules 做最低限度的安全入口处理，例如问候语要求用户补充问题、缺少 uid 时要求补充 uid。

一旦输入具备排障信号且实体足以进入排查，必须先确认真实决策 Agent 可用。不可用时立即停止。

## D2: 真实决策 Agent 来源

满足任一条件才允许进入排障：

- Web 侧启用了本地非交互式 Agent provider，例如 Codex CLI / Claude Code。
- 配置了真实 LLM provider，并显式开启 `DECISION_LLM_ENABLED=true`，使 Decision Engine advisor 由真实模型驱动。

## D3: 守门失败不能查 Gateway 或平台经验

守门失败只记录 case、分类、守门日志和回复；禁止读取业务 readonly tools，避免“看起来像排查过”。

