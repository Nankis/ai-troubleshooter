# 部署检查清单

## 必填配置

```bash
APP_ENV=prod
HTTP_PORT=8080

LARK_APP_ID=cli_xxx
LARK_APP_SECRET=xxx
LARK_VERIFICATION_TOKEN=xxx
LARK_ENCRYPT_KEY=xxx
LARK_ALLOWED_CHAT_IDS=oc_xxx,oc_yyy

LLM_PROVIDER=openai_compatible
LLM_BASE_URL=https://llm-gateway.internal/v1
LLM_API_KEY=xxx
LLM_MODEL=replace-with-model

DB_DRIVER=mysql
DB_DSN='user:password@tcp(mysql.internal:3306)/ai_troubleshooter?parseTime=true&loc=Local'

CONNECTOR_MODE=http
CONNECTOR_API_KEY=xxx
MARKET_READONLY_BASE_URL=https://market-readonly.internal
ASSET_READONLY_BASE_URL=https://asset-readonly.internal
OPS_READONLY_BASE_URL=https://ops-readonly.internal

GATEWAY_AUTH_ENABLED=true
GATEWAY_BEARER_TOKENS='business-troubleshooter-v1:replace-with-strong-token'
GATEWAY_AGENT_QPS=20
GATEWAY_USER_QPS=10
GATEWAY_TOOL_QPS=20

CONTROL_API_AUTH_ENABLED=true
CONTROL_API_BEARER_TOKENS='replace-with-internal-control-token'
```

## 接入前检查

- Lark 机器人只加入允许的群。
- `LARK_ALLOWED_CHAT_IDS` 已配置，避免任意群触发。
- 只读 adapter 已按 `docs/ai-connector-integration.md` 暴露 10 个接口。
- adapter 对所有底层查询设置 timeout、limit 和审计。
- adapter 不提供写操作，不透传 SQL。
- Gateway HTTP 鉴权已开启：`GATEWAY_AUTH_ENABLED=true`。
- Gateway Bearer token 通过密钥系统注入，不写入 Git 或镜像。
- 调用 Gateway 的 orchestrator/worker 已使用与 `agent_id` 绑定的 token。
- Gateway 上游入口已做内网 ACL、Ingress allowlist 或 service mesh 策略。
- 控制面 API 已开启内部 Bearer 鉴权：`CONTROL_API_AUTH_ENABLED=true`。
- root cause、feedback、knowledge、orchestrator case/process API 仅允许内部系统或已授权 owner 调用。
- 所有敏感字段在 adapter 或 Gateway 返回前脱敏。
- 数据库已执行 `migrations/001_initial.sql` 和 `migrations/002_knowledge_evolution.sql`，DSN 必须包含 `parseTime=true`。
- `DB_DSN` 已提供给需要持久化 case、knowledge 和 tool audit 的服务；Gateway 会把工具审计写入 `tool_call_audits`。
- 业务 owner 已明确 root cause 回填责任人和推荐枚举。

## 本地 smoke test

```bash
make test
HTTP_PORT=19091 make dev
```

完整 K线 case：

```bash
curl -s localhost:19091/lark/events \
  -H 'Content-Type: application/json' \
  -d '{
    "chat_id":"oc_dev",
    "thread_id":"thread_dev",
    "message_id":"msg_1",
    "user_id":"ou_dev",
    "text":"@排障机器人 用户反馈 BTCUSDT 1m K线价格不一致，异常时间 2026-05-21T20:00:00+08:00，对比 Binance"
  }'
```

信息不足 case：

```bash
curl -s localhost:19091/lark/events \
  -H 'Content-Type: application/json' \
  -d '{
    "chat_id":"oc_dev",
    "thread_id":"thread_dev",
    "message_id":"msg_2",
    "user_id":"ou_dev",
    "text":"@排障机器人 用户反馈余额变少了"
  }'
```

预期：

- 完整 K线 case 进入 `NEED_HUMAN_CONFIRMATION`。
- 信息不足 case 进入 `WAITING_USER_REPLY`。
- 工具调用审计日志包含 tool name、case id、policy decision、query id。

根因回填与知识自进化：

```bash
curl -s localhost:19091/cases/case_20260521_000001/root-cause \
  -H 'Content-Type: application/json' \
  -d '{
    "human_confirmed_reason":"行情源短时延迟，补偿任务完成前用户看到旧 high",
    "root_cause_category":"external_source_delay",
    "owner_service":"market-service",
    "is_external_source_issue":true,
    "confirmed_by":"owner_1"
  }'

curl -s 'localhost:19091/knowledge?issue_domain=kline'
```

预期：

- case 状态进入 `DONE`。
- 响应包含 `root_cause`、`knowledge_item`、`evolution_run`。
- `/knowledge` 能查到新增或更新后的知识条目。
