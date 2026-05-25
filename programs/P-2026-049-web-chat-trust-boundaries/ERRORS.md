# ERRORS

## E1. 闲聊命中平台经验

- 现象：用户输入“你好”时，case `case_20260525_000055` 命中 `knowledge:5` 并反复返回平台经验结论。
- 根因：`issue_domain` 为空时，平台仍查询全部 active knowledge，Decision Engine 又允许高置信经验直接回答。
- 修复：无 domain/无实体的低信号输入直接追问，不查询平台经验和 Gateway。

## E2. Mock/规则来源不够透明

- 现象：用户关闭本地决策 Agent 后仍收到回复，无法判断系统是否还在用模型、规则或 mock。
- 根因：系统内部能从规则编排、平台经验和 mock Gateway 生成回复，但最终文本缺少来源边界说明。
- 修复：最终总结补充 local_rules 和 mock adapter 来源说明。
