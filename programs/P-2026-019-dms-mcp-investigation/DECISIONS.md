# DECISIONS

## D1：DMS MCP 可以接，但不能直连决策层

DMS MCP 包含元数据、SQL 执行、NL2SQL 和工单相关工具。它必须先通过 MCP readonly adapter 映射成 allowlisted readonly endpoints，再交给 Gateway。

## D2：第一阶段只开放 DB 元数据

实例、库、表和表结构可以帮助 Agent 判断证据范围，风险可控。`executeScript`、`askDatabase`、工单审批和 addInstance 不直接开放。

## D3：SQL 查询必须走 named readonly query

生产排障需要查 DB 时，Agent 只能选择平台预注册的 `query_id` 和参数，adapter 内部映射到固定 SQL 模板，再通过 DMS 或 OpenAPI SDK 执行。Agent 不传 SQL 文本。

## D4：适配官方 MCP 需要参数映射

Gateway 规范倾向 snake_case，官方 MCP tool schema 不一定完全一致。MCP adapter 增加 `param_map`、`fixed_params` 和 `forward_all_params`，避免每接一个 MCP 都写专用脚本。
