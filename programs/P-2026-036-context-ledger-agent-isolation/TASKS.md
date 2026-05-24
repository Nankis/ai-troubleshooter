# TASKS

| Task | 状态 | 说明 |
| --- | --- | --- |
| T1 | done | 建立 Program，确认上下文隔离目标和边界。 |
| T2 | done | 新增 Context Ledger DDL 和 repository 方法。 |
| T3 | done | Agent Platform 写入压缩 ledger，并阻止原始工具 `data` 进入 LLM summarize。 |
| T4 | done | Decision Engine 请求模型支持 context ledger，Supervisor/Verifier 显式记录短上下文策略。 |
| T5 | done | 增加单测覆盖 ledger 写入、证据引用和 LLM 输入压缩。 |
| T6 | done | 执行全量测试、secret scan、diff check 和 MySQL 本地验证。 |
| T7 | pending | 更新最终证据、结果、HANDOFF，commit + push。 |
