# P-2026-048 Web Enabled Codex Decision Agent

## 背景

用户要求用本地发现到的 Codex agent 替代 Qwen 作为决策层 agent 做真实流程验证，并希望机制更简单：用户在 Web 页面看到可启用的本地 agent，点击启用后即可作为决策层 agent 使用。

## 目标

- Web 启用本地 Codex 后，不再要求额外设置 `AI_MODEL_PROFILE=local_agent` 或重启服务，下一个 case 的 `llm_decision_agent` 能动态使用 Codex。
- Web 只突出展示可作为决策层的本地 agent，降低 Cursor editor-only 等不可用项的干扰。
- 修复当前 Codex CLI 参数兼容问题，确保本机 `codex exec` 能真实跑通 JSON 输出。
- 用真实本地 Codex CLI、真实 Web 页面、真实平台 MySQL 和本地 Gateway 完整验证一次排障流程。

## 非目标

- 不让 Codex 绕过 Gateway 查询生产证据。
- 不让 Codex 自动修改业务代码。
- 不接 Lark/飞书真实回调。
- 本轮不要求真实业务生产 adapter；业务证据侧如使用 mock Gateway 必须在 Evidence 中标明。
