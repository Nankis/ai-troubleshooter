# TASKS

## Task 1: [x] 建立 Program 和安全边界

- 文件：`programs/P-2026-008-web-chat-local-agent-mvp/**`
- 验收：Program 记录目标、Scope、敏感信息禁止提交规则。
- Evidence：`EV-T1-001`

## Task 2: [x] Web Chat 页面和 API

- 文件：`web/**`、`cmd/dev-server/**`、必要的 `internal/**`
- 验收：
  - `GET /` 或 `/web` 返回 Web Chat 页面。
  - `POST /web/api/chat` 支持文本和图片 multipart。
  - 返回 case、reply、messages、decision logs 和 tool call ids。
- Evidence：`EV-T2-001`、`EV-T2-002`、`EV-T2-003`、`EV-T2-004`

## Task 3: [x] MySQL 本地迁移和落库验证

- 文件：`scripts/**`、`migrations/**`、`README.md`
- 验收：
  - 提供不含密码的迁移脚本，使用 `DB_DSN` 或独立 env。
  - 本地 MySQL 可创建 schema 并应用 migrations。
  - Web Chat case 能落库。
- Evidence：`EV-T3-001`、`EV-T3-002`

## Task 4: [x] Agent 决策层和 Qwen 本地运行配置

- 文件：`internal/llm/**`、`internal/vision/**`、`apps/decision-engine/**`、`docs/**`
- 验收：
  - Qwen/DashScope 可通过 OpenAI-compatible 配置使用。
  - Python decision-engine 文档记录框架选择和后续 LangGraph 迁移点。
  - 不提交 API key。
- Evidence：`EV-T4-001`

## Task 5: [x] Secret scan 和 git hook

- 文件：`scripts/secret-scan.py`、`githooks/**`、`README.md` 或 `docs/**`
- 验收：
  - staged/all tracked 扫描可运行。
  - 本地 `.git/hooks/pre-commit` 和 `pre-push` 已安装。
  - 能阻断真实 key/password/token 样式。
- Evidence：`EV-T5-001`、`EV-T5-002`

## Task 6: [x] 验证闭环

- 验收：
  - `git diff --check`
  - `go test ./...`
  - Python 单测
  - secret scan
  - 本地服务启动
  - 至少一个 mock K线或资产问题完成 Web Chat 排查闭环
- Evidence：`EV-T6-001`

## Task 7: [x] 回写并提交推送

- 验收：`EVIDENCE.md`、`RESULT.md`、`HANDOFF.md` 完整，commit/push 成功。
- Evidence：最终 git 结果见 Codex final response。
