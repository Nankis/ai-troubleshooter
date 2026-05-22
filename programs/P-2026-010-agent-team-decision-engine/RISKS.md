# RISKS

- Python decision-engine 尚未接入 Go worker 的生产调用链；本轮只实现可独立运行和测试的决策层。
- Agent Team 仍是规则基线，真实 LLM 多 agent 推理和工具结果总结需要后续 Program。
- 如果后续加入 LangGraph，应保持 `/v1/decisions/plan` 外部契约兼容。
