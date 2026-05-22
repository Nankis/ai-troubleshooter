# P-2026-008 Web Chat Local Agent MVP

## 背景

架构和流程图已经按用户手绘图明确。本轮进入开发：先不接 Lark/飞书，优先把平台自带 Web Chat 跑起来，让用户下次可以本地启动项目、上传图片、输入问题，由 Agent 走平台 case、MySQL、Gateway mock 工具和模型链路给出排查回复。

## 目标

- 开发内置 Web Chat 页面，支持文字输入、图片上传和查看 case 排查结果。
- Web Chat 请求创建或继续 case，写入平台 MySQL，并同步调用 Agent 排查闭环。
- 图片通过 Vision client 识别，默认可复用主 LLM 的视觉能力，也支持本地 mock。
- Gateway 继续使用 mock connector 注册行情、资产、日志等只读能力，供 Agent 调用。
- 本地 MySQL 使用用户提供的 root 账号运行验证，但密码只进环境变量，不写入仓库。
- 增加 secret scan 和 git hook，提交/推送前阻断 API key、密码、token 等敏感信息。
- 记录框架选择：当前用轻量有限状态决策层，保留后续切 LangGraph 的空间。

## 非目标

- 本轮不接真实 Lark/飞书。
- 本轮不接真实业务服务，只用 mock connector 验证 Gateway 能力注册和调用。
- 本轮不实现完整 LangGraph workflow。
- 本轮不把任何模型 API key、MySQL 密码或其他敏感信息写入仓库。

## 验收标准

- 打开 Web Chat 后可以发送文本和图片。
- 服务端创建 case、写入 message、识别图片并同步返回 Agent 回复。
- 使用 MySQL DSN 运行时，case、message、AI decision log、tool audit 可落库。
- 使用 mock 业务工具时，K线/资产问题能触发有限工具调用并返回证据摘要。
- `scripts/secret-scan.py` 可扫描 staged/all tracked 文件，git hook 已安装到本地 `.git/hooks`。
- `git diff --check`、`go test ./...`、Python 单测通过。
- 本地服务可启动并用至少一个 mock 问题完成排查闭环。

## 外部依赖和敏感信息处理

- Qwen/DashScope key 只从本机已有项目读取到当前 shell 环境，不写文件。
- MySQL 密码只通过 `DB_DSN` 环境变量传入，不写 README、Program、脚本默认值或 committed 配置。
- commit/push 前必须运行 secret scan。
