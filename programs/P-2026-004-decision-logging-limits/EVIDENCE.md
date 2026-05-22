# EVIDENCE

## EV-T1-PROGRAM

- 状态：PASS
- 证据：Program 文件已建立。

## EV-T2-DECISION-STORE

- 状态：PASS
- 证据：新增 `caseflow.AIDecisionLog`、`Store.AddAIDecisionLog`、`Store.ListAIDecisionLogs`；memory store 和 MySQL store 均已实现；新增 `migrations/003_ai_decision_logs.sql`。

## EV-T3-ORCHESTRATOR-LOGS

- 状态：PASS
- 证据：`internal/decisionbaseline/orchestrator.go` 记录 `classify_issue`、`extract_entities`、`required_fields_check`、`decide_next_action`、`tool_invocation`、`tool_query_stopped`、`summarize_findings`、`process_failure`；smoke 中 `/cases/{case_no}/ai-decisions` 返回 10 条决策日志。

## EV-T4-LIMITS

- 状态：PASS
- 证据：`internal/decisionbaseline/orchestrator_test.go` 覆盖工具失败上限停止继续查询，以及 context timeout 后 case 转 `FAILED` 并记录 timeout `process_failure`。

## EV-T5-FINAL

- 状态：PASS
- 证据：`git diff --check`、`go vet ./...`、`make test`、`go test -race ./...` 均通过；prod smoke 覆盖 Lark case -> worker -> Gateway tools -> `/cases/{case_no}/ai-decisions`，无控制面 token 返回 401，带 token 返回 200 且包含 10 条决策日志。
