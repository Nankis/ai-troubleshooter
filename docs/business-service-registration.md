# 业务服务能力注册规范

本文定义业务服务接入 Investigation Gateway 时需要提供的数据结构。目标是：业务方只声明“我能提供哪些只读证据”，Agent 平台负责鉴权、scope、限流、timeout、审计、脱敏和工具编排。

能力可以通过两种方式注册：

- 提交 manifest 文件，由平台工程合并部署。
- 在 Web 工作台“能力接入”直接粘贴 YAML/JSON，先落库为 draft/rejected，再由平台发布。

Web 发布不会绕过安全边界：只有名称、描述、scope、method、path 都符合只读信号，且 path 位于 `/readonly/` 下的 capability 才会成为 `readonly_candidate`，其余会保持 `needs_review` 或 `rejected`。

## 注册对象

业务服务不要把 DB、SQL、Redis key 或生产 token 直接交给 Agent。业务方只提供一个 readonly adapter，并提交一份能力注册 manifest。若业务方已有 MCP server，也必须先用 MCP readonly adapter 映射成同一套 readonly HTTP contract，再注册到 Gateway：

```yaml
service:
  service_name: health-food
  owner_team: health
  runtime: spring-boot
  environment: local
  base_url: http://127.0.0.1:19081
  health_check:
    method: GET
    path: /healthz
  auth:
    type: bearer
    token_env: CONNECTOR_API_KEY
  data_classification:
    contains_user_data: true
    contains_financial_data: false
    contains_health_data: true
    pii_level: internal_sensitive
  contacts:
    primary: health-food-owner
capabilities:
  - tool_name: get_health_food_recommendation_status
    description: 查询每日推荐生成状态、输入餐食和失败原因
    scope: health_food:recommendation:read
    method: POST
    path: /v1/readonly/health-food/recommendation/status
    timeout_ms: 5000
    max_qps: 5
    max_time_range_minutes: 1440
    sensitivity_level: sensitive
    required_params:
      - uid
      - recommendation_date
    optional_params:
      - start_time
      - end_time
      - trace_id
    response_data_schema_ref: HealthFoodRecommendationStatus
    failure_modes:
      - code: HEALTH_FOOD_JOB_FAILED
        retryable: false
        description: 推荐任务失败，需查看业务日志或任务输入
```

## service 字段

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `service_name` | 是 | 稳定服务名，建议使用部署/日志中的真实服务名，例如 `health-food`。 |
| `owner_team` | 是 | 服务归属团队。 |
| `runtime` | 否 | 技术栈，例如 `spring-boot`、`go`。 |
| `environment` | 是 | `local`、`pre`、`prod`。 |
| `base_url` | 是 | readonly adapter 地址，不是业务公网入口。 |
| `health_check` | 是 | Gateway 或部署脚本用于验证 adapter 是否可达。 |
| `auth` | 是 | adapter 鉴权方式。一期建议 Bearer token，token 只能来自环境变量或密钥系统。 |
| `data_classification` | 是 | 数据分级，用于默认脱敏和审计策略。 |
| `contacts` | 是 | owner / oncall 联系方式，不写私人敏感信息。 |

## capability 字段

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `tool_name` | 是 | Agent 可见的工具名，必须全局唯一。 |
| `description` | 是 | 给 Agent 和人工审核看的能力描述。 |
| `scope` | 是 | Gateway policy 校验的 scope。 |
| `method` | 是 | 一期固定 `POST`，health check 可为 `GET`。 |
| `path` | 是 | readonly adapter endpoint。 |
| `timeout_ms` | 是 | 单次调用超时预算。 |
| `max_qps` | 是 | 单工具或单服务限流建议。 |
| `max_time_range_minutes` | 否 | 时间范围上限；没有时间范围的工具可不填。 |
| `max_limit` | 否 | 返回条数上限。 |
| `sensitivity_level` | 是 | `normal`、`sensitive`、`secret`。 |
| `required_params` | 是 | 最小必要参数，Agent 缺字段时应追问。 |
| `optional_params` | 否 | 可选过滤参数。 |
| `response_data_schema_ref` | 是 | 返回 `data` 的 schema 名称。 |
| `failure_modes` | 否 | 业务方已知错误码、是否可重试、排查提示。 |

## 通用请求 envelope

Gateway 调用 adapter 时统一发送：

```json
{
  "request_id": "req_xxx",
  "case_id": "case_20260523_000001",
  "agent_id": "business-troubleshooter-v1",
  "caller_user_id": "ou_xxx",
  "tool_name": "get_health_food_recommendation_status",
  "timeout_ms": 5000,
  "params": {
    "uid": "hf_user_001",
    "recommendation_date": "2026-05-23"
  }
}
```

请求头必须包含：

```text
Authorization: Bearer ${CONNECTOR_API_KEY}
Content-Type: application/json
X-Request-Id: req_xxx
X-Case-Id: case_20260523_000001
X-Agent-Id: business-troubleshooter-v1
X-Caller-User-Id: ou_xxx
X-Tool-Name: get_health_food_recommendation_status
```

## 通用响应 envelope

```json
{
  "request_id": "req_xxx",
  "source": "health-food",
  "queried_at": "2026-05-23T10:00:03+08:00",
  "data_updated_at": "2026-05-23T10:00:00+08:00",
  "version": "v1",
  "data": {},
  "warnings": []
}
```

错误响应：

```json
{
  "code": "HEALTH_FOOD_JOB_FAILED",
  "error": "daily recommendation job failed"
}
```

可空时间字段必须返回 `null` 或省略字段，不要返回空字符串。否则强类型 adapter 在解析 RFC3339 时间时会把空字符串视为协议错误。

## health-food 首批能力

| tool_name | scope | path | 用途 |
| --- | --- | --- | --- |
| `get_health_food_user_profile` | `health_food:user:read` | `/v1/readonly/health-food/user/profile` | 查询用户是否存在、会员等级、健康目标摘要、最近设备。 |
| `get_health_food_ai_quota` | `health_food:ai_quota:read` | `/v1/readonly/health-food/ai/quota` | 查询 AI token / 每日对话次数配额，定位“明明会员还在但不能问 AI”。 |
| `get_health_food_meal_records` | `health_food:meal:read` | `/v1/readonly/health-food/meals/range` | 查询时间窗内餐食记录、缺失餐次和数据指纹。 |
| `get_health_food_recommendation_status` | `health_food:recommendation:read` | `/v1/readonly/health-food/recommendation/status` | 查询每日推荐是否生成、任务状态、失败原因和输入餐食。 |

### HealthFoodRecommendationStatus

```json
{
  "uid": "hf_user_001",
  "recommend_date": "2026-05-23",
  "has_recommendation": false,
  "job_status": "failed",
  "meal_count": 2,
  "meal_data_fingerprint": "fingerprint_stale",
  "generated_at": null,
  "failure_reason": "meal_data_fingerprint did not refresh after dinner upload",
  "source_meal_ids": ["meal_breakfast", "meal_lunch"]
}
```

## 接入检查清单

- Adapter 必须只读，禁止写业务数据。
- Web 录入时只能填写 `secret_ref`/`token_env` 这样的环境变量名，不能填写真实 token。
- Adapter 必须校验 Bearer token，并记录 `request_id/case_id/agent_id/tool_name`。
- Adapter 必须给每个下游查询设置 timeout。
- 返回前删除原始手机号、邮箱、完整三方 open_id、原始图片 URL、支付凭证、模型 API key 等敏感字段。
- 大字段只返回摘要和证据 ID，不返回完整日志或完整 prompt。
- 业务 DB 由业务服务自己访问；Agent 和 Gateway 不直接连业务 DB。
