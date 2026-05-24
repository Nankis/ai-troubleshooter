# RESULT

## 结论

已修正。

- 架构图从 `Platform -> Gateway` 改为 `Decision Engine -> Platform Tool Executor -> Gateway`。
- `Agent Platform` 不再判断 `_requires_realtime`；它只提供知识候选和 tool catalog 快照。
- `Decision Engine / Knowledge Agent` 统一判断是否需要实时证据，显式日期、查真实数据、网关/数据库/证据不足等信号都会阻止高置信经验短路。
- `Platform Tool Executor` 只执行 Verifier 通过的 tool plan。

## 验证

- `make test` 通过。
- `make secret-scan` 通过。
- `git diff --check` 通过。
