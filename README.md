# ai-troubleshooter

业务问题排查 Agent 平台。一期目标是把 Lark/飞书、Web Chat、截图和简短用户反馈统一转成可审计的排障 case，让 Agent 先复用平台经验，再按预算调用受控只读工具，快速定位生产问题。

当前仓库采用 monorepo：

- Go 1.24+：入口、Case API、Web Chat、Gateway、worker、平台数据落库。
- Python 3.13：`apps/decision-engine` 决策层，承接 Supervisor、多 specialist agent、工具计划、Verifier、后续 RAG 和本地代码辅助排查。
- MySQL：平台 case、消息、AI 决策日志、tool audit、root cause、knowledge item 和自进化记录。

Go 里的 `internal/decisionbaseline` 只作为 phase-0 本地 fallback，不是目标决策层。

## 核心边界

- Agent 不可信，Gateway 可信：Agent 只做理解、规划和总结；生产证据查询必须走 Investigation Gateway。
- 平台数据归平台：case、消息、经验、审计、AI 决策日志属于 Agent 平台，不需要业务方提供 query gateway。
- 业务方只提供只读证据：日志、行情、资产、风控、缓存、发布记录等都通过 readonly adapter 或 MCP readonly adapter 注册到 Gateway。
- 信息不足先追问：用户最多只需要提供自然语言、截图、uid、订单号、时间范围等线索，不要求懂内部字段名。
- 每次排查都可复盘：工具调用、AI 为什么这样判断、为什么停止、人工根因和知识沉淀都会落库。
- 生产安全 fail-closed：Gateway 鉴权、scope、限流、timeout、脱敏、审计、控制面鉴权和 Lark allowlist 都支持配置化。

## 一期架构

完整部署图和流程图见 [docs/architecture-decisions.md](docs/architecture-decisions.md)。README 只保留入口视图：

```mermaid
flowchart LR
  Input["Lark / 飞书 / Web Chat<br/>文字 + 图片"] --> Intake["Channel Adapter<br/>验签 / 解密 / 图片识别"]
  Intake --> Case["Case API + Queue<br/>幂等 / 状态机 / 消息"]
  Case --> Engine["Python Decision Engine<br/>追问 / 经验评分 / 工具预算 / 总结"]
  Engine --> Knowledge["Platform Knowledge<br/>历史 case / root cause / SOP"]
  Engine --> Model["Platform LLM / Vision"]
  Engine --> Store[("Platform MySQL<br/>tb_troubleshoot_*")]
  Knowledge --> Store
  Engine -- "低置信 / 需实时证据" --> Gateway["Investigation Gateway<br/>鉴权 / scope / 限流 / timeout / 审计 / 脱敏"]
  Gateway --> Tools["注册只读工具<br/>logs / market / asset / risk / MCP"]
  Tools --> Business["业务 readonly adapters"]
  Business --> BusinessDB[("业务侧数据")]
  Engine -- "高置信经验" --> Case
  Case --> Input
```

## 当前状态

| 模块 | 状态 |
| --- | --- |
| Web Chat 工作台 | 已实现，支持文字、图片粘贴上传、图片预览、case 列表、进度面板、工具分组、知识预览/编辑。 |
| Lark / 飞书入口 | 代码实现，支持 token、encrypted callback、图片下载和平台差异配置；真实 bot 需要公司凭据验收。 |
| Case / Knowledge / Audit | MySQL 持久化；`DB_DRIVER=mysql` 时没有 `DB_DSN` 会直接失败，避免误用内存。 |
| Decision Engine | Python 目标层已提供 Supervisor、Kline、Asset、Knowledge、Local Code、Verifier 轻量基线；Go fallback 仍可跑本地闭环。 |
| Investigation Gateway | 已实现 Bearer、agent/scope/tool/chat allowlist、限流、timeout、审计、脱敏和配置化 agent。 |
| 业务接入 | 支持 mock、标准 HTTP readonly adapter、MCP readonly adapter、health-food 本地真实 adapter 和生产日志桥接方案。 |
| 本地代码辅助 | debug-only，按服务名和仓库 allowlist 检索符号、调用边、receiver type、接口实现关系，不返回源码片段。 |

详细能力清单和历史验收记录请看 [programs/README.md](programs/README.md) 以及各 `programs/P-*` 的 `RESULT.md` / `EVIDENCE.md`。

## 快速启动

需要 Go 1.24+、Python 3.13 和 MySQL。敏感信息只允许通过环境变量传入。

```bash
go version
python3.13 --version

MYSQL_HOST=127.0.0.1 \
MYSQL_PORT=3306 \
MYSQL_USER=root \
MYSQL_PASSWORD="$LOCAL_MYSQL_PASSWORD" \
MYSQL_DATABASE=ai_troubleshooter \
make migrate-mysql

export DB_DRIVER=mysql
export DB_DSN="$LOCAL_DB_DSN"
export CONNECTOR_MODE=mock
export LLM_PROVIDER=local_rules
export VISION_PROVIDER=local_rules
export HTTP_PORT=8080
make dev
```

打开 `http://localhost:8080/web`。本地开发如果端口冲突，可以改 `HTTP_PORT`，例如 `HTTP_PORT=18088 make dev`。

更完整的本地运行、Web Chat、模型、health-food、MCP、DMS 和容器命令已经移到 [docs/local-runbook.md](docs/local-runbook.md)。

提交前建议安装 hook 并扫描敏感信息：

```bash
make install-hooks
make secret-scan
```

## 目录速览

```text
api/openapi/               Case、Decision Engine、Gateway OpenAPI 草案
apps/decision-engine/      Python 3.13 决策层
cmd/                       dev-server、lark-bot、worker、investigation-gateway
configs/                   配置样例、Gateway agent、业务能力和 MCP route 示例
deploy/                    Docker Compose 示例
docs/                      架构、安全、接入、运行、验证和经验沉淀文档
internal/                  Go 入口、Gateway、case、storage、LLM、vision、worker 等实现
migrations/                MySQL 表结构
programs/                  Program 记录、验收证据、复盘和交付结果
scripts/                   migration、secret scan、adapter、MCP、hook 脚本
web/                       内置 Web Chat 静态页面
```

## 文档地图

| 主题 | 文档 |
| --- | --- |
| 开发规则 | [AGENTS.md](AGENTS.md), [docs/ai-workflow.md](docs/ai-workflow.md), [docs/VERIFICATION.md](docs/VERIFICATION.md), [docs/LESSONS.md](docs/LESSONS.md) |
| 架构与边界 | [docs/architecture-decisions.md](docs/architecture-decisions.md), [docs/phase1.md](docs/phase1.md) |
| 本地运行 | [docs/local-runbook.md](docs/local-runbook.md), [docs/web-workbench.md](docs/web-workbench.md), [apps/decision-engine/README.md](apps/decision-engine/README.md) |
| 业务接入 | [docs/ai-connector-integration.md](docs/ai-connector-integration.md), [docs/business-service-registration.md](docs/business-service-registration.md), [configs/business-capabilities.health-food.example.yaml](configs/business-capabilities.health-food.example.yaml) |
| 安全与控制 | [docs/gateway-security.md](docs/gateway-security.md), [docs/decision-logging-and-limits.md](docs/decision-logging-and-limits.md), [docs/deployment-checklist.md](docs/deployment-checklist.md) |
| MCP / DMS | [docs/mcp-gateway-adapter.md](docs/mcp-gateway-adapter.md), [docs/dms-mcp-integration.md](docs/dms-mcp-integration.md) |
| health-food | [docs/health-food-production-integration.md](docs/health-food-production-integration.md) |
| 经验沉淀 | [docs/knowledge-evolution.md](docs/knowledge-evolution.md), [api/openapi/case-knowledge-api.yaml](api/openapi/case-knowledge-api.yaml) |
| API | [api/openapi/decision-engine.yaml](api/openapi/decision-engine.yaml), [api/openapi/investigation-gateway.yaml](api/openapi/investigation-gateway.yaml), [api/openapi/case-knowledge-api.yaml](api/openapi/case-knowledge-api.yaml) |

## 验证

```bash
make test
make secret-scan
git diff --check
```

涉及前端、Gateway、MySQL、真实 adapter 或生产只读接口的改动，不能只算 mock 通过。验收必须实际启动服务、从入口调用成功，并在对应 Program 或文档里记录命令、结果和证据等级。

## 当前边界

- Redis Stream 仍未替换内存队列，接口已预留。
- 真实业务接入需要业务方提供 readonly adapter 或 MCP server，并按 Gateway manifest 注册。
- 生产问题排查不允许 Agent 直连生产 DB；确需 DB 证据时优先用 DMS MCP 元数据或 named readonly query adapter。
- Lark/飞书真实端到端需要公司 bot 凭据、回调地址和 allowlist 配置。
- 图片默认只做短暂下载并传给视觉模型，原图不持久化；如需留存，应接公司对象存储和数据分级策略。

## 开源许可与贡献

本项目使用 Apache License 2.0，适合企业内部二次开发、私有化部署和按需封装业务 adapter。贡献前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)，安全问题按 [SECURITY.md](SECURITY.md) 私下披露。
