# ai-troubleshooter

一期业务工单排障 Agent 平台。当前仓库按 TRD 建议的 Go 在线主链路组织，先跑通只读、权限可控、可审计的 MVP。

## 一期范围

- Lark 群消息创建排障 case。
- Agent 先做分类、实体抽取和必要字段检查，信息不足时追问。
- 信息足够后通过 Tool Server / Query Gateway 调用只读工具。
- 工具调用默认 deny，只有注册 agent、授权 scope、启用工具才可执行。
- 每次工具调用写 audit，返回前统一脱敏。
- K线/行情异常和资产/余额异常先接 mock connector，后续替换真实只读 API。

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

## 验证

```bash
go test ./...
```

## 当前实现边界

- Redis Stream、MySQL、真实 Lark API、真实日志/DB/Redis connector 尚未接入，已保留接口。
- LLM 默认是规则型本地实现，方便本地跑通；接真实模型时实现 `internal/llm.LLMClient`。
- Gateway 已按一期原则默认拒绝、只读工具、scope 校验、时间范围/limit 约束、审计和脱敏。
