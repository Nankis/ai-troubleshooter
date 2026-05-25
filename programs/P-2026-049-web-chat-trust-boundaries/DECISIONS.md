# DECISIONS

## D1. 闲聊不是排障 case

当规则和 LLM 都没有识别到 `issue_domain`，且没有 uid/service/debug 等实体时，Decision Engine 必须 `ask_user`，不能查平台经验，也不能走默认 logs/deployment 工具。

## D2. Mock 证据必须在最终答复中暴露

只要 tool observation 的 summary/payload 中出现 mock 信号，最终总结前置 mock adapter 警告，避免 L2 mock 链路被误读成真实业务结论。

## D3. 本地决策 Agent 关闭后仍可有规则编排，但必须透明

平台可以在未启用本地决策 Agent 时用规则编排做安全追问或 Gateway 只读证据查询，但必须在最终回复里标明“未启用本地决策 Agent/真实 LLM，当前为规则编排”。

## D4. Enter 发送，Shift+Enter 换行

Web Chat 输入框采用常见聊天交互：Enter 提交表单，Shift+Enter 保留换行，IME 组合输入期间不触发提交。
