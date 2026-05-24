# DECISIONS

## D1. 本地代码结果必须面向开发者可行动

只返回 `path:line` 不够。Local Code Agent 返回的 evidence 必须包含符号、行范围、疑点和下一步核对建议，平台回复按条目展示。

## D2. 有界代码摘录只允许 debug-only + allowlist

本地代码辅助仍需显式 `debug_local_code=true` 且 Gateway 证据不足。只有本地 allowlist 仓库、deny globs 和敏感词脱敏通过后，才允许返回短行代码摘录；摘录不是生产证据。

## D3. 生产排障仍以 Gateway 证据为准

代码定位是最后手段，用来辅助开发者定位可能 bug。结论必须提示“需结合 DB/日志/Gateway 证据确认”。
