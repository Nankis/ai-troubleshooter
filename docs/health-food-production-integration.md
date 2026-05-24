# health-food 生产只读接入

本接入模式用于“本地运行排障平台，查询生产 health-food 的只读证据”。Decision Engine 和 Gateway 都在本地，Agent 只能通过 Gateway 调用受控 readonly adapter，不允许直连生产数据库。

## 当前可用链路

health-food 仓库已存在内部日志查询接口：

```text
GET /food-health/sys/admin/search-logs
```

该接口返回日志条目列表，支持 `date`、`type`、`traceId`、`uid`、`api`、`content`、`page`、`pageSize` 过滤。排障平台不直接暴露这个接口，而是使用本地 `scripts/real-health-food-readonly-adapter.py` 做桥接：

```text
Gateway search_logs_by_service
  -> local health-food readonly adapter
  -> production health-food admin log search
  -> masked LogSearchResult samples
```

## 本地启动

所有 token、密码、生产地址都只能通过环境变量传入，禁止写入仓库文件。

```bash
CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
HEALTH_FOOD_ADMIN_BASE_URL="https://health-food.example.com" \
HEALTH_FOOD_ADMIN_SECRET="$HEALTH_FOOD_ADMIN_SECRET" \
HEALTH_FOOD_ALLOWED_SERVICE_NAMES="health-food,food-health" \
HEALTH_FOOD_LOG_MAX_RANGE_MINUTES=30 \
HEALTH_FOOD_LOG_MAX_LIMIT=20 \
REAL_HEALTH_FOOD_ADAPTER_PORT=19084 \
python3.13 scripts/real-health-food-readonly-adapter.py
```

再启动本地平台：

```bash
CONNECTOR_MODE=http \
CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
OPS_READONLY_BASE_URL=http://127.0.0.1:19084 \
HEALTH_FOOD_READONLY_BASE_URL=http://127.0.0.1:19084 \
DB_DRIVER=mysql \
DB_DSN="$LOCAL_DB_DSN" \
GATEWAY_AUTH_ENABLED=true \
GATEWAY_AGENT_CONFIG_FILE=configs/gateway-agents.example.json \
BUSINESS_TROUBLESHOOTER_GATEWAY_TOKEN="$BUSINESS_TROUBLESHOOTER_GATEWAY_TOKEN" \
HEALTH_FOOD_AGENT_GATEWAY_TOKEN="$HEALTH_FOOD_AGENT_GATEWAY_TOKEN" \
HTTP_PORT=18088 \
make gateway
```

## 查询示例

```bash
curl -sS http://127.0.0.1:18088/tools/search_logs_by_service/invoke \
  -H "Authorization: Bearer $HEALTH_FOOD_AGENT_GATEWAY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "case_id": "case_health_food_prod_001",
    "agent_id": "health-food-readonly-agent",
    "caller_user_id": "local-debugger",
    "chat_id": "oc_health_food_oncall",
    "arguments": {
      "service_name": "health-food",
      "start_time": "2026-05-23T10:00:00+08:00",
      "end_time": "2026-05-23T10:10:00+08:00",
      "level": "error",
      "keyword": "generateDailyFoodRecommend",
      "limit": 5
    }
  }'
```

## 安全约束

- 生产日志密钥只放在 `HEALTH_FOOD_ADMIN_SECRET`，adapter 不把它返回给 Gateway。
- Gateway 侧仍然要求 Bearer、agent policy、scope、tool allowlist、QPS、time range 和 limit 校验。
- adapter 只允许 `HEALTH_FOOD_ALLOWED_SERVICE_NAMES` 内的服务名。
- 单次日志窗口默认最多 30 分钟，默认最多返回 20 条脱敏样例。
- 返回日志会脱敏 `password`、`token`、`secret`、`api_key`、Bearer token、邮箱和手机号，并截断长文本。
- 生产验收不能用 mock 代替：必须实际启动本地 Gateway/adapter，直接调用生产 health-food 日志接口，并查到符合问题时间窗的可靠证据。

## 需要公司提供

| 项 | 说明 |
| --- | --- |
| `HEALTH_FOOD_ADMIN_BASE_URL` | 生产 health-food 的 HTTPS base URL，不包含查询参数。 |
| `HEALTH_FOOD_ADMIN_SECRET` | 只读日志查询密钥，通过本地环境变量提供。 |
| 服务名 | 默认 `health-food`，若生产日志使用 `food-health` 等名字，需要加入 allowlist。 |
| 问题信息 | 至少提供时间窗、uid/trace_id/接口路径/关键词之一。 |

如果公司不希望使用 `password` 查询参数，应在 health-food 侧新增 Bearer 或内网签名版 readonly log endpoint，仍然保持同样的 Gateway adapter envelope：`POST /v1/readonly/ops/logs/search`。
