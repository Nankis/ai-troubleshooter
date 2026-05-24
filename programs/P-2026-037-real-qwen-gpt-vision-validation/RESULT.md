# RESULT

## 交付摘要

- Python Agent Platform 已支持 Qwen/GPT profile 和独立 Vision provider 配置，LLM/Vision 仍全部在 Python 侧，Go Gateway 不接模型。
- Qwen profile 支持从 `DASHSCOPE_API_KEY` / `QWEN_API_KEY` 或只读 `AI_MODEL_CONFIG_FILE` 读取配置；本机 health-food YAML 验证通过。
- GPT profile 支持 OpenAI base/model/key；本机未发现真实 OpenAI key，因此真实 GPT 调用验收阻塞，不用 Qwen 冒充。
- Web/API 图片上传链路已用真实 Qwen-VL 验收，OCR、case、AI 决策日志和进度面板均可见。

## 验证摘要

- 直接调用真实 Qwen Vision：`qwen_openai_compatible / qwen-vl-plus / is_real=true`，OCR 识别出 `HF-USER-VISION`、`TODAY TOKEN USED: 0`。
- API 链路：`case_20260524_000023`，`tb_troubleshoot_ai_decision_log` 中 `vision_analyze=success`。
- Web 链路：Chrome 系统文件选择器上传图片，页面显示 OCR 和最终状态；MySQL 中 `case_20260524_000024` 记录 `vision_analyze=success`。
- 全量验证：`make test`、`make secret-scan`、`git diff --check` 通过；agent-platform 单测 17 条通过。

## 残留风险

- GPT 真实调用需要提供真实 `OPENAI_API_KEY` 后补验收。
- 本轮 Gateway 使用 mock connector，不能视为真实 health-food 生产 adapter 验收。
- Lark/飞书真实回调不在本轮范围。
