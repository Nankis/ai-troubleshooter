# HANDOFF

## 当前状态

- Local Code Agent 已实现并完成单测 / HTTP smoke。
- smoke 证据：`/tmp/ai_troubleshooter_local_code_debug.json`、`/tmp/ai_troubleshooter_local_code_no_mapping.json`。

## 下一步

- 后续如果要接真实主链路，应新开 Program 让 Go worker 在 Gateway 证据不足时二次调用 Python decision-engine，并传入 `debug_local_code=true`、`service_name`、`suspect_area`。
- 后续如果要更强代码理解，应独立加入 AST/call graph，仍保持 debug-only 和只读。
