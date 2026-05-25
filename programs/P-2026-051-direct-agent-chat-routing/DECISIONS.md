# Decisions

## D1: 入口意图以最新用户消息为准

Case 的 `original_text` 可以保留历史上下文用于排障补充，但“是否继续排障”必须优先看最新用户消息。用户问模型、Agent、平台配置或纠错时，不应继承旧 health-food/tool 计划。

## D2: 非排障咨询由决策层 Agent 直接回答

如果已启用本地 Codex/Claude Code 或真实 LLM Decision advisor，平台把非排障消息交给 `llm_decision_agent` 直接回答。直接回答时明确禁止 Gateway、平台经验和工具调用。

## D3: 无 Agent 时仍保持硬边界

没有真实决策 Agent 时，非排障咨询也不能用 `local_rules` 冒充 Agent，只能提示先启用真实决策 Agent。运行时状态类问题也交给真实决策 Agent 基于 `runtime_status` 输入回答，避免平台规则话术继续冒充模型。
