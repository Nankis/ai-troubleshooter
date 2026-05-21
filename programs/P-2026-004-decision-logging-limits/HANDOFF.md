# HANDOFF

## 当前状态

- 分支：`codex/decision-logging-limits`
- 当前任务：已完成，等待合并。
- 已完成：AI 决策日志、MySQL/memory 持久化、case 级 timeout、工具调用上限、工具失败上限、失败收敛、控制面查询接口和文档。
- 验证：`git diff --check`、`go vet ./...`、`make test`、`go test -race ./...`、prod smoke。
