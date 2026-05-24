# 业务方接入快速指南

这份文档给第一次接入的业务方和他们的 AI 使用。目标是：业务方只写只读 adapter 或 MCP readonly route，平台方启动 Agent Platform 和 Investigation Gateway 后，就能通过 Web Chat / Lark / 飞书排查问题。

## 1. 你需要理解的服务分工

| 服务 | 语言 | 谁维护 | 作用 |
| --- | --- | --- | --- |
| Agent Platform | Python 3.13 / FastAPI | Agent 平台 | Web Chat、Lark/飞书、图片、Case API、平台 MySQL、LLM/Vision 配置、orchestrator、经验沉淀。 |
| Decision Engine | Python 3.13 | Agent 平台 | Supervisor、Kline/Asset/HealthFood 等 specialist、Knowledge Agent、Verifier、工具计划和停止条件。 |
| Investigation Gateway | Go 1.24+ | Agent 平台 | 统一接业务只读证据，做 Bearer、agent/scope/tool/chat allowlist、限流、timeout、审计、脱敏。 |
| Readonly Adapter | 任意语言 | 业务方 | 只读查询业务证据，例如日志、用户资料、订单、资产、推荐状态、缓存状态。 |

业务方不需要提供平台 MySQL，也不需要提供 LLM。LLM/Vision 由 Agent Platform 统一配置。

## 2. 平台方初始化

```bash
python3.13 -m venv .venv
.venv/bin/python -m pip install -e apps/agent-platform

MYSQL_HOST=127.0.0.1 \
MYSQL_PORT=3306 \
MYSQL_USER=root \
MYSQL_PASSWORD="$LOCAL_MYSQL_PASSWORD" \
MYSQL_DATABASE=ai_troubleshooter \
make migrate-mysql
```

运行时配置：

```bash
export DB_DRIVER=mysql
export DB_DSN='ai_user:replace@tcp(127.0.0.1:3306)/ai_troubleshooter?parseTime=true&loc=Local'
export GATEWAY_ENDPOINT=http://127.0.0.1:18080
export AGENT_PLATFORM_PORT=19091
```

## 3. LLM 在哪里配置

LLM 只在 Python Agent Platform 配置。

Qwen / DashScope：

```bash
export AI_MODEL_PROFILE=qwen
export DASHSCOPE_API_KEY="$DASHSCOPE_API_KEY"
export QWEN_MODEL=qwen-plus
```

GPT / OpenAI：

```bash
export AI_MODEL_PROFILE=gpt
export OPENAI_API_KEY="$OPENAI_API_KEY"
export OPENAI_MODEL="replace-with-approved-model"
```

Claude / Anthropic：

```bash
export AI_MODEL_PROFILE=claude
export ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY"
export ANTHROPIC_MODEL="replace-with-approved-model"
```

Claude Code 或公司代理：

```bash
export AI_MODEL_PROFILE=claude_code
export CLAUDE_CODE_BASE_URL=http://127.0.0.1:19093
export CLAUDE_CODE_API_KEY="$LOCAL_PROXY_TOKEN"
export CLAUDE_CODE_MODEL="replace-with-proxy-model"
```

公司统一模型网关：

```bash
export LLM_PROVIDER=openai_compatible
export LLM_BASE_URL=https://llm-gateway.example.internal/v1
export LLM_API_KEY="$MODEL_GATEWAY_TOKEN"
export LLM_MODEL=replace-with-model
```

真实验收建议：

```bash
export LLM_ALLOW_RULE_FALLBACK=false
```

如果设置 `local_rules`，只能做页面和链路 smoke，不能宣称真实大模型排障。

## 4. 启动哪些服务

终端 1：Go Investigation Gateway

```bash
export HTTP_PORT=18080
export DB_DRIVER=mysql
export DB_DSN="$LOCAL_DB_DSN"
export CONNECTOR_MODE=mock
make gateway
```

终端 2：Python Agent Platform

```bash
export PYTHONPATH=apps/agent-platform:apps/decision-engine
make dev
```

打开：

```text
http://localhost:19091/web
```

自动化或 AI 接入测试可以直接调用 Agent Platform API：

```bash
curl -s -X POST http://localhost:19091/api/v1/chat \
  -F 'message=health-food uid hf-user-001 today token quota wrong' \
  -F 'async=0'

curl -s http://localhost:19091/api/v1/cases/case_20260524_000001
```

Lark/飞书入口也在 Python Agent Platform：

```bash
export LARK_VERIFICATION_TOKEN="$LOCAL_LARK_TOKEN"
export LARK_ENCRYPT_KEY="$LOCAL_LARK_ENCRYPT_KEY"
export LARK_ALLOWED_CHAT_IDS=oc_dev
```

启用 `LARK_ENCRYPT_KEY` 后，平台只接受 encrypted callback；图片消息需要额外配置 `LARK_APP_ID/LARK_APP_SECRET` 才会下载资源。

## 5. 业务方怎么写只读接口

只允许查询，不允许写、删、执行脚本或透传任意 SQL。

接口路径必须放在 `/v1/readonly/{service}/...` 下，建议 POST：

```http
POST /v1/readonly/health-food/recommendation/status
Authorization: Bearer ${CONNECTOR_API_KEY}
Content-Type: application/json
```

请求 envelope：

```json
{
  "request_id": "req_xxx",
  "case_id": "case_20260524_000001",
  "agent_id": "business-troubleshooter-v1",
  "caller_user_id": "web_user",
  "tool_name": "get_health_food_recommendation_status",
  "timeout_ms": 5000,
  "params": {
    "uid": "hf-user-001",
    "recommendation_date": "2026-05-24"
  }
}
```

响应 envelope：

```json
{
  "request_id": "req_xxx",
  "source": "health-food-readonly-api",
  "queried_at": "2026-05-24T12:00:00+08:00",
  "data_updated_at": "2026-05-24T11:59:00+08:00",
  "version": "v1",
  "data": {
    "uid": "hf-user-001",
    "has_recommendation": false,
    "job_status": "failed",
    "failure_reason": "meal fingerprint did not refresh"
  },
  "warnings": []
}
```

接口实现要求：

- 只读：只能 `SELECT`、只读日志查询、只读缓存查询。
- 参数化：MySQL 使用 ORM / Query Builder / DB-API 参数绑定，禁止字符串拼接 SQL。
- 限制范围：时间窗、limit、用户维度必须有上限。
- 脱敏：手机号、邮箱、token、secret、身份证、完整 open_id 不返回明文。
- 审计：记录 `request_id`、`case_id`、`agent_id`、`tool_name`。
- 超时：每个下游查询必须设置 timeout。

## 6. 写完接口怎么注册到 Gateway

在 Web 工作台“能力接入”粘贴 manifest：

```yaml
service:
  service_name: health-food
  owner_team: health
  environment: prod
  base_url: https://health-food-readonly.internal
  auth:
    type: bearer
    token_env: CONNECTOR_API_KEY
capabilities:
  - tool_name: get_health_food_recommendation_status
    description: 查询 health-food 每日推荐生成状态、输入餐食和失败原因
    scope: health_food:recommendation:read
    method: POST
    path: /v1/readonly/health-food/recommendation/status
    timeout_ms: 5000
    max_time_range_minutes: 1440
    sensitivity_level: sensitive
    required_params:
      - uid
      - recommendation_date
    optional_params:
      - trace_id
```

发布后，Agent Platform 会请求 Gateway reload；如果控制面鉴权开启，需要配置：

```bash
export CONTROL_API_AUTH_ENABLED=true
export CONTROL_API_BEARER_TOKENS="$CONTROL_TOKEN"
export GATEWAY_ADMIN_BEARER_TOKEN="$CONTROL_TOKEN"
```

## 7. Gateway Agent Bearer 怎么配

生产建议用 `configs/gateway-agents.example.json` 这种结构：

```json
{
  "agents": [
    {
      "agent_id": "business-troubleshooter-v1",
      "status": "enabled",
      "bearer_token_env": "BUSINESS_TROUBLESHOOTER_GATEWAY_TOKEN",
      "allowed_scopes": ["health_food:user:read", "health_food:recommendation:read", "logs:read_summary"],
      "allowed_tools": ["get_health_food_user_profile", "get_health_food_recommendation_status", "search_logs_by_service"],
      "rate_limit_qps": 5
    }
  ]
}
```

启动：

```bash
export GATEWAY_AUTH_ENABLED=true
export GATEWAY_AGENT_CONFIG_FILE=configs/gateway-agents.example.json
export BUSINESS_TROUBLESHOOTER_GATEWAY_TOKEN="$RANDOM_LONG_TOKEN"
export GATEWAY_BEARER_TOKEN="$BUSINESS_TROUBLESHOOTER_GATEWAY_TOKEN"
```

Agent Platform 调 Gateway 时会自动带 `Authorization: Bearer $GATEWAY_BEARER_TOKEN`。

## 8. 验收标准

接入完成至少验证：

```bash
curl -s http://127.0.0.1:18080/healthz
curl -s http://127.0.0.1:18080/tools
curl -s http://127.0.0.1:19091/healthz
```

然后在 Web Chat 输入自然语言问题，例如：

```text
health-food uid hf-user-001 今日没有每日推荐
```

验收必须能证明：

- Web 入口创建 case。
- Python orchestrator 记录 `classify_extract`、`orchestrator_plan`、`tool_invocation`、`summarize_findings`。
- Gateway tool audit 有记录。
- 业务 readonly adapter 被真实调用。
- 平台 MySQL 查得到 case、message、ai decision log。

只用 mock、local_rules 或 memory 时，结论必须降级，不能写成真实生产验收。
