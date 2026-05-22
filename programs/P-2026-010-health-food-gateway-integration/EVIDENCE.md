# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | local service | T1 | health-food 能本地启动并可探活 | pass |
| EV-T2-001 | code review | T2 | 明确 Gateway/Connector 当前缺口 | pass |
| EV-T3-001 | implementation | T3 | health-food readonly adapter 和工具接入 | pass |
| EV-T4-001 | local flow | T4 | Web Chat 推荐缺失完整链路 | pass |
| EV-T4-002 | local flow | T4 | Web Chat AI 配额异常完整链路 | pass |
| EV-T4-003 | local flow | T4 | 缺 uid 时追问且不查下游 | pass |
| EV-T4-004 | browser | T4 | 浏览器页面真实提交并展示工具/决策日志 | pass |
| EV-T5-001 | docs | T5 | 业务服务注册 manifest 数据结构 | pass |
| EV-T6-001 | command | T6 | Go/Python 单测 | pass |
| EV-T6-002 | command | T6 | 格式检查 | pass |
| EV-T6-003 | command | T6 | secret scan | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | 启动 `health-food` JAR，覆盖本地 datasource、Redis 和 `server.port=18080` | pass | `/food-health/sys/alive` 返回 `0`；本地使用独立库 `hf_troubleshoot_codex`。 |
| EV-T1-002 | 2026-05-23 | 初始化 `health-food` DDL 到 `hf_troubleshoot_codex` | pass | 历史 DDL 中 `tb_payment_order` 有重复版本，本轮只取会员商品表创建后跑当前主 DDL 和后续 alter。 |
| EV-T3-001 | 2026-05-23 | `python3.13 -m py_compile scripts/mock-health-food-readonly-adapter.py` | pass | mock adapter 语法检查通过。 |
| EV-T6-001 | 2026-05-23 | `make test` | pass | Go `go test ./...` 和 Python decision-engine unittest 通过。 |
| EV-T6-002 | 2026-05-23 | `git diff --check` | pass | 无输出。 |
| EV-T6-003 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass | Secret scan passed。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-23 | `recommendation_missing`：Web Chat 提交 `health-food uid:hf_user_001 ... 今日推荐没有生成` | `/tmp/ai_troubleshooter_hf_recommendation_fixed.json` | case `case_20260523_000002` 进入 `NEED_HUMAN_CONFIRMATION`；调用用户资料、餐食、推荐状态、日志、相似 case；定位 `meal_data_fingerprint did not refresh after dinner upload`。 |
| EV-T4-002 | 2026-05-23 | `quota_exhausted`：Web Chat 提交 `health-food uid:hf_user_002 ... token账户余额不足` | `/tmp/ai_troubleshooter_hf_quota.json` | case `case_20260523_000003` 进入 `NEED_HUMAN_CONFIRMATION`；调用用户资料、AI 配额、日志、相似 case；定位 `daily_chat_count=30/30`。 |
| EV-T4-003 | 2026-05-23 | 缺少 uid：Web Chat 提交 `health-food ... 今日推荐没有生成` | `/tmp/ai_troubleshooter_hf_missing_uid.json` | case `case_20260523_000004` 进入 `WAITING_USER_REPLY`；`tool_count=0`，未查询下游。 |
| EV-T4-004 | 2026-05-23 | 浏览器打开 `http://127.0.0.1:18081/` 并真实提交 health-food 问题 | `/tmp/ai_troubleshooter_hf_web_chat.png` | 页面展示 case `case_20260523_000005`、domain `health_food`、工具调用和决策日志。 |
| EV-T4-005 | 2026-05-23 | Adapter 鉴权负向验证 | direct curl 摘要 | 缺少 Bearer token 调用 adapter 返回 401。 |
| EV-T4-006 | 2026-05-23 | Tool audit 落库验证 | MySQL 摘要 | `tb_troubleshoot_tool_call_audit` 有 15 条记录，包含 health-food 工具 `allowed` 记录。 |
| EV-T2-001 | 2026-05-23 | Connector / Gateway 契约检查 | 代码和文档摘要 | 现状是静态 registry + 标准 HTTP envelope；新增业务域需要工具 spec、scope、connector、policy 和 manifest。 |
| EV-T5-001 | 2026-05-23 | 业务服务注册数据结构 | `docs/business-service-registration.md`、`configs/business-capabilities.health-food.example.yaml` | 已定义 service、capability、auth、data classification、required params、response schema、failure modes。 |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| health-food 本地可运行 | T1 | EV-T1-001 | pass |
| 业务服务接入 Gateway 的缺口明确 | T2 | EV-T2-001 | pass |
| health-food 工具通过 Gateway 注册和授权 | T3 | EV-T3-001, EV-T4-001, EV-T4-002 | pass |
| mock 错误能走完整排障流程 | T4 | EV-T4-001, EV-T4-002, EV-T4-004 | pass |
| 缺字段时不打下游 | T4 | EV-T4-003 | pass |
| 业务服务注册数据结构已沉淀 | T5 | EV-T5-001 | pass |
| 单测和安全扫描通过 | T6 | EV-T6-001, EV-T6-002, EV-T6-003 | pass |

## 未验证项

- 未接真实 `health-food` readonly adapter；本轮使用 mock adapter 包装本地 `health-food` 探活和可控故障数据。
- 未接真实 Lark/Feishu bot；本轮使用 Web Chat。
- 未执行 `health-food` 自身完整业务接口登录态流程；只验证本地服务启动、探活和排障平台接入。

## 已知噪音

- 浏览器自动化过程中出现 Statsig 频率 warning，不影响本地 Web Chat 结果。
- macOS Netty DNS native library warning 出现在 `health-food` 启动日志，不影响本轮本地联调。
