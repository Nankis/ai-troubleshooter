# Decisions

## D-001 Go 不再作为主决策路径

`internal/llm` 和 `internal/decisionbaseline` 保留为历史 baseline / compatibility test，不再出现在主部署路径里。生产目标路径是 Python Agent Platform 调用 Go Investigation Gateway。

## D-002 Web/Lark/图片入口属于 Agent Platform

Web Chat、Lark/飞书 callback、图片上传/下载、Vision、LLM、orchestrator 都归 Python Agent Platform。Go Gateway 只处理业务 readonly tools。

## D-003 先切主路径，再逐步删除遗留

本轮不硬删 Go dev-server/worker，避免破坏旧验证和已有测试；但 README、runbook、接入文档和新增代码都以 Python Agent Platform 为主。

## D-004 同一套 handler 同时服务 Web 和正式 API

Web 工作台继续使用 `/web/api/*`，自动化和接入方使用 `/api/v1/*`；两者共用 Python Agent Platform 的 handler 和 service，避免再次出现 Web、API、Lark 三套逻辑分叉。

## D-005 Lark/飞书安全能力在 Python 承接

迁移后不再依赖 Go lark-bot 做主路径。Python Agent Platform 负责 encrypted callback 解包、verification token、chat allowlist、消息幂等和图片下载入口；真实外部平台送达仍需公司 bot 凭据联调。
