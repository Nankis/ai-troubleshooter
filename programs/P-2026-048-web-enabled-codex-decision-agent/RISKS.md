# RISKS

- 风险：真实 Codex CLI 调用耗时或消耗额度。
  - 缓解：设置 `LLM_TIMEOUT_SECONDS`，只在验收中跑少量 case；Evidence 记录真实调用结果。
- 风险：Codex 输出非 JSON 或夹带说明。
  - 缓解：继续使用 `--output-schema`、JSON 清洗和失败显式记录；失败不伪造成成功。
- 风险：用户误启不可用 editor。
  - 缓解：Web 主列表只展示可做决策层的 provider，服务端仍拒绝 `llm_capable=false` 的启用。
- 风险：本地 agent 被误认为可以直接查生产。
  - 缓解：仅作为 advisor，Verifier 和 Gateway-only 边界保持不变。
