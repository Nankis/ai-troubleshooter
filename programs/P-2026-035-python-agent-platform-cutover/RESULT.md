# Result

## 交付结论

已完成 Python Agent Platform 主路径切换：Web Chat、Lark/飞书 callback、图片入口、Case API、平台 MySQL、LLM 配置、Decision Engine 编排、经验和能力管理都归 Python；Go 主路径收敛为 Investigation Gateway，只负责业务 readonly tools 的鉴权、scope、限流、timeout、审计和脱敏。

## 关键变化

- 新增 `apps/agent-platform` FastAPI 服务。
- 同一套 handler 提供 `/web/api/*` 和 `/api/v1/*`，Web UI 与正式 API 共用业务逻辑。
- Lark/飞书入口迁到 Python，支持 encrypted callback、verification token、群 allowlist、消息幂等、图片下载入口。
- Python Agent Platform 调用 `apps/decision-engine` 的 Supervisor/HealthFood Agent/Verifier，并把 AI 决策过程写入 `tb_troubleshoot_ai_decision_log`。
- 修复 Web UI 异步轮询竞态，确保页面能看到 Agent 结果。
- README、local runbook、业务方快速接入文档已更新。

## 验证摘要

- Python Agent Platform：8 tests pass。
- Python Decision Engine：16 tests pass。
- Go Gateway 关键包：tests pass。
- `make test`、`make secret-scan`、`git diff --check` pass。
- 本地 MySQL + Go Gateway + Python Agent Platform + Web UI 已跑通，代表 case：`case_20260524_000020`。

## 证据

见 `EVIDENCE.md`。

## 剩余边界

- 本轮没有真实 LLM key 和真实 Lark/飞书外部平台联调，不声明 L4。
- Vision provider 仍需下一轮接真实模型。
- legacy Go dev-server/worker/baseline 后续可单独 Program 清理。
