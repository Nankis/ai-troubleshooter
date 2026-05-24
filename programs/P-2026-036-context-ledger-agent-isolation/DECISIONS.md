# DECISIONS

## D1：不用重型多 Agent 框架先解决上下文膨胀

本轮不直接引入 LangGraph/LangChain。原因：当前问题的核心不是框架能力不足，而是证据上下文管理不当。先把事实证据、原始审计、LLM 可见摘要分层，能更小成本地降低幻觉风险。

## D2：Context Ledger 是平台数据，不走 Gateway

Context Ledger 属于 Agent 平台自己的 case 状态、证据摘要和决策痕迹，写入平台 MySQL。Gateway 仍只负责业务 readonly tools 的鉴权、限流、审计和脱敏。

## D3：原始工具 data 不进入 LLM

Gateway 原始返回可以进入 `tb_troubleshoot_ai_decision_log` 的脱敏审计快照，但 LLM 总结阶段只接收 `tool_name/status/summary/result_count/data_shape/evidence_refs`。需要复核原始证据时，通过 `tool_call_id` / `query_id` 回查。

## D4：最终回答必须经过证据引用校验

`verifier_final_answer` 检查是否存在成功工具证据和证据引用。没有证据引用时不阻塞流程，但会降低置信度，明确转人工确认。
