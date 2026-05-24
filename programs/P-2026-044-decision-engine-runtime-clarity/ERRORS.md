# ERRORS

## E1. 文档把“不单独启动”写得像“不使用”

- 现象：`正常 Web Chat 场景不要求业务方单独启动 Decision Engine` 容易被理解为排查没有经过 Decision Engine。
- 根因：文档只描述部署形态，没有明确 Agent Platform 会进程内调用 `DecisionEngine.plan()`。
- 修复：更新接入文档和 Decision Engine README，并增加 Web Chat 回归测试。
