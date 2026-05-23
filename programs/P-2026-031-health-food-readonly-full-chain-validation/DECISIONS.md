# Decisions

## D1：真实下游优先

本轮不使用 `mock-health-food-readonly-adapter.py`。health-food 自身新增 readonly API，排障平台通过 HTTP connector 调用该服务。

## D2：本地测试数据属于真实数据库证据

本地 `meow_pas` 已存在真实 health-food 表和历史数据。本轮不制造 mock 返回；如需验证，直接引用实际表查询结果。

## D3：不存在 uid 不继续硬查

如果 profile 工具返回 `registered=false`，Agent 结论必须要求反馈方确认正确 uid，不能继续假设业务数据存在。
