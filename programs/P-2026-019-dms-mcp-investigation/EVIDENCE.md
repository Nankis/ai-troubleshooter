# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 结论 |
| --- | --- | --- | --- |
| EV-T1-001 | research | T1 | passed：确认 DMS 有官方 MCP Server、OpenAPI 和 CLI |
| EV-T2-001 | documentation | T2 | passed：新增 DMS 接入文档和 metadata route 示例 |
| EV-T3-001 | implementation | T3 | passed：MCP adapter 支持 param_map / fixed_params / forward_all_params |
| EV-T4-001 | command | T4 | passed：本地测试和安全扫描通过 |

## 调研来源

| Evidence ID | 来源 | 结论 |
| --- | --- | --- |
| EV-T1-001 | 阿里云 DMS MCP 官方文档 | DMS MCP 支持本地 PyPI 启动和 DMS 托管 MCP，支持多实例和单库模式。 |
| EV-T1-002 | 阿里云 DMS OpenAPI 文档 | DMS OpenAPI 支持 SDK 和 Alibaba Cloud CLI。 |
| EV-T1-003 | aliyun/alibabacloud-dms-mcp-server | 当前包暴露 `listInstances`、`searchDatabase`、`getDatabase`、`listTables`、`getTableDetailInfo`、`executeScript`、`askDatabase` 等工具。 |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-23 | `python3.13 -m unittest tests/test_mcp_readonly_adapter.py` | passed | 覆盖参数归一化和 route 参数映射 |
| EV-T4-002 | 2026-05-23 | `python3.13 -m py_compile scripts/mcp-readonly-adapter.py` | passed | adapter 语法检查通过 |
| EV-T4-003 | 2026-05-23 | `git diff --check` | passed | 无 whitespace 错误 |
| EV-T4-004 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | passed | Secret scan passed |
| EV-T4-005 | 2026-05-23 | `make test` | passed | Go 全量测试、decision-engine 14 个单测、repo tests 3 个单测通过 |

## 未验证项

- 未连接真实公司 DMS 实例，因为当前会话没有 RAM/STS 凭证、DMS tenant 和授权库表范围。
- 未通过 Gateway 调用 DMS metadata tool，因为本轮范围没有新增 Gateway `DBConnector` 和 DB tool registry。
