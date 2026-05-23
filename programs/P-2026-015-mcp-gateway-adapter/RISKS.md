# RISKS

- 外部 MCP tool 可能不是只读能力，需要 allowlist 和人工审核。
- MCP 返回内容格式可能不统一，adapter 必须只接受结构化结果或可解析 JSON 文本。
- MCP server 长时间卡住时，adapter 必须遵守 timeout。
