# 本地运行手册

README 只保留入口信息，具体启动、调试和 adapter 验证放在这里。所有 key、token、MySQL 密码都只能通过环境变量传入，不能写进仓库文件、示例输出或提交记录。

## 前置依赖

```bash
go version          # 需要 Go 1.24+
python3.13 --version
mysql --version
```

建议先安装本地 hook：

```bash
make install-hooks
make secret-scan
```

## MySQL 初始化

```bash
MYSQL_HOST=127.0.0.1 \
MYSQL_PORT=3306 \
MYSQL_USER=root \
MYSQL_PASSWORD="$LOCAL_MYSQL_PASSWORD" \
MYSQL_DATABASE=ai_troubleshooter \
make migrate-mysql
```

服务运行时使用标准 DSN：

```bash
export DB_DRIVER=mysql
export DB_DSN="$LOCAL_DB_DSN"
```

`DB_DRIVER=mysql` 但没有 `DB_DSN` 会启动失败。只有一次性 smoke 才允许显式使用：

```bash
export DB_DRIVER=memory
unset DB_DSN
```

任何要验证 case、消息、AI 决策日志、工具审计或经验沉淀的场景，都必须使用 MySQL 并查询表确认。

## Web Chat

最小启动：

```bash
export DB_DRIVER=mysql
export DB_DSN="$LOCAL_DB_DSN"
export CONNECTOR_MODE=mock
export LLM_PROVIDER=local_rules
export VISION_PROVIDER=local_rules
export HTTP_PORT=8080
make dev
```

浏览器打开：

```text
http://localhost:8080/web
```

端口冲突时：

```bash
HTTP_PORT=18088 make dev
```

Web 工作台支持：

- 文字输入和图片上传。
- 截图复制后直接粘贴上传。
- 图片缩略图预览和单击放大。
- 新建问题会话、切换会话、继续当前 case。
- 问题会话支持重命名和删除；未发送草稿保存在浏览器 localStorage，正式 case/message/AI 决策仍写 MySQL。
- 左侧按服务分组查看 Gateway tools，并可折叠。
- 在“能力接入”粘贴 Claude/Cursor MCP JSON、MCP routes JSON 或 readonly manifest YAML/JSON，审核后发布只读工具。
- 平台经验预览、编辑、录入和软删除。
- 右侧查看当前排查状态、AI 决策步骤、工具调用进度和运行环境。

## 模型配置

本地 smoke 可以用规则模型：

```bash
export LLM_PROVIDER=local_rules
export VISION_PROVIDER=local_rules
```

接 OpenAI-compatible 文本模型：

```bash
export LLM_PROVIDER=openai_compatible
export LLM_BASE_URL=https://llm-gateway.example.internal/v1
export LLM_API_KEY="$LOCAL_LLM_API_KEY"
export LLM_MODEL=replace-with-model
```

图片识别默认复用主 LLM。只有主模型不支持图片，或需要单独用 Qwen-VL 等视觉模型时，再配置独立视觉 provider：

```bash
export VISION_PROVIDER=qwen_openai_compatible
export VISION_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
export VISION_API_KEY="$DASHSCOPE_API_KEY"
export VISION_MODEL=qwen-vl-plus
export VISION_MAX_IMAGES_PER_MESSAGE=3
export VISION_MAX_IMAGE_BYTES=10485760
```

## Python Decision Engine

单独启动 Python 决策层：

```bash
cd apps/decision-engine
python3.13 -m decision_engine --host 127.0.0.1 --port 19092
```

本地单测：

```bash
PYTHONPATH=apps/decision-engine python3.13 -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'
python3.13 -m unittest discover -s tests -p 'test_*.py'
```

Go worker 当前仍可使用 Go fallback 跑本地闭环。切换到 Python 决策层时，外部契约保持不变：输入入口、Case API、Gateway tools 和 MySQL 表结构不因为决策层语言切换而改变。

## Lark / 飞书本地 payload

启动 dev server 后，可以用本地 payload 模拟事件：

```bash
curl -s localhost:8080/lark/events \
  -H 'Content-Type: application/json' \
  -d '{
    "chat_id":"oc_dev",
    "thread_id":"thread_dev",
    "message_id":"msg_1",
    "user_id":"ou_dev",
    "text":"@排障机器人 用户反馈 BTCUSDT 1m K线价格不一致，异常时间 2026-05-21T20:00:00+08:00，对比 Binance"
  }'
```

飞书中国站兼容入口：

```bash
curl -s localhost:8080/feishu/events \
  -H 'Content-Type: application/json' \
  -d '{
    "chat_id":"oc_dev",
    "thread_id":"thread_dev",
    "message_id":"msg_feishu_1",
    "user_id":"ou_dev",
    "text":"@排障机器人 用户反馈 今日 token 消耗数量不对，uid 123456"
  }'
```

真实 bot 需要按公司环境配置 `LARK_PLATFORM`、`LARK_VERIFICATION_TOKEN`、`LARK_ENCRYPT_KEY`、`LARK_ALLOWED_CHAT_IDS`、`LARK_APP_ID`、`LARK_APP_SECRET`。配置细节见 [gateway-security.md](gateway-security.md) 和 [deployment-checklist.md](deployment-checklist.md)。

## Gateway tools

查看工具：

```bash
curl -s localhost:8080/tools
```

直接调用工具：

```bash
curl -s localhost:8080/tools/get_asset_snapshot/invoke \
  -H 'Content-Type: application/json' \
  -d '{
    "case_id":"case_dev",
    "agent_id":"business-troubleshooter-v1",
    "caller_user_id":"ou_dev",
    "chat_id":"oc_dev",
    "arguments":{
      "user_id":"user_123",
      "asset_symbol":"USDT",
      "at_time":"2026-05-21T20:00:00+08:00"
    }
  }'
```

如果开启 `GATEWAY_AUTH_ENABLED=true`，需要带 Bearer：

```bash
curl -s localhost:8080/tools/get_asset_snapshot/invoke \
  -H "Authorization: Bearer $GATEWAY_BEARER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"case_id":"case_dev","agent_id":"business-troubleshooter-v1","caller_user_id":"ou_dev","chat_id":"oc_dev","arguments":{"user_id":"user_123","asset_symbol":"USDT"}}'
```

推荐用 `GATEWAY_AGENT_CONFIG_FILE` 或 `GATEWAY_AGENT_CONFIG_JSON` 配置 agent、scope、tool、chat allowlist 和 `bearer_token_env`。示例见 [../configs/gateway-agents.example.json](../configs/gateway-agents.example.json)。

## Web 动态能力接入

工作台左侧“能力接入”支持三类配置：

- Claude/Cursor 风格 `mcpServers` JSON：只记录 MCP server 和待发现状态，不执行 command，不发布 tool。
- MCP readonly adapter `routes` JSON：只接受 `/readonly/` 路径，写操作、任意 SQL、命令执行会进入 rejected。
- 标准 HTTP readonly manifest YAML/JSON：按 `service` 和 `capabilities` 创建 draft capability。

发布规则：

- 只有 `readonly_candidate` 能点“发布”。
- 发布后会写入 MySQL `tb_troubleshoot_tool_registry`，并在当前 dev-server 进程热加载到 Gateway tools。
- 如果生产使用显式 `GATEWAY_AGENT_CONFIG_FILE`，新 tool 对应的 `scope` 和 `tool_name` 仍需要加入 agent allowlist；本地默认 agent 允许已发布且 scope 允许的动态工具。
- `secret_ref` 只能填环境变量或密钥引用名，例如 `CONNECTOR_API_KEY`，不要填真实 token。

## 标准 HTTP readonly adapter

让 Gateway 调用公司只读 adapter：

```bash
export CONNECTOR_MODE=http
export CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY"
export MARKET_READONLY_BASE_URL=https://market-readonly.example.internal
export ASSET_READONLY_BASE_URL=https://asset-readonly.example.internal
export OPS_READONLY_BASE_URL=https://ops-readonly.example.internal
export HEALTH_FOOD_READONLY_BASE_URL=https://health-food-readonly.example.internal
```

adapter 规范见 [ai-connector-integration.md](ai-connector-integration.md)。业务服务注册 manifest 见 [business-service-registration.md](business-service-registration.md)，health-food 示例见 [../configs/business-capabilities.health-food.example.yaml](../configs/business-capabilities.health-food.example.yaml)。

## MCP readonly adapter

MCP server 不能让决策层直连，必须先映射成 allowlisted readonly HTTP route，再交给 Gateway。

```bash
MCP_ADAPTER_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
MCP_READONLY_ADAPTER_PORT=19085 \
python3.13 scripts/mcp-readonly-adapter.py
```

配置和验收标准见 [mcp-gateway-adapter.md](mcp-gateway-adapter.md)。health-food MCP route 示例见 [../configs/mcp-health-food-adapter.example.json](../configs/mcp-health-food-adapter.example.json)。

阿里云 DMS 可以作为 DB 元数据和 named readonly query 的证据入口。DMS 接入设计见 [dms-mcp-integration.md](dms-mcp-integration.md)，元数据 route 示例见 [../configs/mcp-dms-adapter.metadata.example.json](../configs/mcp-dms-adapter.metadata.example.json)。

## health-food 本地真实 adapter

`scripts/real-health-food-readonly-adapter.py` 用于本地真实验证。它查询本地 health-food 测试库、探活本地 health-food 服务，并按 readonly adapter envelope 暴露首批接口，不合成 mock 故障数据，也不允许 Agent 直接访问生产 DB。

```bash
# 1. 先启动本地 health-food，确保 /food-health/sys/alive 可访问。

# 2. 启动真实 readonly adapter。
CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
HEALTH_FOOD_MYSQL_HOST=127.0.0.1 \
HEALTH_FOOD_MYSQL_PORT=3306 \
HEALTH_FOOD_MYSQL_USER=root \
HEALTH_FOOD_MYSQL_PASSWORD="$LOCAL_MYSQL_PASSWORD" \
HEALTH_FOOD_MYSQL_DATABASE=hf_troubleshoot_codex \
HEALTH_FOOD_BASE_URL=http://127.0.0.1:18080 \
REAL_HEALTH_FOOD_ADAPTER_PORT=19084 \
python3.13 scripts/real-health-food-readonly-adapter.py

# 3. 让排障平台通过 HTTP connector 接入该 adapter。
CONNECTOR_MODE=http \
CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
MARKET_READONLY_BASE_URL=http://127.0.0.1:19084 \
ASSET_READONLY_BASE_URL=http://127.0.0.1:19084 \
OPS_READONLY_BASE_URL=http://127.0.0.1:19084 \
HEALTH_FOOD_READONLY_BASE_URL=http://127.0.0.1:19084 \
go run ./cmd/dev-server
```

验收标准不是“流程能返回”，而是 Web Chat 或 case API 能查到可靠证据：真实用户存在、真实餐食记录存在、真实推荐记录缺失或任务状态明确、工具审计和 AI 决策日志落库。必要时再启用 debug-only Local Code Agent，根据 `service_name` 定位本地代码路径和调用关系。

## health-food 生产只读日志

生产问题排查优先查询生产只读证据，不直连生产 DB。health-food 生产日志接入通过本地 adapter 桥接内部日志查询接口，并在返回前做服务名 allowlist、时间窗、limit、超时和脱敏。

```bash
CONNECTOR_API_KEY="$LOCAL_CONNECTOR_API_KEY" \
HEALTH_FOOD_ADMIN_BASE_URL="https://health-food.example.com" \
HEALTH_FOOD_ADMIN_SECRET="$HEALTH_FOOD_ADMIN_SECRET" \
REAL_HEALTH_FOOD_ADAPTER_PORT=19084 \
python3.13 scripts/real-health-food-readonly-adapter.py
```

完整命令和生产验收标准见 [health-food-production-integration.md](health-food-production-integration.md)。生产验收必须实际调用生产 health-food 日志接口，并查到问题时间窗内的可靠证据；mock 只能算链路自测。

## 经验沉淀

回填根因：

```bash
curl -s localhost:8080/cases/case_20260521_000001/root-cause \
  -H 'Content-Type: application/json' \
  -d '{
    "human_confirmed_reason":"行情源短时延迟，补偿任务完成前用户看到旧 high",
    "root_cause_category":"external_source_delay",
    "owner_service":"market-service",
    "is_external_source_issue":true,
    "prevention_action":"增加行情源延迟监控和补偿任务告警",
    "confirmed_by":"owner_1"
  }'
```

查询知识库：

```bash
curl -s 'localhost:8080/knowledge?issue_domain=kline&issue_type=价格不一致'
```

查询 AI 决策轨迹：

```bash
curl -s 'localhost:8080/cases/case_20260521_000001/ai-decisions?limit=100'
```

如果开启控制面鉴权，需要带 `Authorization: Bearer $CONTROL_API_BEARER_TOKEN`。

## 容器部署

构建本地一体化服务：

```bash
docker build --build-arg SERVICE=dev-server -t ai-troubleshooter:dev .
docker run --rm -p 8080:8080 --env CONNECTOR_MODE=mock ai-troubleshooter:dev
```

构建独立 Gateway：

```bash
docker build --build-arg SERVICE=investigation-gateway -t ai-troubleshooter-gateway:dev .
```

Compose 示例见 [../deploy/docker-compose.example.yml](../deploy/docker-compose.example.yml)。

## 验证要求

基础验证：

```bash
make test
make secret-scan
git diff --check
```

功能验收必须匹配改动范围：

- Web UI 改动：启动服务，用浏览器实际点击、输入、上传、预览和轮询。
- MySQL 改动：运行 migration，并查询表确认数据落库。
- Gateway 改动：覆盖鉴权、scope、限流、timeout、脱敏和 audit。
- adapter 改动：实际启动 adapter，经 Gateway 调用成功，不能只用 mock 冒充真实接入。
- 生产只读验证：必须查到真实接口返回的可靠证据，并记录 evidence。
