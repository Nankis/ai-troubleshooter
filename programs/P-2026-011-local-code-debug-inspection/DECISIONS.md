# DECISIONS

## D1：Gateway 不下发本地路径

Gateway / adapter 只能提供 `service_name`、`repo_hint`、`suspect_area` 等业务线索。本地路径只来自 Python decision-engine 所在环境的 allowlist registry，避免下游诱导本地 Agent 读取任意路径。

## D2：本地代码检查是最后手段

只有 `debug_local_code=true` 且 `gateway_evidence_status` 表示证据不足时，Local Code Agent 才会运行。正常排障仍优先 Gateway 只读证据。

## D3：不返回源码片段

Local Code Agent 只返回 repo id、相对路径、命中词、行号和安全摘要，不返回源码内容，避免把业务源码和敏感配置写入日志或 IM 回复。
