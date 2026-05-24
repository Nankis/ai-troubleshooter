# Handoff

## 当前目标

P-2026-037：接入 Qwen/GPT 的真实 LLM/Vision provider，暂不接 Lark/飞书，并在 Web/API 图片上传链路做真实 Vision 验收。

## 已完成

- 新建 Program，并更新证据、结果、交接。
- 实现 Python `VisionConfig`、Qwen/GPT profile、只读 `AI_MODEL_CONFIG_FILE` 读取、OpenAI-compatible Vision provider。
- Web/API 上传图片时使用配置化图片数量和大小限制。
- Qwen 真实 Vision provider 已直接调用、API 调用、Chrome Web 上传三层验收。
- README、Agent Platform README、local runbook、业务接入 quickstart、deployment checklist 已同步 Qwen/GPT/Vision 配置。
- 当前未发现可用 GPT key，GPT 真实验收记录为 blocked。

## 证据路径

- `programs/P-2026-037-real-qwen-gpt-vision-validation/EVIDENCE.md`
- `programs/P-2026-037-real-qwen-gpt-vision-validation/artifacts/qwen-vision-input.png`
- `programs/P-2026-037-real-qwen-gpt-vision-validation/artifacts/web-qwen-vision-validation.jpg`

## 已运行命令

- 读取 `AGENTS.md`、`programs/README.md`、`docs/VERIFICATION.md`、`docs/LESSONS.md`。
- 检查本机 health-food 配置，只输出 key 存在性和长度，不输出明文。
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_*.py'`
- 真实 Qwen Vision 直接调用：`is_real=True`，OCR 含测试 UID。
- `MYSQL_HOST=127.0.0.1 ... make migrate-mysql`
- 启动本地 Go Gateway `:18080` 和 Python Agent Platform `:19091`。
- `curl -F message=... -F images=@/tmp/ai-troubleshooter-qwen-vision.png http://127.0.0.1:19091/api/v1/chat`
- Chrome 打开 `http://127.0.0.1:19091/web`，系统文件选择器上传图片并点击发送。
- MySQL 查询 `case_20260524_000023`、`case_20260524_000024` 的 OCR 和 `vision_analyze`。
- `make test`
- `make secret-scan`
- `git diff --check`
- 已停止本轮本地 Gateway / Agent Platform 服务。

## 工作树

- 有未提交实现、文档、Program 和截图证据文件；测试和扫描已通过，仍需 commit + push。

## 下一步

1. commit + push main。
2. 后续如提供真实 `OPENAI_API_KEY`，补 GPT 真实调用验收。

## 风险/阻塞

- 本机当前未发现可用 `OPENAI_API_KEY`，GPT 真实验收为阻塞。
- 本轮 Gateway 使用 mock connector，不代表真实 health-food 生产 adapter 验收。
