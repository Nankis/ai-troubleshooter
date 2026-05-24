# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | docs | T1 | Program 建立 | pass |
| EV-T2-001 | config | T2/T3 | Qwen profile 从 health-food YAML 读取 key/base/model，Vision 默认 Qwen-VL | pass |
| EV-T2-002 | config | T2/T3 | GPT profile 指向 OpenAI base/model，无真实 key 时 fail-open 禁止 | partial |
| EV-T3-001 | tests | T3 | Qwen/GPT profile、Vision request body、无 key fail-fast | pass |
| EV-T5-001 | real-provider | T5 | 直接调用真实 Qwen Vision provider 识别图片 | pass |
| EV-T5-002 | api+mysql | T5 | `/api/v1/chat` 上传图片，Qwen-VL OCR 写入 case/decision log | pass |
| EV-T5-003 | web+mysql | T5 | Chrome Web 工作台真实上传图片，页面和 MySQL 均显示真实 OCR/provider | pass |
| EV-T6-001 | config | T6 | GPT 真实 key 检查 | blocked |
| EV-T7-001 | tests | T7 | 全量 Go/Python 单测 | pass |
| EV-T7-002 | security | T7 | secret scan | pass |
| EV-T7-003 | lint | T7 | diff whitespace check | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | 2026-05-24 | 检查 `/Users/ginseng/IdeaProjects/health-workspace/.../application-local.yml`，只输出 key 存在性和长度 | Qwen key present，base present，model=`qwen3.6-flash` | 未输出明文 key |
| EV-T2-001 | 2026-05-24 | `AI_MODEL_PROFILE=qwen AI_MODEL_CONFIG_FILE=... DB_DRIVER=memory load_config()` | `llm_provider=openai_compatible`，`llm_model=qwen3.6-flash`，`vision_provider=qwen_openai_compatible`，`vision_model=qwen-vl-plus`，key present | pass |
| EV-T2-002 | 2026-05-24 | `AI_MODEL_PROFILE=gpt AI_MODEL_CONFIG_FILE=... DB_DRIVER=memory load_config()` | `llm_provider=openai`，`llm_base=https://api.openai.com/v1`，`llm_model=gpt-4.1-mini`，`key_present=False` | health-food 中的非 OpenAI `spring.ai.openai` 配置被忽略 |
| EV-T3-001 | 2026-05-24 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_*.py'` | 17 tests passed | pass |
| EV-T5-001 | 2026-05-24 | 直接调用 `OpenAICompatibleVisionClient`，图片 `/tmp/ai-troubleshooter-qwen-vision.png` | `is_real=True`，provider=`qwen_openai_compatible`，model=`qwen-vl-plus`，OCR 含 `HF-USER-VISION`、`TODAY TOKEN USED: 0` | pass |
| EV-T5-002 | 2026-05-24 | `make migrate-mysql`，启动 Gateway `:18080` 和 Agent Platform `:19091`，`GET /healthz` | Agent health 显示 `llm_model=qwen3.6-flash`，`vision_model=qwen-vl-plus` | pass |
| EV-T5-002 | 2026-05-24 | `curl -F message=... -F images=@/tmp/ai-troubleshooter-qwen-vision.png http://127.0.0.1:19091/api/v1/chat` | 生成 `case_20260524_000023`，case OCR 含 `HF-USER-VISION`，`vision_analyze` 为 real Qwen-VL | pass |
| EV-T5-002 | 2026-05-24 | MySQL 查询 `tb_troubleshoot_case`、`tb_troubleshoot_ai_decision_log` | `vision_analyze success qwen_openai_compatible qwen-vl-plus true` | pass |
| EV-T5-003 | 2026-05-24 | Chrome 打开 `http://127.0.0.1:19091/web`，系统文件选择器上传图片并点击发送 | 页面显示图片预览、OCR 文本、进度步骤和最终状态 `NEED_HUMAN_CONFIRMATION` | pass |
| EV-T5-003 | 2026-05-24 | MySQL 查询 `original_text LIKE '%hf-user-chromevision%'` | `case_20260524_000024`，OCR 含 `HF-USER-VISION`，`vision_analyze success qwen_openai_compatible qwen-vl-plus true`，决策日志 10 条 | pass |
| EV-T7-001 | 2026-05-24 | `make test` | Go tests + decision-engine 17 tests + agent-platform 17 tests + root 4 tests passed | pass |
| EV-T7-002 | 2026-05-24 | `make secret-scan` | `Secret scan passed (all).` | pass |
| EV-T7-003 | 2026-05-24 | `git diff --check` | no output | pass |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T5-001 | 2026-05-24 | 合成排障图片输入 | [qwen-vision-input.png](artifacts/qwen-vision-input.png) | 图片只含测试 UID 和测试字段，无真实用户数据 |
| EV-T5-003 | 2026-05-24 | Web 工作台真实上传图片并排查 | [web-qwen-vision-validation.jpg](artifacts/web-qwen-vision-validation.jpg) | 页面展示 OCR、agent 回复、进度步骤、`NEED_HUMAN_CONFIRMATION` |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| Qwen/GPT profile 配置 | T2/T3 | EV-T2-001, EV-T2-002, EV-T3-001 | pass/partial |
| Web/API 真实 Qwen Vision 验收 | T5 | EV-T5-001, EV-T5-002, EV-T5-003 | pass |
| GPT 真实 key 验收 | T6 | EV-T6-001 | blocked |
| 全量测试与扫描 | T7 | EV-T7-001, EV-T7-002, EV-T7-003 | pass |

## 未验证项

- 本机未发现真实 `OPENAI_API_KEY`，health-food 本地 YAML 里的 `spring.ai.openai` 指向非 OpenAI endpoint 且 key 长度为 3，因此 GPT 真实 provider 调用未验收；仅完成配置和单测。
- 本轮不接 Lark/飞书真实回调，符合用户本轮限定范围。
- Gateway 使用 `CONNECTOR_MODE=mock`，本轮只证明真实 Qwen/GPT provider 和 Web 图片链路，不声称真实 health-food 生产 readonly adapter 验收。

## 已知噪音

- Chrome extension 的 `fileChooser.setFiles` 返回 `Not allowed`；最终使用真实 Chrome 系统文件选择器选择同一张本地图片完成 Web 上传验收。
