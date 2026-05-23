# FEATURE AUDIT

## 证据等级

| 等级 | 含义 |
| --- | --- |
| L0 | 文档/设计，没有运行验证 |
| L1 | 单测、schema、静态检查 |
| L2 | 本地 mock/fake/smoke 链路 |
| L3 | 本地真实依赖，例如 MySQL、真实本地业务服务、真实本地代码仓 |
| L4 | 预发/生产真实接口或真实外部平台 |

结论不能高于证据等级。使用 `mock`、`fake`、`memory`、`local_rules` 时必须显式降级。

## 功能矩阵

| 功能域 | 当前证据等级 | 证据 | 审计结论 |
| --- | --- | --- | --- |
| Program / 工作流治理 | L1 | `programs/README.md`、`docs/VERIFICATION.md`、P-2026-028 | 已有机制，但历史上仍发生先改代码后补 Program；本轮已强化 `AGENTS.md` 和 `docs/LESSONS.md`。 |
| MySQL store / migration | L3 | P-2026-027、P-2026-028 EV-T3-001/EV-T3-002 | 已改为 mysql 缺 DSN fail-fast；本地 migration、写入、查表、重启读取均通过。 |
| Web 工作台知识创建/预览/编辑/删除 | L3 | P-2026-028 EV-T3-001 | 本轮用 MySQL-backed Web 页面完成真实前端事件验证，并查表确认编辑和删除落库。 |
| root cause / feedback / knowledge evolution | L3 | P-2026-028 EV-T3-003 | root cause 回填后，MySQL 中 `tb_troubleshoot_root_cause`、`tb_troubleshoot_knowledge_item`、`tb_troubleshoot_knowledge_evolution_run` 均有记录。 |
| Web Chat 文本排障 | 平台数据 L3，业务证据 L2 | P-2026-028 EV-T3-002 | case/message/decision/tool audit 落 MySQL；业务证据来自内置 mock connector，不能称为真实业务验收。 |
| Web Chat 图片上传、粘贴、预览、放大 | L2 | P-2026-021、P-2026-024 | UI 行为用 memory/mock 运行验证；图片原图不持久化。真实视觉模型历史 smoke 见 P-2026-008，当前回归默认 local rules。 |
| Lark / 飞书事件入口 | L1 | `internal/lark/*_test.go`、P-2026-003 | payload、Feishu path、加密 callback、幂等、图片下载路径有单测；未接真实 bot 端到端，不能标 L4。 |
| Lark 图片下载 + Vision | L1/L2 | `internal/lark/bot_messenger_test.go`、`internal/vision/*_test.go`、P-2026-008 | 单测覆盖下载和 OpenAI-compatible image payload；真实 Qwen-VL smoke 是历史验证，不是当前持续回归。 |
| Gateway 鉴权、scope、限流、脱敏、审计 | L1/L2 | `internal/gateway/security_test.go`、P-2026-003 | 单测覆盖安全路径；prod smoke 历史通过。生产 mTLS、分布式限流、SIEM 仍属于部署层。 |
| Config-driven Gateway agent auth | L1 | `internal/config/validation_test.go`、P-2026-016 | agent token/env/config 校验有单测；新增业务 agent 不需要改代码。 |
| HTTP readonly connector | L1 | `internal/connectors/http_test.go`、P-2026-017 | envelope、Bearer、snake_case 参数有单测；公司真实接口需业务方提供后做 L3/L4 验收。 |
| 内置 mock connectors | L2 | `internal/connectors/mock.go`、多 Program smoke | 只用于 demo/契约/流程，不得作为真实业务证据。 |
| health-food 本地真实 adapter | L3 | P-2026-012 | 历史已用真实本地 health-food、真实本地 DB 和 adapter 验证；本轮没有重启 health-food，不把它升级为当前现场证据。 |
| health-food 生产日志 adapter | L2，生产 L4 未验收 | P-2026-017 | fake production API 只能证明链路；缺少生产 base URL、只读密钥、问题时间窗，真实生产验收未执行。 |
| MCP readonly adapter | L2 | P-2026-015、`tests/test_mcp_readonly_adapter.py` | adapter + mock MCP server + Gateway 链路通过；真实第三方 MCP server 需单独 L3/L4 验收。 |
| DMS MCP / CLI 接入 | L0/L1 | P-2026-019、`docs/dms-mcp-integration.md`、adapter tests | 已完成调研和 metadata route 示例；无 DMS 凭据和实例，未连接真实 DMS。 |
| Python Agent Team decision-engine | L1/L2 | P-2026-010-agent-team、`apps/decision-engine/tests/test_engine.py` | Supervisor/Specialist/Verifier 规则基线和 plan API 已验证；Go worker 未切到 Python，真实 LLM 多 agent 推理未实现。 |
| Local Code Debug / semantic code index | L1/L2，部分历史 L3 | P-2026-011、P-2026-013、P-2026-014 | 安全 allowlist、敏感跳过、语义索引有测试；真实主链路尚未在 Gateway 证据不足时自动二次调用。 |
| AI 决策日志、超时、工具调用限制 | L3 | P-2026-004、P-2026-028 EV-T3-002 | 单测覆盖失败限制和超时；本轮 Web Chat 写入 MySQL decision logs 和 tool audit。 |
| Secret scan / hooks | L1 | `scripts/secret-scan.py`、`githooks/*`、P-2026-008 | pre-commit/pre-push 和 `make secret-scan` 可用；敏感信息仍只能走环境变量或本机临时文件。 |

## 重点发现

1. 已确认严重问题：P-2026-027 前，平台知识手动录入曾用 memory store 验证，持久化结论不成立。已修复代码、文档和现场验证。
2. UI 近期多个 Program 使用 `DB_DRIVER=memory`。纯布局/交互 smoke 可以接受，但凡涉及平台数据持久化必须升级为 MySQL。P-2026-025 的知识预览/编辑/删除已在本轮补 MySQL 验证。
3. health-food、MCP、DMS、生产日志存在不同等级的“可用”：mock 链路、真实本地、真实生产必须分开写。当前 DMS 和生产日志都不能算真实生产可用。
4. Lark/飞书代码路径较完整，但真实 bot 端到端还没接公司凭据验收。README 已降级为“代码实现 + 本地 payload/unit 验证”。
5. Python Agent Team 已有规则基线，但生产主链路仍由 Go baseline 跑，本轮不应夸大为“Python 决策层已全面接管”。
