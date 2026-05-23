# ERRORS

## E1：MCP adapter 最初只识别 snake_case params

- 现象：Web Chat 调 `search_logs_by_service` 时，Go `LogQuery` 序列化为 `ServiceName`，adapter route required check 只认 `service_name`，误判缺字段。
- 修复：adapter 增加 `normalize_params`，为 PascalCase / camelCase / kebab-case 参数补 snake_case alias。
- 防复发：新增 `tests/test_mcp_readonly_adapter.py` 覆盖字段归一化。
