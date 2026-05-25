# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| E-001 | 单测 | Agent Run API 和 case payload | case payload 返回 agent_runs；runtime 可注册和心跳 | 通过 |
| E-002 | migration | 008 DDL | 本地 canonical MySQL 可应用新表 | 通过 |
| E-003 | 真实服务 API | Agent Platform + MySQL | chat API 创建 case 后写入 agent run 和 events | 通过 |
| E-004 | 回归和安全扫描 | 全量验证 | `make test`、`make secret-scan`、`git diff --check` | 通过 |
| E-005 | 真实服务 API | 当前代码最终冒烟 | 启动 Agent Platform 后调用 `/api/v1/chat`，并查 MySQL | 通过 |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| E-001 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py` | 22 tests OK | 覆盖 `agent_runs` payload、runtime register、heartbeat。 |
| E-002 | 2026-05-25 | `bash -n scripts/mysql-migrate.sh` | 通过 | 语法检查通过。 |
| E-002 | 2026-05-25 | `MYSQL_DATABASE=ai_troubleshooter scripts/mysql-migrate.sh` | 通过 | 001-007 已存在，008 `agent_runtime_runs` 已应用到 canonical schema。密码通过环境变量传入。 |
| E-004 | 2026-05-25 | `make test` | 通过 | Go 全量通过；Decision Engine 18 tests OK；Agent Platform 33 tests OK；根目录 4 tests OK。 |
| E-004 | 2026-05-25 | `make secret-scan` | 通过 | `Secret scan passed (all).` |
| E-004 | 2026-05-25 | `git diff --check` | 通过 | 无 whitespace error。 |
| E-004 | 2026-05-25 | `MYSQL_DATABASE=ai_troubleshooter scripts/mysql-migrate.sh` | 通过 | 001-008 全部 skip，确认 migration 幂等。密码通过环境变量传入。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| E-003 | 2026-05-25 | 临时启动 Python Agent Platform，使用本地 MySQL 和 `local_rules`，调用 `/api/v1/chat` | 返回 `case_20260525_000050 WAITING_USER_REPLY`，response 保存到 `/tmp/ai_troubleshooter_p046_chat.json` | API 主路径可写入 case 和 agent run。 |
| E-003 | 2026-05-25 | 查询平台 MySQL | `tb_troubleshoot_agent_run` 最新 case 写入 4 条，`tb_troubleshoot_agent_run_event` 写入 10 条；agent 包含 `supervisor`、`knowledge_agent`、`health_food_agent`、`verifier` | Agent Run 生命周期实际落库。 |
| E-005 | 2026-05-25 | 当前工作树启动 Agent Platform，调用 `/api/v1/chat` | 返回 `case_20260525_000051 NEED_HUMAN_CONFIRMATION`，response 保存到 `/tmp/ai_troubleshooter_p046_final_chat.json`，payload 包含 4 个 agent runs | 当前代码可从 API 返回 Agent Run 轨迹。 |
| E-005 | 2026-05-25 | 查询平台 MySQL | `case_20260525_000051` 有 4 个 run、12 个 event；`supervisor` 9 个 event，`knowledge_agent`/`health_food_agent`/`verifier` 各 1 个 event | 当前代码可实际落库 Agent Run 生命周期。 |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 新 DDL 可应用到平台 MySQL | 增加 Agent Runtime / Run / Event DDL | E-002 | 通过 |
| Case API 可展示 run 轨迹 | repository/service/API 实现 | E-001, E-003, E-005 | 通过 |
| Decision Engine 主路径写入 run event | Decision Engine 埋点 | E-001, E-003, E-005 | 通过 |
| 文档说明 Multica 借鉴边界 | 更新架构和接入文档 | E-004 | 通过 |
| 全量回归、安全扫描和 diff 检查 | 增加验证 | E-004 | 通过 |

## 未验证项

- 未接入真实 Multica 服务，按本轮非目标处理。
- 未实现完整本地 runtime daemon，当前只提供平台契约、注册和心跳。
- 暂未接真实 Codex/Claude Code/Cursor runtime；当前只实现平台契约。

## 已知噪音

- E-003 使用 `local_rules` 验证生命周期落库，不代表真实大模型排障验收；本 Program 的目标是 Agent Run 生命周期和 runtime 契约。
- E-005 同样使用 `local_rules` 冒烟，只证明当前代码路径和 MySQL 落库，不证明真实模型判断质量。
