# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P053-001 | code | Brief model/API/Web | DecisionRequest、Agent Platform、Web 均支持 brief | pass |
| EV-P053-002 | command | 单测/全测 | brief 单测和全量测试通过 | pass |
| EV-P053-003 | field | MySQL 真实链路 | `case_20260525_000068` 写入 `investigation_brief` ledger | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-P053-002 | 2026-05-25 | `make test` | pass | Go/Python/tests 全量通过 |
| EV-P053-002 | 2026-05-25 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-P053-002 | 2026-05-25 | `git diff --check` | pass | 无输出 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P053-003 | 2026-05-25 | 真实 MySQL + Gateway + adapter case | `case_20260525_000068`，`tb_troubleshoot_context_ledger.ledger_type='investigation_brief'`，summary=`定位 health_food 用户 2054603630081875968...` | pass |
| EV-P053-003 | 2026-05-25 | Web 展示 | `programs/P-2026-056-case-scheduler-state-machine/artifacts/web-case-000068-brief.png` 显示 Brief 和 `recommendation_generation` | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| case API 返回 `investigation_brief` | API payload | EV-P053-001 / EV-P053-003 | pass |
| MySQL Context Ledger 有 `investigation_brief` | MySQL 真实链路 | EV-P053-003 | pass |
| Web 页面能看到 brief 内容 | Web 验证 | EV-P053-003 | pass |

## 未验证项

- 无。

## 已知噪音

- health-food `/food-health/sys/alive` 返回业务未登录 JSON，但 HTTP reachable；本轮 adapter 只使用 MySQL readonly 数据作为业务证据。
