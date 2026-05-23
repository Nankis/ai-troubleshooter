# Alibaba Cloud DMS MCP / CLI Integration

调研日期：2026-05-23。

## 结论

阿里云 DMS 可以接入。

- DMS 有官方 MCP Server：`alibabacloud-dms-mcp-server`，可用 `uvx alibabacloud-dms-mcp-server@latest` 本地启动，也可以在 DMS 控制台启用托管 MCP 服务。
- DMS OpenAPI 支持 Alibaba Cloud SDK 和 Alibaba Cloud CLI；CLI 产品名是 `dms-enterprise`，例如 `aliyun dms-enterprise --help`。
- 对我们这套排障平台，推荐顺序是：DMS MCP 做快速接入和元数据查询；DMS OpenAPI/SDK 做生产级 named readonly query adapter；CLI 只适合人工诊断或临时脚本，不建议作为 Gateway 的常驻调用方式。

参考来源：

- [阿里云 DMS MCP 官方文档](https://www.alibabacloud.com/help/en/dms/use-cases/deploy-dms-mcp)
- [阿里云 DMS OpenAPI 调用方式](https://www.alibabacloud.com/help/en/dms/using-openapi)
- [阿里云 DMS CLI 集成示例](https://www.alibabacloud.com/help/en/dms/developer-reference/cli-integration-example-dms)
- [aliyun/alibabacloud-dms-mcp-server](https://github.com/aliyun/alibabacloud-dms-mcp-server)
- [DMS MCP tool list](https://raw.githubusercontent.com/aliyun/alibabacloud-dms-mcp-server/main/doc/Tool-List-en.md)

## 推荐链路

```text
Decision Engine / Worker
  -> Investigation Gateway
  -> MCP readonly adapter
  -> Alibaba Cloud DMS MCP Server
  -> DMS managed database scope
```

决策层仍然不能直连 DMS MCP。DMS 能力必须先被 `scripts/mcp-readonly-adapter.py` 映射成受控 readonly endpoint，再经过 Gateway 的鉴权、scope、限流、timeout、审计和脱敏。

## 能力分层

第一阶段先开放元数据能力：

- `listInstances`：搜索 DMS 实例。
- `searchDatabase`：按 schema name 搜索数据库。
- `getDatabase`：按 host / port / schema 获取数据库详情。
- `listTables`：按 database_id 查询表。
- `getTableDetailInfo`：按 table_guid 查询表结构。

暂不直接开放这些能力：

- `executeScript`：可执行 SQL，必须再包一层 named readonly query。
- `askDatabase`：NL2SQL + 执行 SQL，必须先限制库表范围、limit、timeout 和敏感字段。
- `createDataChangeOrder`、`submitOrderApproval`、`approveOrder`：写入或流程变更能力，不能进入排障 Agent 工具集。
- `addInstance`：可能携带数据库账号密码，不能让 Agent 直接调用。

## 本仓库接入方式

配置示例见 [configs/mcp-dms-adapter.metadata.example.json](../configs/mcp-dms-adapter.metadata.example.json)。

启动 DMS MCP readonly adapter 时，DMS 凭证只通过环境变量传入，不写入仓库：

```bash
export ALIBABA_CLOUD_ACCESS_KEY_ID="replace-at-runtime"
export ALIBABA_CLOUD_ACCESS_KEY_SECRET="replace-at-runtime"
export ALIBABA_CLOUD_SECURITY_TOKEN=""
export ALIBABA_CLOUD_DMS_ENDPOINT="dms-enterprise.cn-hangzhou.aliyuncs.com"

export MCP_ADAPTER_API_KEY="$LOCAL_CONNECTOR_API_KEY"
export MCP_READONLY_ADAPTER_PORT=19085
export MCP_ADAPTER_CONFIG_JSON="$(python3.13 -c 'import json,sys; print(json.dumps(json.load(open(sys.argv[1]))))' configs/mcp-dms-adapter.metadata.example.json)"
python3.13 scripts/mcp-readonly-adapter.py
```

`/healthz` 必须能看到 allowlisted routes 和实际 DMS MCP tools。注意 DMS 文档和不同版本包里，`listTable` / `listTables` 命名可能不一致，最终以 `tools/list` 返回为准；当前示例按 GitHub/PyPI 最新包使用 `listTables`。

## Gateway 使用边界

当前 Gateway 的生产工具是静态注册的 K线、资产、日志、health-food 和相似案例工具。DMS adapter 已经可以作为 readonly HTTP adapter 启动，但要让决策层通过 Gateway 正式调用 DB 元数据，需要再做一个小的 Gateway 侧能力：

- 新增 `DBConnector`，读取 `DB_READONLY_BASE_URL`。
- 注册 `search_db_instances`、`search_db_databases`、`list_db_tables`、`get_db_table_detail`。
- 新增 scope：`db:metadata:read`。
- 将这些工具加入 `configs/gateway-agents.example.json` 的受控 agent。

SQL 查询能力不要直接暴露为 `executeScript`。生产方案应该是：

```text
Gateway tool: run_named_db_query
  -> readonly adapter: /v1/readonly/db/query/named
  -> adapter allowlist: query_id -> fixed SQL template
  -> DMS executeScript or DMS OpenAPI SDK
```

标准参数建议：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `query_id` | string | 平台预注册查询 ID，例如 `health_food_user_ai_quota_by_uid`。 |
| `database_id` | string | DMS database ID；生产可由 adapter 根据 `service_name` 映射，不一定暴露给 Agent。 |
| `params` | object | 查询模板参数，只允许白名单字段。 |
| `limit` | int | 默认 50，最大 100。 |
| `timeout_ms` | int | 默认 3000，最大 5000。 |
| `reason` | string | 本次查询原因，用于审计。 |

## 安全要求

- RAM 用户或 STS 令牌必须按最小权限授权；官方快速开始提到 `AliyunDMSFullAccess`，公司级接入应优先收敛到只读、指定实例、指定库表范围。
- DMS 实例必须先加入 DMS 并启用安全托管，避免在 adapter 保存数据库账号密码。
- Gateway 和 MCP adapter 都必须启用 Bearer token。
- adapter 不允许返回完整表 dump，默认 limit <= 50，最大 limit <= 100。
- 所有 SQL 执行类能力必须记录 `case_id`、`agent_id`、`query_id`、`database_id`、模板 ID、耗时、行数和脱敏状态。
- 生产接入前必须跑负向验证：无 token、越权 scope、未知 route、非 allowlisted query_id、超 limit、超时、DML/DDL SQL。

## 验收标准

真实验收不能只证明脚本启动。

- DMS MCP server 实际启动，`initialize` / `tools/list` 成功。
- MCP readonly adapter `/healthz` 返回 DMS tools 和 allowlisted routes。
- Gateway 实际启动，且通过 Bearer 调用 DB 元数据工具成功。
- 至少查到一个真实 DMS 实例、数据库、表和表结构。
- 如果开启 named query，必须查到真实只读数据，并验证 DML/DDL 被拒绝。
- Gateway tool audit 和 adapter 日志都能按 `request_id` / `case_id` 关联到本次调用。
