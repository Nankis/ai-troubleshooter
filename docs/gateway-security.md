# Gateway 安全与鉴权边界

Investigation Gateway 是业务生产只读查询的安全边界。Python Agent Platform、Decision Engine、Lark/飞书入口都不应直接访问生产 DB、Redis、日志平台或业务服务，所有工具调用必须经过 Gateway。平台自己的 case、审计、知识库 MySQL 不属于业务生产证据源，由 Agent 平台服务按内部权限访问。

业务服务通过 Gateway 注册只读能力，例如日志 MCP、行情、资产、风控和其他只读工具。MCP server 必须先通过 MCP readonly adapter 转成受控 readonly endpoint，不能让决策层直接连接。Gateway 校验 `agent / scope / case / user / tool / limit / timeout`，业务服务也应校验来自 Gateway 的内部身份，形成双层鉴权。Agent 隔离在 Gateway 之前，不直接对业务 DB。

## 已内置能力

### 1. HTTP 入口鉴权

生产环境开启：

```bash
GATEWAY_AUTH_ENABLED=true
GATEWAY_AGENT_CONFIG_FILE=configs/gateway-agents.example.json
BUSINESS_TROUBLESHOOTER_GATEWAY_TOKEN='replace-with-strong-token'
```

推荐使用 `GATEWAY_AGENT_CONFIG_FILE` 或 `GATEWAY_AGENT_CONFIG_JSON` 配置 agent、scope、tool 和 chat allowlist。agent 配置只写 `bearer_token_env`，真实 token 由环境变量或密钥系统注入：

```json
{
  "agents": [
    {
      "agent_id": "health-food-readonly-agent",
      "status": "enabled",
      "bearer_token_env": "HEALTH_FOOD_AGENT_GATEWAY_TOKEN",
      "allowed_scopes": ["health_food:user:read", "health_food:recommendation:read"],
      "allowed_tools": ["get_health_food_user_profile", "get_health_food_recommendation_status"],
      "allowed_chat_ids": ["oc_health_food_oncall"]
    }
  ]
}
```

兼容旧方式：`GATEWAY_BEARER_TOKENS` 使用 `agent_id:token`，多个 agent 用英文逗号分隔：

```bash
GATEWAY_BEARER_TOKENS='business-troubleshooter-v1:token-a,market-agent:token-b'
```

调用 Gateway：

```bash
curl -s localhost:8080/tools/get_asset_snapshot/invoke \
  -H 'Authorization: Bearer replace-with-strong-token' \
  -H 'Content-Type: application/json' \
  -d '{
    "case_id":"case_1",
    "agent_id":"business-troubleshooter-v1",
    "arguments":{"user_id":"user_123","asset_symbol":"USDT"}
  }'
```

### 2. 认证身份与 agent_id 绑定

Bearer token 认证成功后会解析出可信 `agent_id`。如果请求体传入其它 `agent_id`，Gateway 直接返回 `403`，不会进入工具 handler。

这样可以避免某个调用方拿自己的 token 冒用其它 agent 的授权 scope。

### 3. Policy 默认拒绝

工具执行前必须通过 policy：

- agent 已注册。
- tool 已启用。
- agent 拥有 tool 所需 scope。
- Lark 用户和群满足授权策略。
- 工具对应的业务能力已注册，并且只能执行只读 handler。

默认策略是 deny。生产环境新增工具和新增 agent 必须显式授权；本地内置默认 agent 为了支持 Web 动态只读能力接入，允许 `allowed_tools` 使用 `*`，但仍要求 tool 已发布、scope 允许、参数边界通过。

如果没有配置 `GATEWAY_AGENT_CONFIG_FILE` / `GATEWAY_AGENT_CONFIG_JSON`，系统会使用内置默认 agent，便于本地 demo。生产建议总是提供 agent 配置文件。`GATEWAY_AGENT_ID` 控制内置默认 agent id，默认 `business-troubleshooter-v1`。

### 4. 参数边界

每个工具可以声明：

- `MaxTimeRangeMinutes`
- `MaxLimit`
- `RequiredScope`
- 入参 schema

Gateway 会在执行 handler 前校验时间范围和 limit，防止 Agent 一次查询过大范围数据。

### 5. 网关级限流

当前内置单实例固定窗口限流：

```bash
GATEWAY_AGENT_QPS=20
GATEWAY_USER_QPS=10
GATEWAY_TOOL_QPS=20
```

默认值会覆盖一期单个 case 在同一秒内连续调用 5 个左右工具的 burst；超过限制返回 `429`。多实例部署时建议用公司 API Gateway、Envoy、Redis 或 service mesh 做分布式限流。

### 6. 网关级超时

Gateway 会为每次工具 handler 调用创建超时上下文。超过 `DEFAULT_TOOL_TIMEOUT_SECONDS` 后，HTTP 入口返回 `504 Gateway Timeout`，本次调用写入工具审计，不再继续等待下游服务。

### 7. 控制面 API 鉴权

root cause 回填、feedback、knowledge 查询、case/process 这类控制面接口不走 Lark 入口信任，生产环境必须启用内部 Bearer 鉴权：

```bash
CONTROL_API_AUTH_ENABLED=true
CONTROL_API_BEARER_TOKENS='replace-with-internal-control-token'
```

受保护接口包括：

- `GET /cases/{case_no}`
- `POST /cases/{case_no}/root-cause`
- `GET /cases/{case_no}/root-cause`
- `POST /cases/{case_no}/feedback`
- `GET /cases/{case_no}/feedback`
- `GET /cases/{case_no}/evolution-runs`
- `GET /knowledge`
- Python Agent Platform 的 case、knowledge 和控制面 API

### 8. 生产配置 fail-closed

`APP_ENV=prod` 时，以下配置缺失会启动失败：

- `GATEWAY_AUTH_ENABLED=true`
- `GATEWAY_BEARER_TOKENS` 或 agent 配置中的 `bearer_token_env`
- `CONTROL_API_AUTH_ENABLED=true`
- `CONTROL_API_BEARER_TOKENS`
- `LARK_VERIFICATION_TOKEN`
- `LARK_ALLOWED_CHAT_IDS`

其中 `LARK_*` 只对暴露 Lark 入口的服务强制校验；Gateway 与控制面 API 分别按服务入口校验。

### 9. 审计与脱敏

每次工具调用都会记录：

- case id
- agent id
- Lark user id
- chat id
- tool name
- policy decision
- deny reason
- query id
- latency
- error

`DB_DRIVER=mysql` 时必须配置 `DB_DSN`，Gateway 会把审计写入 MySQL `tb_troubleshoot_tool_call_audit`；调用人字段使用 `caller_user_id VARCHAR(128)`，兼容 Lark/飞书和公司自定义字符串 UID。只有显式 `DB_DRIVER=memory` 才使用内存审计，适合一次性本地 smoke。工具返回前会统一脱敏手机号、邮箱、token、secret、身份证等敏感字段。

## 部署层必须补齐

这些能力建议放在平台基础设施或公司网关层：

- mTLS 或 service mesh 身份认证。
- 仅允许 Agent Platform / Decision Engine 所在网段访问 Gateway。
- Secret Manager 注入 Gateway token，不写入 Git、镜像或明文配置。
- 控制面 token 与 Gateway token 分开管理，root cause 回填入口只给可信内部系统或业务 owner。
- 统一 WAF/API Gateway 访问日志。
- 多实例分布式限流。
- MySQL tool audit 可继续同步到统一日志、SIEM 或安全审计平台。

## 验收示例

未带 token：

```bash
curl -i localhost:8080/tools/get_asset_snapshot/invoke \
  -H 'Content-Type: application/json' \
  -d '{"arguments":{}}'
```

预期：`401 Unauthorized`。

token 对应 agent 与请求体不一致：

```bash
curl -i localhost:8080/tools/get_asset_snapshot/invoke \
  -H 'Authorization: Bearer replace-with-strong-token' \
  -H 'Content-Type: application/json' \
  -d '{
    "case_id":"case_1",
    "agent_id":"other-agent",
    "arguments":{"user_id":"user_123","asset_symbol":"USDT"}
  }'
```

预期：`403 Forbidden`。
