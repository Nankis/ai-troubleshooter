# HANDOFF

当前目标：修复 Web Chat 可信边界问题：闲聊误命中经验、mock/local_rules 来源不透明、Enter 不发送。

已完成：

- Program 已建立。
- 已确认 case `case_20260525_000055` 的“你好”是空 domain 却查询全部 knowledge 导致的误答。
- 已修复低信号输入：不命中平台经验、不调用工具，追问具体生产问题。
- 已修复 mock/local_rules 透明度：最终回复会标注 mock adapter 和未启用本地决策 Agent。
- 已支持 Web textarea Enter 发送、Shift+Enter 换行。
- 已通过单测、真实 Web 和 MySQL 验证，证据见 `EVIDENCE.md`。

下一步：

- commit + push main。

风险：

- 业务证据侧如果仍用 Gateway mock connector，只能做 L2 链路验证，不能当真实业务结论。
