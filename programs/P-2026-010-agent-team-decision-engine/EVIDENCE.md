# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | code review | T1 | Python 决策层现状和边界明确 | pass |
| EV-T2-001 | implementation | T2 | Agent Team 已实现 | pass |
| EV-T3-001 | test | T3 | Agent Team 单测覆盖核心路径 | pass |
| EV-T4-001 | command | T4 | 全量验证通过 | pass |
| EV-T4-002 | smoke | T4 | HTTP plan API 返回 agent reports 和 verification | pass |
| EV-T4-003 | security | T4 | secret scan 通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T3-001 | 2026-05-23 | `PYTHONPATH=apps/decision-engine python3.13 -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'` | pass | 8 个 Python decision-engine 单测通过。 |
| EV-T4-001 | 2026-05-23 | `make test` | pass | Go 全仓测试和 Python decision-engine 单测通过。 |
| EV-T4-001 | 2026-05-23 | `git diff --check` | pass | 无空白错误。 |
| EV-T4-003 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass | Secret scan passed。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | 读取 `apps/decision-engine` 和 Program 规范 | 代码审查摘要 | 现有 Python 决策层只有有限工具计划骨架；本轮作用域明确不改 Go Gateway。 |
| EV-T2-001 | 2026-05-23 | 实现 `decision_engine.agent_team` | `apps/decision-engine/decision_engine/agent_team.py` | Supervisor、Knowledge Agent、Kline Agent、Asset Agent、Fallback Agent 和 Verifier 已实现。 |
| EV-T4-002 | 2026-05-23 | 启动 Python 服务并调用 Kline plan | `/tmp/ai_troubleshooter_agent_team_kline.json` | 返回 `invoke_tools`，包含 `supervisor/knowledge_agent/kline_agent` reports 和 verifier accepted。 |
| EV-T4-002 | 2026-05-23 | 启动 Python 服务并调用知识直答 plan | `/tmp/ai_troubleshooter_agent_team_knowledge.json` | 返回 `answer_from_knowledge`，包含知识来源和 verifier accepted。 |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 不改 Go Gateway | T1 | EV-T1-001 | pass |
| Supervisor / Kline / Asset / Knowledge / Verifier 可用 | T2 | EV-T2-001 | pass |
| 单测覆盖多 agent 决策路径 | T3 | EV-T3-001 | pass |
| 验证和安全扫描通过 | T4 | EV-T4-001, EV-T4-003 | pass |

## 未验证项

- 未接入真实 LLM 多 agent 推理；本轮是规则型 Agent Team 基线。
- 未把 Go worker 切到 Python decision-engine；本轮保持 Go Gateway 和 Go worker 不动。

## 已知噪音

- 本地 HTTP smoke 结束时用 Ctrl-C 停止服务，终端输出 `KeyboardInterrupt`，不影响验证结论。
