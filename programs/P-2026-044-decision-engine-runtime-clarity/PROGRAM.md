# P-2026-044 Decision Engine Runtime Clarity

## 背景

用户指出文档 `Decision Engine 是否单独启动` 容易让人误解为 Web Chat 排查没有使用 Decision Engine。Decision Engine 是排障系统核心，文档和测试必须明确：正常 Web Chat 不单独启动一个外部 Decision Engine 服务，但 Agent Platform 必须在进程内调用 Python `DecisionEngine.plan()`。

## 目标

- 澄清 Agent Platform 与 Decision Engine 的运行关系。
- 增加测试，证明 Web Chat 进入排查时会调用 `decision_engine.plan()`。
- 明确之前本地验证中 mock 的范围：可以 mock Gateway/LLM provider，但不能 mock 掉 Decision Engine 主流程并宣称排查验证。

## 非目标

- 本轮不拆分 Decision Engine 为独立生产服务。
- 本轮不调整 Agent Team 决策算法。
