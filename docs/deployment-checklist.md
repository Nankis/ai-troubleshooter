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

CONNECTOR_MODE=http
CONNECTOR_API_KEY=xxx
MARKET_READONLY_BASE_URL=https://market-readonly.internal
ASSET_READONLY_BASE_URL=https://asset-readonly.internal
OPS_READONLY_BASE_URL=https://ops-readonly.internal
```

## 接入前检查

- Lark 机器人只加入允许的群。
- `LARK_ALLOWED_CHAT_IDS` 已配置，避免任意群触发。
- 只读 adapter 已按 `docs/ai-connector-integration.md` 暴露 10 个接口。
- adapter 对所有底层查询设置 timeout、limit 和审计。
- adapter 不提供写操作，不透传 SQL。
- 所有敏感字段在 adapter 或 Gateway 返回前脱敏。

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
