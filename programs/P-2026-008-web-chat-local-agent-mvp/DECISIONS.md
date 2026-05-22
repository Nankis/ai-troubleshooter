# DECISIONS

## D1：Web Chat 先内置纯 HTML/CSS/JS

原因：一期本地验证优先，需要减少 Node/Vite/Next.js 依赖和构建复杂度。开源聊天 UI 只借鉴交互形态，不复制大框架。

## D2：决策层先用轻量有限状态编排，保留 LangGraph 迁移点

原因：LangGraph 适合长时程有状态 agent，但本轮要先跑通安全、审计、限流和 Gateway mock 闭环。先用当前 Go baseline + Python decision-engine skeleton 的有限工具计划，后续再把状态图迁到 LangGraph。

## D3：Qwen 通过 OpenAI-compatible 配置接入

原因：当前 `internal/llm` 和 `internal/vision` 已支持 OpenAI-compatible chat completions，DashScope compatible-mode 可以复用，不需要引入模型 SDK。

## D4：secret scan 作为本地硬门禁

原因：用户强制要求 API key、密码、敏感信息禁止 push。仓库提交 hook 模板，并在本地 `.git/hooks` 安装 pre-commit/pre-push。
