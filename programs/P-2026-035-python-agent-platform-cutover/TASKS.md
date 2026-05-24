# Tasks

- [x] 盘点现有 Go/Python 边界和平台表结构。
- [x] 新增 Python Agent Platform 主服务。
- [x] Python 主服务接入平台 MySQL、Gateway tools、决策层和 Web 静态页面。
- [x] 将 Web Chat、Lark/飞书、图片、Case API、LLM/Vision 和 orchestrator 主路径迁到 Python。
- [x] Go Investigation Gateway 保持为业务 readonly tools、安全、鉴权、scope、限流、timeout、审计和脱敏边界。
- [x] 补充 GPT、Claude、Claude Code、Qwen 的 Python LLM 配置入口和接入文档。
- [x] 补充 Python Lark/飞书 encrypted callback、verification token、chat allowlist、消息幂等和图片下载入口。
- [x] 修复 Web Chat 异步排障轮询竞态，避免页面停在用户消息不显示 Agent 结果。
- [x] 运行 Go/Python 单测、secret scan、diff check。
- [x] 本地启动 Python Agent Platform 和 Go Gateway，走 Web/API 端到端验证。
- [x] 更新 Evidence、Result、Handoff，提交并推送。
