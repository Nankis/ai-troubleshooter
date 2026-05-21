# ai-troubleshooter

一期业务工单排障 Agent 平台。当前仓库按 TRD 建议的 Go 在线主链路组织，先跑通只读、权限可控、可审计的 MVP。

## 为什么做这个

线上业务问题经常从客服工单、Lark 群、截图和简短描述进入研发排查流程。典型输入并不完整，比如“余额变少了”“K线不对”“数据不对”，人工排查需要反复追问、查日志、查缓存、核对业务数据和外部交易所。这个项目的目标不是替代 SRE 平台，也不是让 Agent 自动修复生产，而是先把“用户反馈类业务故障”的排查过程 case 化、工具化、审计化，并逐步沉淀成可复用经验库。

一期先把生产可控性放在第一位：Agent 可以推理和编排，但不能直接拥有生产权限；所有查询必须走受控网关；所有工具只读；所有调用留审计；信息不足时先问人。

## 一期范围

- Lark 群消息创建排障 case。
- Agent 先做分类、实体抽取和必要字段检查，信息不足时追问。
- 信息足够后通过 Tool Server / Query Gateway 调用只读工具。
- 工具调用默认 deny，只有注册 agent、授权 scope、启用工具才可执行。
- 每次工具调用写 audit，返回前统一脱敏。
- K线/行情异常和资产/余额异常先接 mock connector，后续替换真实只读 API。

## 设计思路

### 1. Agent 不可信，Gateway 可信

Agent 负责理解问题、抽取字段、判断是否需要追问、决定调用哪些工具和总结证据。但 Agent 不直接连接生产 DB、Redis、日志系统或业务服务。所有生产查询都必须经过 Investigation Gateway。

Gateway 是生产只读查询门禁，负责：

- 校验 agent 身份和授权 scope。
- 校验 Lark 用户、群和工具权限。
- 限制时间范围、limit、工具调用超时。
- 统一脱敏。
- 统一审计。
- 只允许调用注册过的只读工具。

这样即使 LLM 输出不稳定，也不会把权限判断交给模型。

### 2. 信息不足先问人

排障的第一步不是查生产，而是判断最小必要字段是否齐全。比如 K线问题至少需要 `symbol`、`interval`、`abnormal_time`、`issue_type`；资产问题至少需要 `user_id` 或 `account_id`、`asset_symbol`、`abnormal_time`、`issue_type`。

字段不足时，Agent 最多追问 3 个关键问题，并把 case 状态推进到 `WAITING_USER_REPLY`。字段足够后才进入工具查询阶段。

### 3. Case 状态机管理生命周期

每个用户反馈都会被创建成独立 case，并通过状态机推进：

```text
NEW
  -> NEED_MORE_INFO
  -> WAITING_USER_REPLY
  -> READY_TO_INVESTIGATE
  -> INVESTIGATING
  -> WAITING_TOOL_RESULT
  -> NEED_HUMAN_CONFIRMATION
  -> DONE / FAILED / CANCELLED
```

状态更新带版本号，后续接 MySQL 时用乐观锁避免多个 worker 同时处理同一个 case。

### 4. 事件驱动并发

Lark Bot 只负责接收事件、创建 case、发送即时回复和投递队列，不做复杂推理，也不查生产。Agent Worker 从队列消费 case event，用 worker pool 并发处理多个客服问题。队列目前是内存实现，接口已经抽象，后续替换为 Redis Stream。

### 5. Tool Server 和 Query Gateway 一期合并，边界不混

TRD 里 Tool Server 和 Query Gateway 是两个逻辑层。一期为了更快跑通，在 `investigation-gateway` 进程内同时实现：

- Tool Registry：注册工具描述、入参 schema、scope、handler。
- Tool Invoke API：`GET /tools` 和 `POST /tools/{tool_name}/invoke`。
- Policy Engine：默认拒绝，只允许注册 agent 和授权 scope。
- Audit：记录每次工具调用。
- Masking：返回前脱敏。
- Connectors：对接业务只读 API、日志、缓存、外部交易所。

代码层面仍然保持包边界，后续可以把 Tool Server 和 Query Gateway 拆成两个服务，或者补 MCP adapter。

### 6. 业务接入优先只读 API

Gateway 底层优先接业务服务提供的只读 API，而不是让 Agent 或 Gateway 自由 SQL。确实需要直查 DB 时，只允许走预注册 SQL 模板、read replica、参数化查询、强制 limit 和 timeout。

当前实现先用 mock connector：

- K线：内部 K线、外部交易所对比、缓存状态、行情源状态。
- 资产：资产快照、资产事件流、用户近期错误。
- 通用：日志摘要、发布记录、历史相似 case。

### 7. 每次排查都沉淀

一期数据表围绕 case、实体、消息、investigation、tool audit、root cause、knowledge item 设计。即使 AI 没查准，也要保留原始问题、抽取字段、调用过程、AI 判断和人工最终根因。失败样本同样是后续优化 prompt、工具和知识库的材料。

## 总体架构

```text
Lark 群聊
  -> lark-bot
  -> queue
  -> agent-worker
  -> agent-orchestrator
  -> investigation-gateway
       -> policy / audit / masking / tool registry
       -> readonly connectors
       -> business services / logs / cache / external exchange
```

本地 MVP 用 `cmd/dev-server` 把这些模块合并在一个进程里，方便先验证闭环。部署时可以按 TRD 拆成 `lark-bot`、`orchestrator`、`worker`、`investigation-gateway` 四个服务。

## 目录

```text
cmd/
  dev-server/              本地一体化调试入口
  lark-bot/                Lark 事件入口
  orchestrator/            Agent 编排服务
  worker/                  case event worker
  investigation-gateway/   Tool Server + Query Gateway
internal/
  caseflow/                case 模型、状态机、内存 store
  lark/                    Lark 事件 handler 和消息发送抽象
  llm/                     LLM 抽象和规则型本地实现
  orchestrator/            case 处理主流程
  queue/                   可替换队列接口和内存实现
  tool/                    Tool Spec、Registry、Invocation 模型
  gateway/                 Tool API、policy、audit、masking、connector 编排
  policy/                  默认拒绝策略
  audit/                   工具调用审计
  masking/                 脱敏
  connectors/              K线、资产、日志 mock connector
api/openapi/               HTTP API 草案
configs/                   配置样例
migrations/                MySQL 初始化表
docs/                      TRD 摘要与一期说明
```

关键文档：

- [AI 接入规范：业务只读接口封装](docs/ai-connector-integration.md)
- [Gateway 安全与鉴权边界](docs/gateway-security.md)
- [AI 决策日志与查询限制](docs/decision-logging-and-limits.md)
- [部署检查清单](docs/deployment-checklist.md)
- [经验沉淀与自进化闭环](docs/knowledge-evolution.md)
- [ai-workflow 开发规范接入](docs/ai-workflow.md)

## 已实现能力

- 独立 Go 仓库和一期目录结构。
- 本地一体化 `dev-server`。
- Lark 事件入口：`POST /lark/events`，支持本地模拟 payload 和 Lark v2 消息 payload。
- Lark verification token 和 allowed chat 基础门禁。
- Lark `source + message_id` 幂等去重，平台重复投递不会重复创建 case 或重复入队。
- 配置 `LARK_APP_ID` / `LARK_APP_SECRET` 后，Bot 会通过飞书开放平台发送文本回复；未配置时本地只写日志。
- Case 创建、状态流转、消息和实体记录。
- Worker pool 消费 case event。
- LLMClient 抽象和规则型本地实现。
- AI 决策日志：分类、实体抽取、字段检查、工具计划、工具调用、总结、失败原因、重复处理跳过原因、陈旧处理中状态收敛原因均可审计，快照写入前会统一脱敏。
- Case 级排查超时、工具调用总数上限和工具失败上限，避免查不到问题时持续打下游。
- Orchestrator 处理前先认领 case；重复 worker、重复事件或终态 case 会安全跳过，不再查询下游；陈旧处理中状态会恢复或失败收敛。
- Tool Registry 和内部 Tool Invoke API。
- Query Gateway 默认拒绝策略、scope 校验、参数边界控制。
- Gateway HTTP Bearer 鉴权、认证 agent 与请求 `agent_id` 绑定、agent/user/tool 固定窗口限流。
- 控制面 API Bearer 鉴权，生产环境缺少关键安全配置时 fail-closed。
- Audit sink、MySQL tool audit 持久化和脱敏。
- 10 个一期只读工具。
- K线、资产、日志 mock connector。
- 标准 HTTP 只读 connector，可按文档对接公司接口。
- 人工 root cause 回填、case feedback、knowledge item 自进化和 evolution run 记录。
- MySQL store：配置 `DB_DSN` 后 case、消息、根因、反馈、知识库和自进化运行记录持久化；不配置时本地自动使用内存 store。
- MySQL 初始化 migration。
- 知识沉淀增强 migration。
- AI 决策日志 migration。
- 事件幂等索引 migration。
- OpenAPI 草案。
- 单元测试覆盖状态机、policy、masking、tool registry、HTTP connector envelope、Lark payload、知识自进化。

## 本地启动

本机如果 `go` 不在 PATH，可以临时使用：

```bash
export PATH="/Users/ginseng/sdk/go1.26.2/bin:$PATH"
```

启动一体化 dev server：

```bash
go run ./cmd/dev-server
```

模拟 Lark 事件：

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
    "lark_user_id":"ou_dev",
    "chat_id":"oc_dev",
    "arguments":{"user_id":"user_123","asset_symbol":"USDT","at_time":"2026-05-21T20:00:00+08:00"}
  }'
```

对接公司只读 adapter：

```bash
CONNECTOR_MODE=http
CONNECTOR_API_KEY=replace-with-internal-token
MARKET_READONLY_BASE_URL=https://market-readonly.internal
ASSET_READONLY_BASE_URL=https://asset-readonly.internal
OPS_READONLY_BASE_URL=https://ops-readonly.internal
```

adapter 需要实现的接口见 [docs/ai-connector-integration.md](docs/ai-connector-integration.md)。

回填根因并触发知识自进化：

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

查询某个 case 的 AI 决策轨迹：

```bash
curl -s 'localhost:8080/cases/case_20260521_000001/ai-decisions?limit=100'
```

## 容器部署

构建本地一体化服务：

```bash
docker build --build-arg SERVICE=dev-server -t ai-troubleshooter:dev .
docker run --rm -p 8080:8080 --env CONNECTOR_MODE=mock ai-troubleshooter:dev
```

构建独立 gateway：

```bash
docker build --build-arg SERVICE=investigation-gateway -t ai-troubleshooter-gateway:dev .
```

compose 示例见 [deploy/docker-compose.example.yml](deploy/docker-compose.example.yml)。

## 验证

```bash
go test ./...
```

## Gateway 安全边界

平台内已实现 Gateway 入口安全：`GATEWAY_AUTH_ENABLED=true` 后，`POST /tools/{tool}/invoke` 必须携带 Bearer token；`GATEWAY_BEARER_TOKENS` 用 `agent_id:token` 配置，并把认证 agent 与请求体 `agent_id` 强绑定，防止调用方伪造其它 agent。Gateway 还内置工具默认拒绝、scope 校验、时间范围/limit 约束、调用 timeout、agent/user/tool 固定窗口限流、审计持久化和返回脱敏。

root cause、feedback、knowledge、orchestrator case/process 这类控制面 API 通过 `CONTROL_API_AUTH_ENABLED=true` 和 `CONTROL_API_BEARER_TOKENS` 单独鉴权。`APP_ENV=prod` 时，Gateway、控制面 API、Lark verification token 和 allowed chats 缺失会直接启动失败。

Agent 编排层不是无限循环查询：`MAX_INVESTIGATION_SECONDS` 控制单 case 总耗时，`MAX_TOOL_CALLS_PER_CASE` 控制工具调用总数，`MAX_TOOL_FAILURES_PER_CASE` 控制连续失败后停止继续查下游。每个关键决策都会写入 `ai_decision_logs`，快照入库前脱敏，可以复盘“为什么这么判断、为什么选这些工具、为什么停止”。Lark 入口用 `source + message_id` 幂等去重，Orchestrator 处理前先认领 case，重复事件或重复 worker 只记录 `process_skipped`，不会重复打下游；陈旧处理中状态会恢复或失败收敛。

部署层仍建议加上 mTLS、内网 ACL、Ingress allowlist 或 service mesh 策略；多实例生产限流可接 Redis、Envoy 或公司 API Gateway，审计日志也建议落到统一日志或安全审计平台。

## 当前实现边界

- Redis Stream、真实日志/DB/Redis connector 尚未接入，已保留接口。
- LLM 默认是规则型本地实现，方便本地跑通；接真实模型时实现 `internal/llm.LLMClient`。
- Gateway 已按一期原则实现入口鉴权、身份绑定、默认拒绝、只读工具、scope 校验、时间范围/limit 约束、限流、审计和脱敏。
- Orchestrator 一期采用有限工具计划，不做无限自主循环；后续如引入多轮 ReAct，需要继续复用当前 timeout、tool call budget 和 decision log。
- 公司只读接口可通过标准 HTTP connector 接入；如接口字段不同，应写 adapter 做映射。
- 飞书事件回调加密和图片下载还未接入；内部联调先关闭回调加密。

## 下一步

1. 把 `queue.Queue` 从内存实现替换为 Redis Stream。
2. 接真实 Lark 回调加密验签和图片下载。
3. 接真实 LLM provider，并保留规则型实现作为本地 fallback。
4. 用真实业务只读 API 替换 mock connector。
5. 把 MySQL tool audit / decision logs 同步到统一日志或 SIEM。
