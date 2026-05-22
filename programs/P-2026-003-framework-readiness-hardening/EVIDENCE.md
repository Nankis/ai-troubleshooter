# EVIDENCE

## EV-T1-PROGRAM

- 状态：PASS
- 证据：Program 文件已建立。

## EV-T2-CONTROL-AUTH

- 状态：PASS
- 证据：`internal/httpauth/bearer_test.go` 覆盖控制面 Bearer 鉴权；prod smoke 中 `/knowledge` 未带 token 返回 401、带 `Authorization: Bearer ct` 的 `/cases/{case_no}` 返回 200。

## EV-T3-PROD-VALIDATION

- 状态：PASS
- 证据：`internal/config/validation_test.go` 覆盖 Gateway、Control API、Lark prod fail-closed；手工验证 `APP_ENV=prod go run ./cmd/investigation-gateway`、`./cmd/baseline-orchestrator`、`./cmd/lark-bot` 在缺少安全配置时退出并报告 invalid config。

## EV-T4-POLICY

- 状态：PASS
- 证据：`internal/policy/policy_test.go` 覆盖 agent 配置 allowed groups 时缺少 `chat_id` 必须 deny。

## EV-T5-FINAL

- 状态：PASS
- 证据：`git diff --check`、`go vet ./...`、`make test`、`go test -race ./...` 均通过；prod smoke 覆盖 Lark case -> worker -> Gateway tools -> root cause -> knowledge 查询。

## EV-T6-AUDIT-PERSISTENCE

- 状态：PASS
- 证据：`internal/storage/mysql/audit.go` 实现 MySQL audit sink；`storage.Open` 为 MySQL 返回持久化 audit sink，为本地 memory 返回 memory sink；Gateway 支持 `NewFromConfigWithAudit` 注入 audit sink；`migrations/001_initial.sql` 的 `tb_troubleshoot_tool_call_audit` 使用 `case_ref`/`investigation_ref` 与工具调用记录类型一致。
