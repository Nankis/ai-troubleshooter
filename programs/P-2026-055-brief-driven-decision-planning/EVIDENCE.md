# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P055-001 | code | Supervisor planning | Supervisor report 暴露 brief goal/hypotheses，select_tools 按 brief/issue type 排序 | pass |
| EV-P055-002 | command | 单测/全测 | 推荐问题排序和绑定单测通过 | pass |
| EV-P055-003 | field | 真实链路 | `case_20260525_000068` 工具顺序为 recommendation、meals、profile、logs、similar | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-P055-002 | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py` | pass | 包含 `test_brief_guides_tool_order_and_binds_plan` |
| EV-P055-002 | 2026-05-25 | `make test` | pass | 全量通过 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P055-003 | 2026-05-25 | health-food 推荐问题真实链路 | `case_20260525_000068` 调用顺序：`get_health_food_recommendation_status`、`get_health_food_meal_records`、`get_health_food_user_profile`、`search_logs_by_service`、`get_similar_cases` | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 推荐问题优先推荐状态/餐食/用户资料，配额问题优先 quota | select_tools 排序 | EV-P055-002 / EV-P055-003 | pass |
| Verifier 仍控制预算、scope、可用工具和 brief 绑定 | Verifier 组合验证 | EV-P054-002 / EV-P055-002 | pass |

## 未验证项

- 无。

## 已知噪音

- 真实案例中 Codex advisor 参与了决策，但最终工具计划仍经过平台 Verifier。
