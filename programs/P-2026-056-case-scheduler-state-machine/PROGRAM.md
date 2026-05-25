# P-2026-056 Case Scheduler State Machine

## Objective

实现最小 case scheduler 状态机，让 case claim/start/finish/timeout 过程有统一事件和状态，而不是散落在同步流程里。

## Scope

- 新增 Python case scheduler 模块，定义状态、合法迁移和 run envelope。
- Agent Platform process_case 接入 scheduler claim/finish event。
- Web/进度可以看到 scheduler claim/finish。
- 暂不引入复杂后台 worker；保留后续可替换为异步 scheduler 的边界。

## Acceptance

- 单测证明非法状态不会重复 claim。
- MySQL Agent Run/Event 记录 scheduler claimed/finished。
- 全链路 Web/API 验证能看到排查状态变化。
