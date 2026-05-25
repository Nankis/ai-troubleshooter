# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P056-001 | code | scheduler | 新增 `CaseScheduler`，process_case 记录 claim/finish | pass |
| EV-P056-002 | command | 单测/全测 | scheduler 单测和全量测试通过 | pass |
| EV-P056-003 | field | L3 全链路 | real adapter/Gateway/Agent Platform/Codex 决策 Agent/MySQL 跑通 `case_20260525_000068` 和 `case_20260525_000069` | pass |
| EV-P056-004 | ui | Web | Playwright 打开 Web 并验证 Brief/Agent Runs/Codex 可见 | pass |
| EV-P056-005 | fix | adapter bug | 修复 real adapter DATE_FORMAT 百分号转义和 source_meal_ids 类型 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-P056-002 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_case_scheduler.py apps/agent-platform/tests/test_agent_platform_fastapi.py` | pass | 35 targeted tests，修复后全量为 46 agent-platform tests |
| EV-P056-002 | 2026-05-25 | `make test` | pass | Go/Python/tests 全量通过 |
| EV-P056-002 | 2026-05-25 | `make secret-scan` | pass | 无敏感信息 |
| EV-P056-002 | 2026-05-25 | `git diff --check` | pass | 无输出 |
| EV-P056-002 | 2026-05-25 | `python3.13 scripts/validate_program.py programs/P-2026-052... programs/P-2026-056...` | pass | `validated 5 program(s)` |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P056-003 | 2026-05-25 | 服务启动 | real adapter `http://127.0.0.1:19086`、Go Gateway `http://127.0.0.1:18181`、Agent Platform `http://127.0.0.1:19191` | pass |
| EV-P056-003 | 2026-05-25 | 本地 Codex 决策 Agent | `/web/api/local-agents/probe` 返回 `probe_status=ok`，`provider=local_agent`，`model=codex` | pass |
| EV-P056-003 | 2026-05-25 | 查到真实数据 | `case_20260525_000068`：5 个 Gateway readonly tools 全成功，推荐记录 exists=true，用户 registered=true | pass |
| EV-P056-003 | 2026-05-25 | 查不到真实数据 | `case_20260525_000069`：5 个 Gateway readonly tools 全成功，推荐 exists=false，用户 registered=false | pass |
| EV-P056-003 | 2026-05-25 | scheduler 事件 | `case_20260525_000068` supervisor events 包含 `scheduler_claimed`、`scheduler_finished` | pass |
| EV-P056-004 | 2026-05-25 | Web UI | Playwright 打开 `http://127.0.0.1:19191/web`，点击 `case_20260525_000068`，检测 `Brief`、`recommendation_generation`、`codex`、`Agent Runs` 可见；截图 `artifacts/web-case-000068-brief.png` | pass |
| EV-P056-005 | 2026-05-25 | 真实链路暴露并修复 adapter bug | 首次真实调用发现 PyMySQL `DATE_FORMAT('%Y...')` 百分号未转义；第二次发现 `source_meal_ids` 数字数组和 Go schema 不匹配；均已修复并重跑通过 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 非法状态不会重复 claim | scheduler 单测 | EV-P056-002 | pass |
| MySQL Agent Run/Event 记录 scheduler claimed/finished | L3 全链路 | EV-P056-003 | pass |
| Web/API 验证能看到排查状态变化 | Web + API | EV-P056-003 / EV-P056-004 | pass |
| 不用 mock 或内存作为最终验收 | L3 全链路 | EV-P056-003 | pass |

## 未验证项

- Lark/Feishu 真实回调不在本轮范围。
- 生产 health-food 日志后台未配置，本轮日志工具走 real adapter 返回 0 samples，并明确记录没有 log upstream/local file。

## 已知噪音

- 本机已有历史非 canonical schema；本轮未新建，平台库使用 `ai_troubleshooter`，业务库使用已有 `meow_pas`。
