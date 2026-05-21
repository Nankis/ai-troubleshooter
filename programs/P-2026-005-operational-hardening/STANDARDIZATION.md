# STANDARDIZATION

## 新增平台约束

- 所有外部事件入口必须提供稳定幂等键；Lark 使用 `source + message_id`。
- Worker 处理 case 前必须先认领状态，重复事件只能跳过并留审计。
- 处理中状态必须有陈旧窗口，避免 worker 崩溃后永久卡住。
- AI 决策日志可用于复盘，但快照不能保存敏感明文。
- 数据库唯一索引是入口幂等的最后兜底，不依赖应用层“先查再写”作为唯一保护。

## 后续复用

新增事件源时复用同一模式：

1. 在 `CreateCaseInput` 中填入 `Source` 和 `MessageID`。
2. 入口 handler 先调用 `FindCaseByMessageID`。
3. 重复事件返回已有 `case_no`，不发布新队列事件。
4. Orchestrator 依赖 `process_skipped` 保护重复 worker。
