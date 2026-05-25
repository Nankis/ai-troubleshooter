# P-2026-047 Local Agent Runtime Discovery Decision LLM

## 背景

用户要求平台支持自动发现本地安装的 agent 或 AI 工具，例如 Cursor、Claude Code、Codex，并让本地大模型能够作为决策层 LLM agent 参与排障。

## 目标

- 新增 Local Agent Runtime Discovery，自动发现本机 Claude Code、Cursor、Codex 等工具的安装、版本、MCP 配置引用和 LLM 可用性。
- Discovery 结果能注册到平台 runtime，供 Web/API 查看。
- 增加 `local_agent` LLM provider，支持使用本地 agent CLI 做结构化 JSON 输出。
- 在 Python Decision Engine 中加入可选 `llm_decision_agent`，本地 LLM 可建议 action/tools，但最终仍由 Verifier 执行预算、工具可用性和 Gateway-only 校验。
- 补齐文档、测试和本地验证。

## 非目标

- 本轮不自动修改业务代码。
- 本轮不让本地 agent 绕过 Investigation Gateway 查询生产证据。
- 本轮不读取或存储本机 agent 配置中的 token/key。
- 本轮不要求 Cursor editor 本身具备非交互 LLM 能力；若没有 `cursor-agent`，只标记为 installed/editor-only。
