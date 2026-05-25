# Risks

- 部分旧单测默认 `local_rules` 就能完整排障，需要调整为先启用 fake local Agent。
- 真实 Qwen/GPT 验收依赖本地 key；本 Program 先验证守门和本地 Agent 路径，真实外部模型继续按后续验收补齐。
- 如果用户只是询问平台状态，不应被当作生产排障；需要单独回答运行时状态。

