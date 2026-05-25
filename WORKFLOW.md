# WORKFLOW.md

本仓库的排障工作流采用“Brief 驱动”的轻量模式：先把问题、目标、假设和证据边界说清，再由 Python Decision Engine 决定是否查询 Gateway 只读工具。

## InvestigationBrief

每个 case 在调用 Decision Engine 前必须形成 `InvestigationBrief`：

```yaml
problem: 用户可理解的问题描述
goal: 本轮排障要回答的核心问题
success_criteria:
  - 至少有一条可追溯证据支持结论
  - 没有证据时必须追问或转人工
constraints:
  gateway_only: true
  readonly_only: true
  max_tool_calls: 10
hypotheses:
  - id: recommendation_generation
    question: 每日推荐是否生成或是否已过期
    expected_evidence: recommendation status / meal fingerprint
available_evidence:
  - source: platform_mysql
    kind: context_ledger
stop_conditions:
  - missing_required_user_identifier
  - tool_budget_exhausted
  - no_real_decision_agent
```

## Decision Rules

- 真实排障必须先确认真实决策 Agent 可用；`local_rules` 不能进入 Gateway/Knowledge/Tool 主路径。
- Brief 是高层引导，不是证据。最终结论只来自平台经验、Gateway 只读工具、真实日志/DB、或 debug-only 本地代码线索。
- 每个工具计划必须绑定 `hypothesis_id`、`reason` 和 `expected_evidence`，否则 Verifier 不接受。
- 查询必须有停止条件：工具预算、失败预算、超时、缺字段、证据不足都要结束或转人工。
- 本地代码检查只能在 Gateway 证据不足且用户/配置显式允许时作为最后手段。

## Workflow Task

复杂排障或平台变更可以拆成 workflow task，结构见 `.workflow/task.schema.json`。task 只能描述目标、验收和证据，不允许把密钥、原始日志或完整生产 payload 写入仓库。

## Evidence Level

结论不能高于证据等级：

- L0：文档/设计。
- L1：schema、单测、静态检查。
- L2：mock/fake smoke。
- L3：本地真实依赖，例如 MySQL、真实本地服务、真实本地代码仓。
- L4：预发/生产真实接口或外部平台。

涉及业务排障验收时，必须至少达到 L3；不能用 mock、memory 或 local_rules 冒充真实通过。
