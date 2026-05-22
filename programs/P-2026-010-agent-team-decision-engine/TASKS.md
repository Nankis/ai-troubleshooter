# TASKS

## Task 1: [x] 梳理现有 Python 决策层和 Program 边界

- 确认现有 `DecisionEngine`、models、HTTP API 和测试。
- 确认不修改 Go Gateway。
- Evidence：`EV-T1-001`

## Task 2: [x] 实现 Supervisor / Specialist / Verifier

- Supervisor 负责编排 Knowledge、Kline、Asset 和 fallback。
- Kline Agent 负责 K 线必要字段和工具计划。
- Asset Agent 负责资产必要字段和工具计划。
- Knowledge Agent 负责高置信历史经验优先。
- Verifier 负责预算、工具白名单、去重和最终响应收敛。
- Evidence：`EV-T2-001`

## Task 3: [x] 补单测和接口文档

- 覆盖知识直答、Kline 计划、Asset 计划、缺字段追问、Verifier 截断/过滤。
- 更新 Python README 和 OpenAPI。
- Evidence：`EV-T3-001`

## Task 4: [x] 验证、提交和推送

- `make test`
- `git diff --check`
- `python3.13 scripts/secret-scan.py --mode all`
- Evidence：`EV-T4-001`
