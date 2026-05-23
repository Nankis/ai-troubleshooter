# MCP Gateway Adapter

本系统支持通过 MCP 接入外部只读能力，但不允许决策层直接连接 MCP server。推荐链路：

```text
Decision Engine / Worker
  -> Investigation Gateway
  -> MCP readonly adapter
  -> allowlisted MCP server tools
```

这样 MCP 能力仍然经过 Gateway 的鉴权、scope、限流、timeout、审计和脱敏。

## 运行方式

本地 health-food MCP 实验：

```bash
MCP_ADAPTER_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
MCP_READONLY_ADAPTER_PORT=19085 \
MOCK_HEALTH_FOOD_SCENARIO=recommendation_missing \
python3.13 scripts/mcp-readonly-adapter.py
```

默认配置会启动 `scripts/mock-health-food-mcp-server.py` 作为 MCP stdio server，并暴露以下 readonly endpoints：

- `/v1/readonly/health-food/user/profile`
- `/v1/readonly/health-food/ai/quota`
- `/v1/readonly/health-food/meals/range`
- `/v1/readonly/health-food/recommendation/status`
- `/v1/readonly/ops/logs/search`
- `/v1/readonly/ops/cases/similar`

让平台通过 Gateway 调用该 adapter：

```bash
CONNECTOR_MODE=http \
CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
MARKET_READONLY_BASE_URL=http://127.0.0.1:19085 \
ASSET_READONLY_BASE_URL=http://127.0.0.1:19085 \
OPS_READONLY_BASE_URL=http://127.0.0.1:19085 \
HEALTH_FOOD_READONLY_BASE_URL=http://127.0.0.1:19085 \
go run ./cmd/dev-server
```

## 自定义 MCP server

通过 `MCP_ADAPTER_CONFIG_JSON` 指定 MCP server 和 route allowlist：

```json
{
  "server": {
    "command": ["python3.13", "scripts/mock-health-food-mcp-server.py"],
    "request_timeout_seconds": 5,
    "protocol_version": "2025-06-18"
  },
  "routes": [
    {
      "path": "/v1/readonly/health-food/recommendation/status",
      "tool_name": "health_food_recommendation_status",
      "source": "health-food-mcp/mock",
      "version": "mcp-mock-v1",
      "required_params": ["uid", "recommendation_date"],
      "forward_all_params": true,
      "param_map": {}
    }
  ]
}
```

要求：

- `routes` 是唯一暴露面，不会自动开放 MCP `tools/list` 的全部工具。
- 每个 route 必须映射到一个 MCP `tools/list` 中存在的 tool。
- `required_params` 使用 Gateway/readonly adapter 侧的标准字段名，adapter 会自动为 Go/Pascal/camel/kebab 参数补 snake_case alias。
- `param_map` 可把 Gateway 标准字段映射成 MCP tool schema 字段；`forward_all_params=false` 时只转发 `param_map` 和 `fixed_params` 中声明的字段，适合 DMS 这类严格 schema 的官方 MCP。
- MCP tool 应返回 `structuredContent` 对象；如果只返回 text，adapter 只接受可解析为 JSON object 的文本，否则会包装为 `{ "text": "..." }`。
- adapter 对外仍使用标准 readonly envelope，不暴露 MCP 原始协议细节。

阿里云 DMS MCP 接入方案见 [dms-mcp-integration.md](dms-mcp-integration.md)，元数据 route 示例见 [../configs/mcp-dms-adapter.metadata.example.json](../configs/mcp-dms-adapter.metadata.example.json)。

## 安全边界

- 不允许把写工具、文件读取、命令执行、任意 SQL、任意日志 dump 直接配置成 route。
- 生产环境必须设置 `MCP_ADAPTER_API_KEY`，并由 Gateway 用 `CONNECTOR_API_KEY` 访问。
- Gateway 仍然负责 agent 身份、scope、rate limit、参数边界、timeout、audit 和 masking。
- MCP server 的工具说明只能作为接入参考，最终是否允许暴露由 manifest/route allowlist 决定。

## 验收标准

MCP 接入不能只算“脚本能启动”。验收必须同时满足：

- MCP server 进程已启动并成功响应 `initialize`、`tools/list`。
- MCP readonly adapter `/healthz` 能返回 allowlisted route 和 MCP tools。
- `cmd/dev-server` 或 `cmd/investigation-gateway` 已启动。
- 通过 Gateway `POST /tools/{tool}/invoke` 成功调用 MCP tool。
- 返回结果包含业务预期字段，并且 Gateway summary 与 data 符合预期。
