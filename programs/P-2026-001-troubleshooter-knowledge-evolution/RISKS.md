# RISKS

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| 本机没有 MySQL 测试库 | 无法实际执行 migration | 单元测试覆盖 store 接口外的自进化逻辑；真实 MySQL migration 标记为部署前检查项 |
| 多进程部署仍使用内存 queue | lark-bot/worker 分离时事件无法共享 | 本轮明确为非目标；后续 Program 接 Redis Stream |
| root cause 枚举不稳定 | 知识条目聚合分散 | 文档提供建议枚举，业务方接入时先确认 |
