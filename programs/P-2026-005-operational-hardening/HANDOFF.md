# HANDOFF

## 当前状态

代码、迁移、文档和单测已补齐，最终验证已通过，等待合并推送。

## 注意事项

- 生产库执行 `004_case_idempotency.sql` 前先检查历史重复数据。
- 如果后续接 Redis Stream，仍需保留 Orchestrator 侧 `process_skipped`，队列的 at-least-once 投递不能替代业务幂等。
