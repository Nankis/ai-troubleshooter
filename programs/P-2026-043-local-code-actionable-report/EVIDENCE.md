# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-CODE-001 | 代码审查 | T1/T2 | 本地代码报告包含文件、方法、行范围、疑点、下一步和有界摘录 | pass |
| EV-UNIT-001 | 单测 | T3 | LocalCode evidence 结构和 verifier 检查未破坏 | pass |
| EV-UNIT-002 | 单测 | T2/T3 | Agent Platform 本地代码报告格式和日志压缩 helper 可用 | pass |
| EV-L3-API-001 | 本地真实依赖 | T4 | MySQL + 真实本地 health-food 源码映射下 API 返回可操作代码报告 | pass |
| EV-L3-DB-001 | 本地真实依赖 | T4 | 决策日志落 MySQL 且不存整段源码 | pass |
| EV-L3-WEB-001 | Web UI | T4 | Web 端打开 case 后能看到方法名、代码行和行范围 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-UNIT-001a | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m py_compile apps/decision-engine/decision_engine/local_code.py apps/decision-engine/decision_engine/agent_team.py` | pass | Python 编译通过。 |
| EV-UNIT-001b | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py` | pass | 18 tests OK。 |
| EV-UNIT-002 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_service_helpers.py` | pass | 3 tests OK。 |
| EV-REG-001 | 2026-05-25 | `make test` | pass | Go tests、decision-engine 18 tests、agent-platform 26 tests、root tests 4 tests OK。 |
| EV-REG-002 | 2026-05-25 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-REG-003 | 2026-05-25 | `git diff --check` | pass | 无 whitespace error。 |
| EV-L3-API-001a | 2026-05-25 | 启动 Agent Platform：`DB_DRIVER=mysql DB_DSN=<redacted> CONNECTOR_MODE=mock ... LOCAL_CODE_REPOS_JSON=<health-food 本地源码路径> .venv/bin/python -m agent_platform` | pass | 服务运行在 `127.0.0.1:19091`，平台持久化为 MySQL，本地代码仓为真实 health-food 源码。 |
| EV-L3-API-001b | 2026-05-25 | `POST /api/v1/chat`，消息包含 `debug_local_code=true gateway_evidence_status=insufficient service_name=health-food suspect_area=RecommendFoodJob FoodServiceImpl meal_data_fingerprint ...` | pass | 生成 `case_20260525_000049`，状态 `NEED_HUMAN_CONFIRMATION`。 |
| EV-L3-DB-001 | 2026-05-25 | `mysql ai_troubleshooter` 查询 `case_20260525_000049` 的 `orchestrator_plan` | pass | `output_snapshot_json` 长度 13390；local_code evidence 无 `code_excerpt` 字段，有 `code_excerpt_line_count=14`。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-CODE-001 | 2026-05-25 | LocalCodeInspector 直接扫描真实 health-food 本地仓 | Top hits 包含 `RecommendFoodJob.refreshDailyRecommend (119-132)`、`FoodServiceImpl.generateDailyFoodRecommendWithFingerprint (486-499)`、`HealthFoodReadonlyTroubleshooterService.recommendationStatus (218-224)`。 | pass |
| EV-L3-API-001 | 2026-05-25 | API 完整链路 | 回复前四条包含 `MealDataFingerprintUtil.calculateFingerprint`、`RecommendFoodJob.refreshDailyRecommend`、`FoodServiceImpl.generateDailyFoodRecommendWithFingerprint`、`HealthFoodReadonlyTroubleshooterService.recommendationStatus`，且展示相关代码行。 | pass |
| EV-L3-WEB-001 | 2026-05-25 | In-app browser 打开 `http://127.0.0.1:19091/web` 并点击 `case_20260525_000049` | 页面文本检查：`hasLocalReport=true`、`hasFoodService=true`、`hasCodeLine=true`、`hasRecommendJob=true`、`hasLineRange=true`；截图保存 `/tmp/p043_local_code_web_report.png`。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 报告不再只是 `path:line(term)`，必须有方法、行范围、疑点和下一步 | T1/T2 | EV-CODE-001, EV-L3-API-001 | pass |
| 代码摘录有界且只在 debug-only + allowlist 下返回 | T1 | EV-UNIT-001, EV-L3-API-001 | pass |
| 决策日志不存整段源码，避免撑爆 MySQL 字段和泄漏 | T2 | EV-L3-DB-001 | pass |
| Web 页面实际可打开并展示报告 | T4 | EV-L3-WEB-001 | pass |
| 回归、secret scan、diff check 通过 | T5 | EV-REG-001, EV-REG-002, EV-REG-003 | pass |

## 未验证项

- 本轮使用 `LLM_PROVIDER=local_rules`，只验证本地代码辅助排查报告结构和平台链路，不宣称真实 Qwen/GPT/Claude 推理验收。
- 本轮 Gateway 为 `CONNECTOR_MODE=mock`，因为验证目标是 Gateway 证据不足后的本地代码最后手段；不宣称真实生产 readonly adapter 证据验收。
- 本轮仍是 lightweight/cross_module_resolver，不是完整 LSP/LSIF 精确语义引擎。

## 已知噪音

- Web 左侧已有多条历史同名 debug case；本轮明确使用 `case_20260525_000049` 做页面验证。
- 截图保存在 `/tmp`，不提交二进制截图到仓库。
