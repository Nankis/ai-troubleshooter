# P-2026-005 Operational Hardening

## 背景

继续从生产事故视角审查一期框架：Lark 事件可能重复投递，worker 可能重复消费，同一 case 可能被人工或任务重复触发；AI 决策日志虽然可审计，但快照不能落敏感明文。

## 目标

- Lark 事件按 `source + message_id` 幂等去重。
- MySQL 通过唯一索引兜住并发重复创建。
- Decision runner 处理前先认领 case，重复触发只记录跳过原因，不再打下游。
- 陈旧处理中状态能恢复或失败收敛，避免 worker 崩溃后永久卡住。
- AI 决策日志快照写入前统一脱敏。
- 补齐文档、迁移和单测。

## 非目标

- 不引入新的队列基础设施；Redis Stream 后续单独做。
- 不实现跨实例分布式限流；本期聚焦 case 幂等和重复处理保护。
- 不改变业务只读 adapter 接口契约。

## 验收标准

- 同一 Lark `message_id` 重放只发布一次 case event。
- 同一 case 二次 `ProcessCase` 不再触发工具调用。
- 陈旧 `READY_TO_INVESTIGATE` 可重新认领，陈旧 `INVESTIGATING` / `WAITING_TOOL_RESULT` 会失败收敛。
- `tb_troubleshoot_ai_decision_log` 快照不包含手机号、token、api key 明文。
- `make test`、`go vet ./...`、`go test -race ./...` 通过。
