# DECISIONS

## D1：Qwen/GPT 走统一 OpenAI-compatible 封装

Qwen Cloud 和 OpenAI 都支持 Chat Completions 的 `messages[].content[]` 图片输入形态。平台用一套 Python OpenAI-compatible HTTP client 实现文本 JSON 和 Vision，不引入 provider SDK，便于公司模型网关复用。

参考：

- OpenAI Images and Vision: https://platform.openai.com/docs/guides/images-vision
- Qwen Cloud OpenAI compatibility: https://docs.qwencloud.com/api-reference/toolkitframework/openai-compatible/overview

## D2：Vision 随 profile 默认启用，也可以独立配置

`AI_MODEL_PROFILE=qwen` 默认把图片理解接到 `qwen-vl-plus`，复用 DashScope OpenAI-compatible key/base URL；`AI_MODEL_PROFILE=gpt` 默认复用 OpenAI vision-capable 模型。也支持显式 `VISION_PROVIDER`、`VISION_BASE_URL`、`VISION_API_KEY`、`VISION_MODEL`，这样可以实现“主 LLM 用 GPT，图片用 Qwen-VL”或反过来。

## D3：本机配置只读，key 不落仓库

`AI_MODEL_CONFIG_FILE` 只读取本机 YAML 中的模型 provider 配置，运行时注入环境变量。Program 证据只记录 key 是否存在和长度，不记录 key 明文。
