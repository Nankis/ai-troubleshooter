# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | docs | Task 1 | Program 建立 | pass |
| EV-T2-001 | docs | Task 2 | README 架构图和流程图更新 | pass |
| EV-T3-001 | docs | Task 3 | ADR 边界同步 | pass |
| EV-T4-001 | docs | Task 4 | Gateway 安全和一期原则同步 | pass |
| EV-T5-001 | command | Task 5 | `git diff --check` 通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T5-001 | 2026-05-23 01:09 CST | `git diff --check` | pass | 无输出 |

## 文档证据

| Evidence ID | 时间 | 文件/范围 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | 2026-05-23 01:09 CST | `README.md` | 架构图改为 Agent 平台、Gateway、业务服务、业务 DB 四块；流程图加入经验评分和统一回复出口。 | pass |
| EV-T3-001 | 2026-05-23 01:09 CST | `docs/architecture-decisions.md` | ADR 目标边界改为用户手绘图对应边界，并新增约束。 | pass |
| EV-T4-001 | 2026-05-23 01:09 CST | `docs/gateway-security.md`、`docs/phase1.md` | 补充业务能力注册、双层鉴权、Agent 隔离 DB、平台数据不走 Gateway。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| README 架构图体现三段边界 | Task 2 | EV-T2-001 | pass |
| README 流程图体现经验评分和 Gateway 查询分支 | Task 2 | EV-T2-001 | pass |
| ADR 明确 Gateway 不查平台 MySQL | Task 3 | EV-T3-001 | pass |
| Gateway 安全文档明确双层鉴权和 Agent 隔离 | Task 4 | EV-T4-001 | pass |
| `git diff --check` 通过 | Task 5 | EV-T5-001 | pass |

## 未验证项

- 本轮为 docs-only，不涉及服务启动、Go/Python 单测或接口 smoke。

## 已知噪音

- 无。
