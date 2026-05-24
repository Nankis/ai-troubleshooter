# Program: P-2026-036 Context Ledger Agent Isolation

## 背景

用户指出：如果排查过程始终由一个主 Agent 承接所有工具结果和推理历史，上下文会快速膨胀，容易在后半程失真、遗忘边界或编造结论。

## 目标

- 主 Agent 保持短上下文，只读取 case snapshot、Context Ledger 摘要、specialist 报告和证据引用。
- 工具原始返回不直接进入 LLM 总结上下文。
- 每次工具证据和最终结论都可通过 `tool_call_id` / `query_id` 回查。
- 最终回答前做证据引用校验，缺证据时降低置信度并要求人工确认。
- 不改变 Go Gateway 职责；本轮只升级 Python Agent Platform / Decision Engine 和平台 MySQL。

## 非目标

- 不引入 LangGraph/LangChain 等重型编排框架。
- 不改 Gateway readonly policy。
- 不做真实生产 health-food 验收。
- 不做 UI 布局调整。

## 验收标准

- 新增 MySQL Context Ledger DDL。
- Agent Platform 写入 case state、gateway tools、knowledge retrieval、agent report、tool evidence、final summary ledger。
- LLM summarize 接收的 observation 不包含原始 Gateway `data`。
- Decision Engine 能读取 context ledger 摘要并在 verifier 中标记。
- Python 单测、`make test`、`make secret-scan`、`git diff --check` 通过。
- 如本地 MySQL 可用，执行 migration 并通过 API 真实写入、查询 `tb_troubleshoot_context_ledger`。
