# ERRORS

## E1. 本地代码结果只给路径和命中词，不够可操作

- 现象：Web Chat 输出 `file:line (term)` 列表，开发者不知道具体方法、疑点、应该看哪几行。
- 根因：Local Code evidence 虽然有符号和调用边，但平台回复只使用 `_local_code_top_hits` 拼接路径；证据结构也缺少面向开发者的 `suspect_reasons`、`follow_up_checks` 和有界摘录。
- 修复：本 Program 增加 actionable code findings，并重写平台回复格式。

## E2. 原始代码摘录不能直接塞进决策日志

- 现象：首次 API 验证时，`orchestrator_plan` 写入 MySQL 因 `output_snapshot_json` 超长失败。
- 根因：Local Code evidence 增加代码摘录后，`decision.to_dict()` 直接进入决策日志，超出字段容量，也扩大源码泄漏面。
- 修复：Agent Platform 写决策日志前使用 `_decision_log_snapshot()` 压缩 local_code evidence，只保留文件、符号、行范围、疑点、下一步和 `code_excerpt_line_count`。
