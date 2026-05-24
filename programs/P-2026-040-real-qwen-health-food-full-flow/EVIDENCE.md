# EVIDENCE

证据等级：L3。本轮使用本机真实 MySQL、真实本地 health-food 服务、真实 Go Gateway、真实 Python Agent Platform 和真实 Qwen 文本模型。未连接生产环境，未使用 mock adapter 作为验收结论。

原始 JSON、截图和 MySQL 查询输出保留在本地 `programs/P-2026-040-real-qwen-health-food-full-flow/evidence/`，该目录被 `.gitignore` 忽略，避免把本地业务数据和截图推到公开仓库。

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 验收点 | 结论 |
| --- | --- | --- | --- | --- |
| EV-SVC-001 | service | T2 | health-food 本地服务启动 | PASS |
| EV-SVC-002 | service | T3 | Go Gateway 通过 HTTP connector 调 health-food readonly | PASS |
| EV-SVC-003 | service | T4 | Python Agent Platform 使用 Qwen profile | PASS |
| EV-CASE-001 | API | T5/T7 | 平台经验命中，不调用 Gateway | PASS |
| EV-CASE-002 | API | T5/T7 | 平台经验未命中后调用 Gateway 查真实推荐错配 | PASS |
| EV-CASE-003 | API | T6/T7 | AI 配额数据查询，摘要脱敏 | PASS |
| EV-CASE-004 | API | T6/T7 | uid 不存在，要求反馈者确认 | PASS |
| EV-CASE-005 | API | T7 | Gateway 证据不足时本地代码辅助定位 | PASS |
| EV-WEB-001 | Browser | T8 | Web 工作台真实提交并走 Gateway | PASS |
| EV-DB-001 | MySQL | T9 | case/message/decision/context/tool audit 落库 | PASS |
| EV-FIX-001 | test | T10 | 分类、实时查证、脱敏、本地代码和 Gateway 单测 | PASS |
| EV-FIX-002 | final check | T10 | 完整测试、密钥扫描、diff 检查 | PASS |

## 命令验证

| Evidence ID | 命令 | 结果 |
| --- | --- | --- |
| EV-SVC-001 | health-food `mvn -pl health-food-srv -am -DskipTests package` 后以 local profile 启动 | PASS，`/food-health/sys/alive` 返回 200 |
| EV-SVC-001 | health-food readonly healthz，正确 token / 错误 token | PASS，正确 token 200，错误 token 401 |
| EV-SVC-002 | `curl http://127.0.0.1:18081/tools` | PASS，注册 14 个 tools，health-food tools 4 个 |
| EV-SVC-002 | Gateway 直接调用 `get_health_food_recommendation_status` | PASS，返回 `recommend_date=2026-05-23`、`job_status=source_date_mismatch` |
| EV-SVC-003 | `curl http://127.0.0.1:19091/healthz` | PASS，`llm_provider=openai_compatible`、`llm_model=qwen3.6-flash` |
| EV-FIX-001 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_classifier.py apps/agent-platform/tests/test_service_helpers.py apps/decision-engine/tests/test_engine.py` | PASS，26 tests |
| EV-FIX-001 | `go test ./internal/gateway` | PASS |
| EV-FIX-002 | `make test` | PASS，Go 全量测试、Python decision/agent/root tests 通过 |
| EV-FIX-002 | `make secret-scan` | PASS |
| EV-FIX-002 | `git diff --check` | PASS |
| EV-FIX-002 | health-food `git diff --check` | PASS |
| EV-FIX-002 | health-food `JAVA_HOME=$(/usr/libexec/java_home -v 23) mvn -pl health-food-srv -am -DskipTests package` | PASS，默认 Maven 使用 JDK 25 时失败，指定 JDK 23 后构建成功 |

## 场景验证

| Evidence ID | Case | 问题 | 结果 |
| --- | --- | --- | --- |
| EV-CASE-001 | `case_20260525_000026` | health-food 餐食重复导致热量翻倍，应该优先怎么判断 | 命中 `knowledge:4`，无 Gateway tool calls |
| EV-CASE-002 | `case_20260525_000028` | uid `2054603630081875968`，`2026-05-23` 推荐结果不准，怀疑旧餐食来源 | Gateway 入参含 `recommendation_date=2026-05-23`；health-food 返回 `source_date_mismatch` |
| EV-CASE-003 | `case_20260525_000030` | 同一 uid AI 对话次数接近上限但还能继续使用 | Gateway 返回 `daily_chat=996/1000`，结论为正常；token 余额在数据和摘要中均脱敏 |
| EV-CASE-004 | `case_20260525_000031` | uid `12345` 每日推荐缺失 | health-food profile 返回 `registered=false`，Agent 要求反馈者确认正确 uid |
| EV-CASE-005 | `case_20260525_000033` | `debug_local_code=true` 且 Gateway 证据不足 | Local Code Agent 扫描 345 个文件，命中 8 个文件；回复列出相对文件、行号和分析模式 |
| EV-WEB-001 | `case_20260525_000035` | Web 端提交“请查真实数据，确认每日推荐来源餐食是否日期错配” | 浏览器实际提交；页面展示 `source_date_mismatch`；截图保存在本地 evidence 目录 |
| EV-DB-001 | MySQL 查询 | 反查 case/message/decision/context/tool audit | `tb_troubleshoot_case_message`、`tb_troubleshoot_ai_decision_log`、`tb_troubleshoot_context_ledger`、`tb_troubleshoot_tool_call_audit` 均有记录 |

## 发现并修复的问题

- health-food readonly controller 被自定义登录拦截器拦截，已在 health-food 仓库排除 `/food-health/v1/readonly/**`。
- Qwen 输出 `health-food/token_usage/数据异常` 等不稳定 taxonomy，已做领域归一和规则 taxonomy 保护。
- Qwen 抽取 `date` 未映射到 `recommendation_date`，导致查错日期，已修复。
- Gateway 配额摘要泄漏 token 余额，已修复并补单测。
- Web “查真实数据”被经验库短路，已把显式日期和真实数据诉求纳入 realtime gate。
- Local Code Agent 回复缺少可操作定位，已改为输出 top 文件、行号、命中词。

## 覆盖映射

| 验收标准 | Evidence ID | 状态 |
| --- | --- | --- |
| 启动所有相关服务 | EV-SVC-001/002/003 | PASS |
| Python 端真实走 Qwen | EV-SVC-003/EV-CASE-* | PASS |
| 平台经验查得到 | EV-CASE-001 | PASS |
| 平台经验查不到后查 Gateway | EV-CASE-002/003/004/EV-WEB-001 | PASS |
| Gateway 实际调用 health-food readonly 接口 | EV-SVC-002/EV-CASE-002/003/004/EV-WEB-001 | PASS |
| DB 查询类问题 | EV-CASE-002/003/004 | PASS |
| 需要代码辅助的问题 | EV-CASE-005 | PASS |
| Web 端实际点击/提交 | EV-WEB-001 | PASS |
| MySQL 持久化反查 | EV-DB-001 | PASS |

## 未验证项

- 未接真实 Lark/飞书回调，本轮范围明确排除。
- 未连接 health-food 生产环境，本轮为 L3 本地真实依赖验收。
- 未执行 health-food 写操作、任务重跑或业务修复，只验证 readonly 排障链路。

## 已知噪音

- `case_20260525_000025`、`case_20260525_000027`、`case_20260525_000029`、`case_20260525_000034` 是修复前失败/半失败证据，不作为最终 PASS 结论。
- health-food `mvn -DskipTests package` 有 MapStruct warning，但构建成功。
