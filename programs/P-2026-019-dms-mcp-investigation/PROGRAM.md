# P-2026-019 DMS MCP Investigation

## 背景

用户要求调研阿里云 DMS 是否有 CLI 或 MCP，能否帮助排障平台通过 Gateway 查询 DB 证据。

## 目标

- 确认 DMS MCP / CLI 的官方可用性。
- 明确哪些 DMS 能力可以进入排障平台，哪些必须禁止或二次封装。
- 补充 DMS MCP 接入文档和元数据 route 示例。
- 优化 MCP readonly adapter 的参数映射能力，降低后续接入官方 MCP 的胶水成本。

## 非目标

- 不在没有公司 DMS 凭证和生产库授权的情况下宣称完成真实 DMS 验收。
- 不直接开放 `executeScript`、`askDatabase` 或任何 DDL/DML 能力给 Agent。
- 不让决策层直接连接 DMS MCP。
