# P-2026-050 Require Real Decision Agent

## Goal

把“必须使用真实决策 Agent 排查问题”从文案提醒升级为代码硬约束。

当没有启用本地决策 Agent，且没有开启真实 LLM 驱动的 Decision Engine 时：

- 不允许进入生产排障流程。
- 不允许查询 Gateway readonly tools。
- 不允许查询平台经验后直接给结论。
- 不允许用 `local_rules` / deterministic rules 冒充 Agent 排查。

## Trigger

用户指出 `local_rules` 继续回答问题属于欺骗用户，要求禁止这种规则排障，必须使用 Agent。

## Scope

- Python Agent Platform 主排障路径。
- Web Chat 相关行为验证。
- 单测和本地浏览器验证。
- 工作流规则补充。

