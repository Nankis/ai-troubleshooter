# DECISIONS

## D1. Web 启用优先于环境变量

本轮把本地 agent 的启用状态作为决策 advisor 的运行时来源。只要当前本机 runtime 中有 `enabled=true` 且 `llm_capable=true` 的 provider，下一个 case 的 `llm_decision_agent` 就优先使用它。这样用户不需要理解 `AI_MODEL_PROFILE=local_agent`。

## D2. 环境变量仍作为后备和强制开关

如果没有启用本地 provider，仍保留原有 `DECISION_LLM_ENABLED=true` + 平台 LLM 的后备路径。若显式设置 `DECISION_LLM_ENABLED=false`，则关闭决策 LLM advisor。

## D3. Web 只突出可做决策层的本地 agent

Web “本地 Agent”区域默认只展示已安装且 `llm_capable=true` 的 provider。Cursor editor-only、未安装 Cursor Agent 这类不可用项不再占据主列表，避免用户误解。

## D4. Codex 作为 advisor，不作为 executor

Codex 只输出 action、reason、selected_tools 等决策建议。Verifier 仍拦截不可用工具、超预算和非 Gateway 工具。平台 executor 仍只执行 Verifier 通过的 Gateway readonly tool plan。

## D5. 本地决策 provider 单活

同一个本地 runtime 下只保持一个可做决策层的 provider 为 enabled。Web/API 启用 Codex 会自动关闭 Claude Code，启用 Claude Code 会自动关闭 Codex，避免 case 运行时隐性选择多个 advisor。

## D6. Agent Run 记录真实 advisor 来源

当 `llm_decision_agent` 来自 Web 启用的本地 provider 时，agent run 的 `model_provider/model_name` 记录为 `local_agent/codex` 或对应 provider。平台主模型仍可保持 `local_rules`、Qwen、GPT 等配置，避免把 advisor 来源和主模型来源混在一起。
