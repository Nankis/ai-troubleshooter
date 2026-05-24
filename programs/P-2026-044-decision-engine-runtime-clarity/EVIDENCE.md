# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-CODE-001 | 代码审查 | T2/T3 | Web Chat 通过 Agent Platform 调用 Decision Engine | pass |
| EV-UNIT-001 | 单测 | T3 | Web Chat 必须调用注入的 `decision_engine.plan()` | pass |
| EV-REG-001 | 回归 | T4 | 全量测试和安全扫描通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-UNIT-001 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py` | pass | 16 tests OK；新增 `test_web_chat_must_call_decision_engine_plan`。 |
| EV-UNIT-002 | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py` | pass | 18 tests OK。 |
| EV-REG-001 | 2026-05-25 | `make test` | pass | Go tests、decision-engine 18 tests、agent-platform 27 tests、root tests 4 tests OK。 |
| EV-REG-002 | 2026-05-25 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-REG-003 | 2026-05-25 | `git diff --check` | pass | 无 whitespace error。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-CODE-001 | 2026-05-25 | 静态代码路径 | `AgentPlatform.__init__` 默认 `self.decision_engine = decision_engine or DecisionEngine()`；`process_case()` 调用 `self.decision_engine.plan(request)`。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 文档明确进程内调用 Decision Engine | T2 | EV-CODE-001 | pass |
| 测试验证 Web Chat 调用 `decision_engine.plan()` | T3 | EV-UNIT-001 | pass |
| 回归、secret scan、diff check 通过 | T4 | EV-REG-001, EV-REG-002, EV-REG-003 | pass |

## 未验证项

- 本轮不重新验证真实 Qwen/Gateway/health-food 业务结果；只验证 Decision Engine 主路径调用关系和文档澄清。

## 已知噪音

- `RecordingDecisionEngine` 是测试桩，用来证明 Agent Platform 会调用注入的决策层；它不代表真实排障能力。
