# Program: P-2026-037 Real Qwen / GPT Vision Validation

## 背景

P-2026-036 已完成 Context Ledger 和短上下文隔离。下一步最硬 checkpoint 是真实模型接入，先只做 Qwen 和 GPT，不接 Lark/飞书。

## 目标

- Python Agent Platform 支持 Qwen 和 GPT 的统一 LLM 配置。
- Python Agent Platform 支持真实 Vision provider，不再只有本地占位图片识别。
- 支持从本机 health-food `application-local.yml` 读取 Qwen/OpenAI-compatible 配置，但不把 key 写入仓库。
- Web 端通过真实 provider 上传图片并进入排障链路。
- 验证结果写入 Program 证据，明确 Qwen/GPT 的真实验收状态。

## 非目标

- 不接 Lark/飞书真实回调。
- 不接 Claude/Claude Code。
- 不验证真实 health-food 生产 readonly adapter。
- 不在仓库保存任何 API key、token 或密码。

## 验收标准

- Qwen profile 可从 `DASHSCOPE_API_KEY` 或 `AI_MODEL_CONFIG_FILE` 获取 key、base URL、model。
- GPT profile 支持 `OPENAI_API_KEY`、`OPENAI_MODEL`、`OPENAI_BASE_URL` 或通用 `LLM_*`/`VISION_*` 覆盖。
- Vision provider 使用 OpenAI-compatible Chat Completions `image_url` data URL 形态。
- 单测覆盖 Qwen/GPT 配置、Vision request body、无 key fail-fast。
- Web/API 实际上传图片，使用真实 Qwen Vision 返回非占位 OCR/summary。
- 如本机缺 GPT key，必须标记 GPT 真实验收未完成，不能用 Qwen 或 mock 冒充。
