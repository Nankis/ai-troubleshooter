# P-2026-049 Web Chat Trust Boundaries

## 背景

用户指出三类问题：

1. 直接咨询 Agent 时一直像是在回复 mock 数据。
2. 关闭启用的本地决策 Agent 后仍然能回复，用户无法判断是否走了其他模型或规则。
3. Web 输入框不支持 Enter 直接发送。

这些问题会削弱平台可信度，命中历史复盘 `mock-as-real-evidence` 和 `model-output-overtrust`。

## 目标

- 闲聊、问候、无有效排障线索的问题不得命中平台经验或调用 Gateway，必须追问真实问题描述。
- Gateway 返回 mock evidence 时，最终回复必须显式标记“mock adapter，只能验证链路”。
- 当未启用本地决策 Agent 且主模型为 `local_rules` 时，回复必须说明当前是规则编排/平台经验/Gateway 证据，不是本地 Agent 或真实 LLM 推理。
- Web textarea 支持 Enter 发送、Shift+Enter 换行。
- 补单测和真实 Web 验证。

## 非目标

- 本轮不接真实 health-food adapter。
- 本轮不接真实 Lark/飞书。
- 本轮不改变 Go Gateway 安全边界。
