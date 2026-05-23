# HANDOFF

## 当前状态

- Program 已完成。Gateway agent/scope/tool/chat 授权已支持 JSON/file 配置，runner agent id 已支持 `GATEWAY_AGENT_ID`。

## 下一步

- 如要继续提升开箱能力，下一步建议做 manifest-driven Tool Registry，让业务 capability manifest 能自动注册工具 spec，而不是继续在 Go 中维护默认工具列表。
