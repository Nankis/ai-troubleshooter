# RISKS

- DMS 文档和包版本的 tool name 可能存在差异，例如 `listTable` / `listTables`，必须以实际 `tools/list` 为准。
- 官方快速开始常用 `AliyunDMSFullAccess`，公司级接入需要进一步收敛到最小权限。
- `executeScript` 和 `askDatabase` 即使 DMS 有审计，也仍可能造成大批量数据返回或误查敏感字段，必须二次封装。
- 本轮没有真实公司 DMS 凭证，不能宣称生产链路验收通过。
