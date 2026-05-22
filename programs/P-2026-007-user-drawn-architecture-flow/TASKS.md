# TASKS

## Task 1: [x] 建立 Program 和 Scope

- 文件：`programs/P-2026-007-user-drawn-architecture-flow/**`
- 验收：Program 记录本次独立架构图修正，不回写旧 Program。
- Evidence：`EV-T1-001`

## Task 2: [x] 更新 README 架构图和流程图

- 文件：`README.md`
- 验收：
  - 架构图体现 Agent 平台、Investigation Gateway、业务服务和业务 DB 边界。
  - 流程图体现平台经验评分、高置信直接返回、低置信查 Gateway、统一回复出口。
- Evidence：`EV-T2-001`

## Task 3: [x] 同步 ADR 长期边界

- 文件：`docs/architecture-decisions.md`
- 验收：ADR 目标边界和约束与用户手绘图一致。
- Evidence：`EV-T3-001`

## Task 4: [x] 同步 Gateway 安全和一期原则

- 文件：`docs/gateway-security.md`、`docs/phase1.md`
- 验收：文档明确业务能力注册、双层鉴权、Agent 隔离 DB、平台数据不走 Gateway。
- Evidence：`EV-T4-001`

## Task 5: [x] 验证并提交

- 文件：本 Program
- 验收：
  - [x] `git diff --check` 通过。
  - [x] Evidence / Result / Handoff 回写。
- Evidence：`EV-T5-*`
