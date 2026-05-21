# 部署检查清单

## 必填配置

```bash
APP_ENV=prod
HTTP_PORT=8080

LARK_APP_ID=cli_xxx
LARK_APP_SECRET=xxx
LARK_PLATFORM=lark
# Optional override. Lark default: https://open.larksuite.com; Feishu default: https://open.feishu.cn
LARK_API_BASE_URL=
LARK_VERIFICATION_TOKEN=xxx
LARK_ENCRYPT_KEY=xxx
LARK_ALLOWED_CHAT_IDS=oc_xxx,oc_yyy

LLM_PROVIDER=openai_compatible
LLM_BASE_URL=https://llm-gateway.example.internal/v1
LLM_API_KEY=xxx
LLM_MODEL=replace-with-model

VISION_PROVIDER=qwen_openai_compatible
VISION_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
VISION_API_KEY=xxx
VISION_MODEL=qwen3-vl-plus
VISION_MAX_IMAGES_PER_MESSAGE=3
VISION_MAX_IMAGE_BYTES=10485760

DB_DRIVER=mysql
DB_DSN='ai_user:replace-with-db-password@tcp(mysql.example.internal:3306)/ai_troubleshooter?parseTime=true&loc=Local'

CONNECTOR_MODE=http
CONNECTOR_API_KEY=xxx
MARKET_READONLY_BASE_URL=https://market-readonly.example.internal
ASSET_READONLY_BASE_URL=https://asset-readonly.example.internal
OPS_READONLY_BASE_URL=https://ops-readonly.example.internal

GATEWAY_AUTH_ENABLED=true
GATEWAY_BEARER_TOKENS='business-troubleshooter-v1:replace-with-strong-token'
GATEWAY_AGENT_QPS=20
GATEWAY_USER_QPS=10
GATEWAY_TOOL_QPS=20

CONTROL_API_AUTH_ENABLED=true
CONTROL_API_BEARER_TOKENS='replace-with-internal-control-token'

MAX_TOOL_CALLS_PER_CASE=10
MAX_TOOL_FAILURES_PER_CASE=3
MAX_INVESTIGATION_SECONDS=120
```

## 接入前检查

- Lark 优先接入：默认 `LARK_PLATFORM=lark`，系统使用 `https://open.larksuite.com` 调用开放平台。
- 飞书中国站接入：设置 `LARK_PLATFORM=feishu`，系统使用 `https://open.feishu.cn`；如公司有代理网关，可用 `LARK_API_BASE_URL` 覆盖。
- Lark/飞书机器人只加入允许的群。
- `LARK_ALLOWED_CHAT_IDS` 已配置，避免任意群触发。
- Lark 事件订阅使用 `POST /lark/events`；飞书中国站可以使用 `POST /feishu/events`，两者复用同一套 payload、加密和消息处理逻辑。
- 内部联调如需机器人在群里真实回复，已配置 `LARK_APP_ID` 和 `LARK_APP_SECRET`，并确保应用开启机器人能力。
- Lark/飞书事件订阅启用 Encrypt Key 时，必须同步配置 `LARK_ENCRYPT_KEY`；系统会先解密 `encrypt` 回调体，再校验 `LARK_VERIFICATION_TOKEN`。
- 配置 `LARK_ENCRYPT_KEY` 后，Lark/飞书入口只接受密文回调，明文 payload 会返回 `400`，避免加密降级。
- 如果需要识别截图，Lark/飞书应用必须具备读取消息资源/图片资源的权限，并配置 `LARK_APP_ID`、`LARK_APP_SECRET` 供系统下载图片。
- 视觉模型建议独立配置：`VISION_PROVIDER=qwen_openai_compatible` 使用千问视觉识别图片，后续 `LLM_*` 仍可接 GPT/Claude 做文字推理。
- 原图默认只在内存中短暂处理，不写入 MySQL；如需留存原图，必须接公司对象存储、保留周期和数据分级审批。
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
- 数据库已依次执行 `migrations/001_initial.sql`、`migrations/002_knowledge_evolution.sql`、`migrations/003_ai_decision_logs.sql` 和 `migrations/004_case_idempotency.sql`，DSN 必须包含 `parseTime=true`。
- `DB_DSN` 已提供给需要持久化 case、knowledge、tool audit 和 AI decision logs 的服务；Gateway 会把工具审计写入 `tool_call_audits`，orchestrator 会把 AI 决策写入 `ai_decision_logs`。
- `MAX_TOOL_CALLS_PER_CASE`、`MAX_TOOL_FAILURES_PER_CASE`、`MAX_INVESTIGATION_SECONDS` 已按业务下游承载能力设置。
- Lark/飞书重复投递已验证：同一个 `source + message_id` 重放只返回已有 `case_no`，不重复入队、不重复查下游。
- `ai_decision_logs` 快照已验证脱敏：手机号、邮箱、token、secret、api key 不出现明文。
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

图片消息本地模拟：

```bash
curl -s localhost:19091/lark/events \
  -H 'Content-Type: application/json' \
  -d '{
    "chat_id":"oc_dev",
    "thread_id":"thread_dev",
    "message_id":"msg_image_1",
    "user_id":"ou_dev",
    "text":"@排障机器人 帮忙看截图",
    "image_keys":["img_dev_1"]
  }'
```

飞书中国站入口兼容检查：

```bash
curl -s localhost:19091/feishu/events \
  -H 'Content-Type: application/json' \
  -d '{
    "chat_id":"oc_dev",
    "thread_id":"thread_dev",
    "message_id":"msg_feishu_1",
    "user_id":"ou_dev",
    "text":"@排障机器人 用户反馈 BTCUSDT 1m K线价格不一致，异常时间 2026-05-21T20:00:00+08:00"
  }'
```

真实 Lark/飞书联调时，图片下载需要 `LARK_APP_ID` / `LARK_APP_SECRET`；本地 mock 模式下如未配置真实 Bot，响应中会提示图片未下载，不影响文字工单链路。

重复投递检查：

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

预期：

- 完整 K线 case 进入 `NEED_HUMAN_CONFIRMATION`。
- 信息不足 case 进入 `WAITING_USER_REPLY`。
- 工具调用审计日志包含 tool name、case id、policy decision、query id。
- 重复投递响应包含 `duplicate=true`，worker 不会产生第二轮工具调用。
- 有图片时，`cases.ocr_text` 或 `/cases/{case_no}` 响应中的 `ocr_text` 包含视觉识别结果；下游工具选择会使用这些字段。

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

curl -s 'localhost:19091/cases/case_20260521_000001/ai-decisions?limit=100'
```

预期：

- case 状态进入 `DONE`。
- 响应包含 `root_cause`、`knowledge_item`、`evolution_run`。
- `/knowledge` 能查到新增或更新后的知识条目。
- `/cases/{case_no}/ai-decisions` 能查到分类、实体抽取、工具计划、工具调用和总结日志。
- 决策日志快照里的敏感字段已脱敏；重复处理会出现 `process_skipped`。
