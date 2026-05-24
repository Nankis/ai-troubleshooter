# Handoff

## 当前目标

P-2026-035：执行大改，主路径迁为 Python Agent Platform + Go Investigation Gateway。Python 承接 Web/Lark/飞书/图片/Case API/平台 MySQL/LLM/orchestrator/经验沉淀；Go 收敛为 readonly Investigation Gateway。

## 已完成

- 新增 `apps/agent-platform` FastAPI 服务。
- 增加 `/web/api/*` 与 `/api/v1/*` 同 handler。
- Web Chat 使用 Python Agent Platform 调用 Decision Engine 和 Go Gateway。
- Lark/飞书 Python callback 支持 challenge、encrypted callback 解包、verification token、群 allowlist、消息幂等和图片下载入口。
- Go `cmd/investigation-gateway` 增加动态能力 reload 控制面，保持 readonly Gateway 职责。
- README、local runbook、业务方接入文档、AGENTS 规则和 Program 文档已更新。
- 发现并修复 Web Chat async 轮询竞态。

## 证据路径

- `programs/P-2026-035-python-agent-platform-cutover/EVIDENCE.md`
- `programs/P-2026-035-python-agent-platform-cutover/artifacts/web-ui-case-000020.png`

## 已运行命令

- `.venv/bin/python -m pip install -e apps/agent-platform`
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m compileall apps/agent-platform/agent_platform apps/decision-engine/decision_engine`
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_*.py'`
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'`
- `go test ./cmd/investigation-gateway ./internal/gateway ./internal/connectors ./internal/storage ./internal/storage/mysql ./internal/capability ./internal/policy ./internal/httpauth ./internal/masking ./internal/tool`
- `make test`
- `make secret-scan`
- `git diff --check`
- 本地启动 Gateway `:18080` 和 Agent Platform `:19091`，通过 API、MySQL、Browser 验证。

## 当前服务状态

- 交付前应停止本地 `go run ./cmd/investigation-gateway` 和 `.venv/bin/python -m agent_platform`，除非用户要求继续试用。

## 工作树

- 待提交并推送 `main`。

## 下一步

- 提交并推送本 Program。
- 后续可新 Program：真实 LLM/Vision 接入验收、真实 Lark/飞书外部联调、legacy Go baseline 清理、真实 health-food production readonly adapter 验收。

## 风险/阻塞

- 本轮本地排障使用 `local_rules` 和 mock connector，不能声明真实大模型或真实业务生产验收。
- 真实外部 Lark/飞书和真实 Vision provider 仍未 L4 验证。
