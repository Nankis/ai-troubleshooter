# InvestigationBrief 设计

`InvestigationBrief` 是 Decision Engine 的短上下文入口。它不是监督系统，也不是结论生成器，而是把“要解决什么问题、为什么查这个工具、什么时候停下来”写清楚。

## 字段

| 字段 | 描述 |
| --- | --- |
| `problem` | 用户原始问题的可读摘要 |
| `goal` | 本轮排障要回答的核心问题 |
| `success_criteria` | 认为本轮排障有效的条件 |
| `constraints` | 只读、Gateway-only、预算、超时、隐私等边界 |
| `hypotheses` | 候选假设，每个包含 id、question、expected_evidence |
| `available_evidence` | 已加载的 case/entity/knowledge/gateway tools/context 摘要 |
| `stop_conditions` | 缺字段、无真实 Agent、预算耗尽、证据不足等停止条件 |

## 使用方式

1. Agent Platform 分类后生成 Brief。
2. Brief 写入 `tb_troubleshoot_context_ledger`。
3. Brief 进入 `DecisionRequest`。
4. Supervisor 和 LLM advisor 只能输出绑定假设的工具计划。
5. Verifier 校验工具预算、可用工具、Gateway-only 和 Brief 绑定。

## 非目标

- 不替代 Gateway 鉴权、限流、scope、脱敏和审计。
- 不允许直接查询业务 DB。
- 不把模型猜测当证据。
