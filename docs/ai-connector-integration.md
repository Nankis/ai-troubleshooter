# AI 接入规范：业务只读接口封装

本文档给后续负责“接公司内部接口”的 AI 或工程师使用。目标是：不用改 Agent 主流程，只要把公司已有的业务只读接口封装成本文档定义的标准 HTTP API，`ai-troubleshooter` 就可以通过 `CONNECTOR_MODE=http` 直接调用。

## 一句话任务

你要为公司现有系统写一层只读 adapter。adapter 对外暴露本文档定义的接口命名、请求 envelope、参数和返回格式；adapter 内部可以调用公司已有 Go/Java 服务、日志平台、Redis 只读查询、DB read replica 或外部交易所 API。

不要让 Agent 传 SQL。不要实现写操作。不要返回未脱敏敏感字段。

## 部署配置

`ai-troubleshooter` 侧只需要配置：

```bash
CONNECTOR_MODE=http
CONNECTOR_API_KEY=replace-with-internal-token
CONNECTOR_TIMEOUT_SECONDS=5
MARKET_READONLY_BASE_URL=https://market-readonly.example.internal
ASSET_READONLY_BASE_URL=https://asset-readonly.example.internal
OPS_READONLY_BASE_URL=https://ops-readonly.example.internal
```

也可以把三个 base URL 指向同一个 adapter 服务。

## 通用约定

### 协议

- HTTP JSON。
- 所有接口必须是 `POST`。
- 所有接口必须只读。
- 所有时间字段使用 RFC3339，例如 `2026-05-21T20:00:00+08:00`。
- 金额和数量建议使用字符串，避免浮点精度问题，例如 `"1024.12000000"`。
- adapter 必须设置底层查询 timeout，不能无限等待。
- adapter 必须限制返回条数，不能返回大批量原始日志。

### 请求头

`ai-troubleshooter` 会发送：

```text
Authorization: Bearer ${CONNECTOR_API_KEY}
Content-Type: application/json
X-Request-Id: req_xxx
X-Case-Id: case_20260521_000001
X-Agent-Id: business-troubleshooter-v1
X-Caller-User-Id: ou_xxx
X-Tool-Name: get_asset_snapshot
```

adapter 必须至少记录 `X-Request-Id`、`X-Case-Id`、`X-Agent-Id`、`X-Tool-Name` 到自身日志。

### 通用请求 Envelope

所有接口请求体统一：

```json
{
  "request_id": "req_xxx",
  "case_id": "case_20260521_000001",
  "agent_id": "business-troubleshooter-v1",
  "caller_user_id": "ou_xxx",
  "tool_name": "get_internal_kline",
  "timeout_ms": 5000,
  "params": {}
}
```

`params` 是每个接口自己的业务参数。

### 通用成功响应 Envelope

所有接口成功响应体统一：

```json
{
  "request_id": "req_xxx",
  "source": "asset-service",
  "queried_at": "2026-05-21T20:00:03+08:00",
  "data_updated_at": "2026-05-21T20:00:00+08:00",
  "version": "v1",
  "data": {},
  "warnings": []
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `request_id` | 是 | 原样返回请求中的 request_id。 |
| `source` | 是 | 数据来源，例如 `asset-service`、`market-service`、`log-platform`。 |
| `queried_at` | 是 | adapter 实际查询时间。 |
| `data_updated_at` | 是 | 底层数据最后更新时间；如果无精确值，填查询时间并在 warnings 说明。 |
| `version` | 是 | adapter 或底层接口版本。 |
| `data` | 是 | 业务数据。 |
| `warnings` | 是 | 非致命告警，例如数据延迟、降级、部分字段不可得。 |

### 通用错误响应

HTTP 状态码使用：

- `400`：参数错误。
- `401`：鉴权失败。
- `403`：无权限。
- `408`：超时。
- `429`：限流。
- `500`：内部错误。
- `503`：底层服务不可用。

错误响应体：

```json
{
  "code": "TIME_RANGE_TOO_LARGE",
  "error": "time range exceeds 30 minutes"
}
```

## 接口清单

### 1. 查询我方 K线

```text
POST /v1/readonly/market/kline/internal
tool_name: get_internal_kline
```

用途：查询我方系统在指定币对、周期、时间范围内生成的 K线结果。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `symbol` | string | 是 | 币对，例如 `BTCUSDT`。 |
| `interval` | string | 是 | K线周期，例如 `1m`、`5m`、`1h`、`1d`。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |

限制：

- 默认最大时间范围 30 分钟。
- candles 最大 500 条。

`data`：

```json
{
  "candles": [
    {
      "open_time": "2026-05-21T20:00:00+08:00",
      "open": 65000.12,
      "high": 65120.12,
      "low": 64980.01,
      "close": 65012.40,
      "volume": 123.45
    }
  ]
}
```

### 2. 对比外部交易所 K线

```text
POST /v1/readonly/market/kline/compare
tool_name: get_external_kline_compare
```

用途：对比我方 K线和外部交易所 K线，返回差异摘要。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `symbol` | string | 是 | 币对，例如 `BTCUSDT`。 |
| `interval` | string | 是 | K线周期。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |
| `exchange` | string | 否 | `binance`、`okx`、`bybit`，默认 `binance`。 |

`data`：

```json
{
  "internal_candles_summary": "10 candles, latest close 65012.40",
  "external_candles_summary": "10 candles from binance, latest close 65129.42",
  "max_price_diff_ratio": 0.0018,
  "max_volume_diff_ratio": 0.012,
  "abnormal_points": [
    {
      "time": "2026-05-21T20:03:00+08:00",
      "field": "high",
      "internal_value": 65120.12,
      "external_value": 65237.33,
      "diff_ratio": 0.0018,
      "evidence_source": "binance"
    }
  ],
  "consistency_note": "20:03 high differs from binance by 0.18%"
}
```

### 3. 查询 K线缓存状态

```text
POST /v1/readonly/market/kline/cache-status
tool_name: get_kline_cache_status
```

用途：查询 K线缓存 key 是否存在、生成时间、TTL 和数据版本。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `symbol` | string | 是 | 币对。 |
| `interval` | string | 是 | K线周期。 |
| `time_bucket` | string | 是 | K线所在时间桶，RFC3339。 |

`data`：

```json
{
  "cache_key": "kline:BTCUSDT:1m:202605212000",
  "exists": true,
  "generated_at": "2026-05-21T20:01:10+08:00",
  "ttl": 180,
  "version": "v1",
  "data_updated_at": "2026-05-21T20:00:58+08:00"
}
```

### 4. 查询行情源状态

```text
POST /v1/readonly/market/source/status
tool_name: get_market_source_status
```

用途：查询某币对在指定时间窗内的行情源延迟、重连和数据缺口事件。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `symbol` | string | 是 | 币对。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |

`data`：

```json
{
  "source_name": "market-source-binance",
  "delay_events": [
    {
      "start_time": "2026-05-21T20:02:00+08:00",
      "end_time": "2026-05-21T20:04:00+08:00",
      "delay_ms": 1800
    }
  ],
  "reconnect_events": [],
  "data_gap_events": []
}
```

### 5. 查询资产快照

```text
POST /v1/readonly/asset/snapshot
tool_name: get_asset_snapshot
```

用途：查询用户在指定时间附近的资产快照。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_id` | string | 条件必填 | 与 `account_id` 二选一。 |
| `account_id` | string | 条件必填 | 与 `user_id` 二选一。 |
| `asset_symbol` | string | 是 | 资产币种，例如 `USDT`。 |
| `at_time` | string | 否 | RFC3339；为空时查当前快照。 |

`data`：

```json
{
  "available_balance": "1024.12000000",
  "frozen_balance": "18.50000000",
  "total_balance": "1042.62000000",
  "updated_at": "2026-05-21T19:59:58+08:00",
  "source": "asset-service",
  "version": "v1"
}
```

### 6. 查询资产事件流

```text
POST /v1/readonly/asset/events
tool_name: get_asset_events
```

用途：查询用户资产变更事件，用于解释余额变化、冻结异常、流水缺失。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_id` | string | 条件必填 | 与 `account_id` 二选一。 |
| `account_id` | string | 条件必填 | 与 `user_id` 二选一。 |
| `asset_symbol` | string | 是 | 资产币种。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |
| `event_types` | array[string] | 否 | 事件类型过滤。 |

限制：

- 默认最大时间范围 120 分钟。
- events 最大 100 条。

`data`：

```json
{
  "events": [
    {
      "event_id": "evt_001",
      "event_type": "trade_freeze",
      "delta": "-18.50000000",
      "balance_after": "1024.12000000",
      "occurred_at": "2026-05-21T20:05:00+08:00",
      "reference_id": "order_001"
    }
  ],
  "balance_before": "1042.62000000",
  "balance_after": "1024.12000000",
  "data_updated_at": "2026-05-21T20:05:03+08:00"
}
```

### 7. 查询用户近期错误

```text
POST /v1/readonly/asset/user-errors
tool_name: get_user_recent_errors
```

用途：查询与用户相关的资产、订单等服务错误摘要。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_id` | string | 条件必填 | 与 `account_id` 二选一。 |
| `account_id` | string | 条件必填 | 与 `user_id` 二选一。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |
| `service_names` | array[string] | 否 | 服务名过滤，例如 `asset-service`、`order-service`。 |

`data`：

```json
{
  "service_names": ["asset-service", "order-service"],
  "errors": [
    {
      "time": "2026-05-21T20:05:00+08:00",
      "service": "asset-service",
      "level": "warn",
      "message": "balance view refresh lag detected",
      "trace_id": "trace_xxx"
    }
  ]
}
```

### 8. 查询服务日志摘要

```text
POST /v1/readonly/ops/logs/search
tool_name: search_logs_by_service
```

用途：按服务、时间、关键词查询日志摘要。只能返回摘要和少量样例，不返回大批量原始日志。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `service_name` | string | 是 | 服务名。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |
| `level` | string | 否 | `error`、`warn`、`info`。 |
| `keyword` | string | 否 | 关键词。 |
| `trace_id` | string | 否 | trace id。 |
| `limit` | integer | 否 | 默认 50，最大 100。 |

`data`：

```json
{
  "service_name": "market-service",
  "total": 2,
  "samples": [
    {
      "time": "2026-05-21T20:03:00+08:00",
      "level": "warn",
      "service": "market-service",
      "message": "kline aggregation delayed",
      "trace_id": "trace_xxx"
    }
  ]
}
```

### 9. 查询近期发布

```text
POST /v1/readonly/ops/deployments/recent
tool_name: get_recent_deployments
```

用途：查询某服务在指定时间窗内的发布、配置、灰度记录。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `service_name` | string | 是 | 服务名。 |
| `start_time` | string | 是 | RFC3339。 |
| `end_time` | string | 是 | RFC3339。 |

`data`：

```json
{
  "service_name": "market-service",
  "items": [
    {
      "time": "2026-05-21T19:50:00+08:00",
      "version": "20260521.1",
      "operator": "release-bot",
      "change_note": "kline aggregation config update"
    }
  ]
}
```

### 10. 查询历史相似 case

```text
POST /v1/readonly/ops/cases/similar
tool_name: get_similar_cases
```

用途：查询历史相似问题。adapter 可以先用 SQL LIKE、标签检索或简单规则；后续再接向量库。

`params`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `issue_domain` | string | 是 | `kline` 或 `asset`。 |
| `issue_type` | string | 否 | 问题类型。 |
| `text` | string | 否 | 原始问题文本。 |
| `entities` | object | 否 | Agent 抽取的实体。 |
| `limit` | integer | 否 | 默认 5，最大 20。 |

`data`：

```json
{
  "items": [
    {
      "case_no": "case_20260501_000123",
      "issue_domain": "kline",
      "issue_type": "价格不一致",
      "summary": "行情源延迟导致 1m high 与外部交易所短时不一致",
      "score": 0.92
    }
  ]
}
```

## 给接入 AI 的实现步骤

1. 阅读本文档和 `internal/connectors/types.go`，确认返回字段与 Go 类型一致。
2. 在公司代码库中搜索是否已有对应只读接口。
3. 如果已有接口字段名不同，写 adapter 做字段映射，不要改 `ai-troubleshooter` 的 tool 参数。
4. 如果只能查 DB，必须使用 read replica 和预注册 SQL 模板，禁止透传 SQL。
5. 每个接口都要加 timeout、limit、审计日志和错误码。
6. 本地用 `CONNECTOR_MODE=http` 指向 adapter，执行 `make test`，再启动 `make dev` 用 Lark 模拟请求验证。

## 验收标准

- `GET /tools` 能看到 10 个工具。
- K线完整问题能自动调用 K线相关工具并生成报告。
- 资产完整问题能自动调用资产相关工具并生成报告。
- 字段不足的问题不会调用任何只读接口，只会追问。
- adapter 日志能按 `request_id` 和 `case_id` 查到每次调用。
- 任何写操作、自由 SQL、大范围日志查询都会被拒绝。
