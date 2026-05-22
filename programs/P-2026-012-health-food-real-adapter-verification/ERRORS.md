# ERRORS

## E1：上一轮把 mock adapter 验证描述成接入验证

- 问题：mock adapter 只能证明平台编排和接口契约可用，不能证明 health-food 业务证据可靠。
- 修正：本 Program 改为真实注册、真实写业务 DB、真实 readonly adapter 查询、Web Chat 端到端和平台审计表验证。
- 防复发：后续业务接入验收必须在 `EVIDENCE.md` 写明证据来源；如果使用 mock，结论只能写“mock 流程验证”，不能写“真实接入验证”。

## E2：UI 验证暴露英文推荐问题识别缺口

- 问题：逐键输入英文 `recommendation missing` 后，旧规则层不能稳定归类为 `每日推荐缺失`。
- 修正：补充 health-food 英文关键词识别和单元测试。
- 防复发：Web Chat 验收至少保留一个英文或中英混合 case，避免规则层只适配中文样例。
