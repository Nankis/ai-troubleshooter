# HANDOFF

## 当前状态

- Program 已完成。MCP server 现在可以通过 readonly adapter 接入 Gateway。
- 决策层仍不直连 MCP，安全边界保持在 Gateway。
- health-food MCP mock server、adapter、dev-server 和 Web Chat 端到端验证均已通过。

## 下一步

- 接真实公司 MCP server 时，先配置 route allowlist，再用 `docs/mcp-gateway-adapter.md` 的验收标准验证。
- 如需生产化，建议把 MCP adapter 做成独立容器，并接入统一进程守护、日志、metrics 和密钥管理。
