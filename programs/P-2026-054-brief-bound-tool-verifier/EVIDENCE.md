# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P054-001 | code | ToolPlan/Verifier | 工具计划必须有 reason、hypothesis_id、expected_evidence | pass |
| EV-P054-002 | command | 单测/全测 | Verifier 缺 hypothesis 拒绝路径覆盖 | pass |
| EV-P054-003 | field | MySQL 真实链路 | `tool_invocation.input_snapshot_json` 写入 hypothesis 和 expected evidence | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-P054-002 | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py` | pass | 22 tests |
| EV-P054-002 | 2026-05-25 | `make test` | pass | 全量通过 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P054-003 | 2026-05-25 | MySQL 工具调用绑定 | `JSON_EXTRACT(input_snapshot_json,'$.hypothesis_id')` 返回 `recommendation_generation`、`input_data_completeness`、`user_eligibility`、`service_error`、`similar_case` | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 缺少 hypothesis/reason 的计划会被 Verifier 拦截或降级 | Verifier 单测 | EV-P054-002 | pass |
| MySQL 决策日志中工具调用 input 包含 hypothesis/expected_evidence | MySQL 真实链路 | EV-P054-003 | pass |

## 未验证项

- 无。

## 已知噪音

- 无。
