# RISKS

- 风险：本地 agent 读取过多本地文件或配置。
  - 缓解：Discovery 不读取配置内容；作为 LLM 调用时默认禁用工具或只读 sandbox。
- 风险：本地 LLM 输出不稳定，影响工具计划。
  - 缓解：只作为 advisor；Verifier 过滤不可用工具和越界计划；失败可 fallback 到 deterministic supervisor。
- 风险：把 installed editor 误认为可用 LLM。
  - 缓解：provider descriptor 拆分 `installed`、`llm_capable`、`probe_status`。
- 风险：本地 CLI 调用挂起。
  - 缓解：统一 subprocess timeout。
