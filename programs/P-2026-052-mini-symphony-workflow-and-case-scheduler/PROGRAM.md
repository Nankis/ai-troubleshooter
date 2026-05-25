# P-2026-052 Mini Symphony Workflow And Case Scheduler

## Objective

把“Brief 驱动排障”和“轻量 case scheduler”先固化为最小可执行契约：`WORKFLOW.md`、workflow task schema、Program 验证脚本和调度设计草图。

## Scope

- 新增根目录 `WORKFLOW.md`，约束 Agent 如何把用户问题转成目标、假设、证据和停止条件。
- 新增 workflow task schema，供后续 Program 和调度任务引用。
- 新增 Program 结构验证脚本，避免 Program 只有目录没有证据。
- 新增 case scheduler 设计草图，不在本 Program 做生产调度实现。

## Acceptance

- 文档和 schema 可以被后续 Program 引用。
- 验证脚本能检查本轮新增 Program 的基本文件和 Evidence/Result 结构。
- 不能引入 mock/memory 验收口径。
