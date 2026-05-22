# P-2026-007 User Drawn Architecture Flow

## 背景

用户提供了新的手绘架构图，指出当前设计图仍然有边界问题。新的图强调：

- Agent 问题排查平台负责入口、case、状态机、决策、平台经验、LLM 配置和知识沉淀。
- Investigation Gateway 是独立的业务只读能力门禁，不查询平台自己的 MySQL。
- 业务服务注册只读能力并访问自己的 DB，Agent 隔离，不直接对 DB。
- 决策层先判断信息是否足够；信息足够后先查平台经验并评分，经验不足或需要实时证据时再查 Gateway。
- 高置信经验可以直接返回，但必须记录为什么这样判断。

## 目标

- 按用户手绘图和后续讨论，完善 README 的一期部署架构图。
- 完善 README 的单 case 排障流程图。
- 同步 ADR 中的长期目标边界。
- 补充 Gateway 安全文档和一期原则，明确能力注册、双层鉴权和 Agent 隔离 DB。

## 非目标

- 不实现新的运行时代码。
- 不改旧 Program 历史记录。
- 不调整数据库 schema 或 OpenAPI。

## 验收标准

- README 架构图体现三段边界：Agent 平台、Investigation Gateway、业务服务/业务 DB。
- README 流程图体现：信息不足追问、平台经验评分、高置信直接返回、低置信查 Gateway、统一回复出口。
- ADR 明确 Gateway 不查平台 MySQL，业务能力注册到 Gateway，业务服务自己访问业务 DB。
- Gateway 安全文档明确双层鉴权和 Agent 隔离。
- `git diff --check` 通过。
