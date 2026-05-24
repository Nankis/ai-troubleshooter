# RISKS

| 风险 | 影响 | 处理 |
| --- | --- | --- |
| Context Ledger 表未迁移 | Agent Platform 处理 case 时写 ledger 失败 | 部署前必须执行 `make migrate-mysql`；本轮验证记录 migration。 |
| 压缩摘要丢失细节 | LLM 可能看不到某些原始字段 | observation 保留 `data_shape/result_count/evidence_refs`，需要细节时通过引用回查。 |
| 旧审计日志仍保存脱敏原始返回 | 可能产生较大日志体积 | 本轮不改历史审计策略；后续可加审计 payload 截断或对象存储。 |
| 本轮未引入异步多 Agent worker | 并行排查速度未提升 | 先完成上下文隔离；后续可在 ledger 基础上做 specialist 并行和结果汇总。 |
