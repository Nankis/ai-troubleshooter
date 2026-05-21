# Gateway 安全与鉴权边界

Investigation Gateway 是生产只读查询的安全边界。Agent、worker、Lark bot 都不应直接访问生产 DB、Redis、日志平台或业务服务，所有工具调用必须经过 Gateway。

## 已内置能力

### 1. HTTP 入口鉴权

生产环境开启：

```bash
GATEWAY_AUTH_ENABLED=true
GATEWAY_BEARER_TOKENS='business-troubleshooter-v1:replace-with-strong-token'
```

`GATEWAY_BEARER_TOKENS` 使用 `agent_id:token`，多个 agent 用英文逗号分隔：

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

默认策略是 deny，新增工具和新增 agent 必须显式授权。

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
GATEWAY_AGENT_QPS=5
GATEWAY_USER_QPS=2
GATEWAY_TOOL_QPS=10
```

超过限制返回 `429`。多实例部署时建议用公司 API Gateway、Envoy、Redis 或 service mesh 做分布式限流。

### 6. 审计与脱敏

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

工具返回前会统一脱敏手机号、邮箱、token、secret、身份证等敏感字段。

## 部署层必须补齐

这些能力建议放在平台基础设施或公司网关层：

- mTLS 或 service mesh 身份认证。
- 仅允许 orchestrator/worker 所在网段访问 Gateway。
- Secret Manager 注入 Gateway token，不写入 Git、镜像或明文配置。
- 统一 WAF/API Gateway 访问日志。
- 多实例分布式限流。
- audit sink 接入统一日志、SIEM 或安全审计平台。

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
