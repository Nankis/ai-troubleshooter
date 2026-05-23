# HANDOFF

## 当前状态

- 已确认阿里云 DMS 存在官方 MCP Server 和 CLI。
- 已新增 DMS MCP 接入文档和 metadata allowlist 配置。
- MCP readonly adapter 已支持参数映射，后续接 DMS 这类官方 MCP 更顺。

## 下一步

- 在拿到公司 DMS RAM/STS 凭证、DMS endpoint、实例范围和只读库表清单后，实际启动 DMS MCP adapter。
- 新开 Gateway DB evidence Program，新增 `DBConnector` 和 DB 元数据工具。
- 如需查真实业务数据，先实现 `/v1/readonly/db/query/named`，只允许 query_id 模板，不允许 Agent 传 SQL。
