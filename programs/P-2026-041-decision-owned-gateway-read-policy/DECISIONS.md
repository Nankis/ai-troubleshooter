# Decisions

## D1. Gateway 读取策略归属 Decision Engine

是否复用平台经验、是否需要实时证据、是否查询 Gateway、查询哪些工具，都属于 Decision Engine 的决策。

Agent Platform 可以做三件事：

- 入口接收消息、图片、case。
- 提供平台数据和 Gateway tool catalog 的快照给 Decision Engine。
- 执行 Verifier 通过的 tool plan，并写入审计/上下文 ledger。

Agent Platform 不允许根据关键词自行绕过或触发 Gateway 读取。
