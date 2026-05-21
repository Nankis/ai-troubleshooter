# RESULT

## 结果

- Lark 事件入口支持 `source + message_id` 幂等去重。
- MySQL 增加 case 和 Lark message 幂等唯一索引。
- Orchestrator 只允许入口状态开始处理，重复触发写 `process_skipped` 并跳过。
- 陈旧 `READY_TO_INVESTIGATE` 会重新认领，陈旧 `INVESTIGATING` / `WAITING_TOOL_RESULT` 会失败收敛。
- AI 决策日志快照入库前统一脱敏。
- README、部署检查清单、决策日志文档已更新。

## 验证

- `git diff --check`
- `make test`
- `go vet ./...`
- `go test -race ./...`
