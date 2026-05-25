# DECISIONS

## D1. 默认只发现，不自动启用

Local Agent Discovery 可以扫描 PATH、应用目录和配置文件存在性，但不读取敏感内容，不自动把本地 agent 设为决策模型。启用由环境变量或 Web/API 控制。

## D2. 本地 LLM 只做 advisor，Verifier 仍是硬边界

`llm_decision_agent` 可以建议 action、工具和理由，但所有工具计划仍要通过 Decision Engine Verifier：预算、去重、available tools、Gateway-only 边界都不能被 LLM 绕过。

## D3. 本地 agent 调用必须非交互、结构化、超时

只有支持非交互 CLI 的 provider 才能作为 LLM 使用。输出必须是 JSON object；命令必须有 timeout；失败要记录为 agent report 或错误，不能沉默伪造成功。

## D4. Cursor 先区分 editor 和 cursor-agent

`cursor` editor CLI 可以被发现，但它不是稳定非交互 LLM provider。只有发现 `cursor-agent` 或后续明确的非交互能力时，才标记为 LLM-capable。

## D5. 借鉴 Multica 的 runtime/provider 模式，但不引入依赖

Multica 的 daemon 会扫描本机 agent CLI，并把 runtime、provider、在线状态交给 UI 选择。我们采用同样的抽象方向：

- `LocalAgentProvider` 统一描述 provider id、display name、kind、executable、version、capabilities、config refs、enabled、status。
- `GET/POST /local-agents/discover` 负责扫描和注册本机 runtime。
- `POST /local-agents/enable` 负责显式启用，不自动把新发现 agent 放进决策路径。
- `POST /local-agents/probe` 负责非交互探测。
- 不开放任意 custom args，避免把危险参数注入 Claude/Codex/Cursor Agent 命令。
- 本地 agent 只作为 `llm_decision_agent` advisor；Verifier 仍是硬边界。
