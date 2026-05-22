# RESULT

## 结果

已完成 AI 决策日志与查询限制：

- 新增 `tb_troubleshoot_ai_decision_log` DDL、Go model、memory store 和 MySQL store。
- Decision runner 记录分类、实体抽取、必要字段检查、工具计划、工具调用、停止查询、总结和失败原因。
- 新增 `GET /cases/{case_no}/ai-decisions?limit=100`。
- 新增 `MAX_INVESTIGATION_SECONDS`、`MAX_TOOL_CALLS_PER_CASE`、`MAX_TOOL_FAILURES_PER_CASE`。
- 超时或不可恢复错误会把 investigation/case 收敛到失败状态。

## 验证

- `git diff --check`
- `go vet ./...`
- `make test`
- `go test -race ./...`
- prod smoke：Lark case 触发完整工具查询后，可查到 10 条 AI 决策日志。

## 剩余边界

- 真实 LLM provider 接入后，应把模型原始 trace id、prompt version、completion id 写入 `tb_troubleshoot_ai_decision_log` 或扩展字段。
- 后续如引入多轮 ReAct，必须继续复用当前 timeout、tool call budget 和 failure budget。
