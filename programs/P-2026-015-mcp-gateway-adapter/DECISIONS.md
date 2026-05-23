# DECISIONS

## D1：不让决策层直连 MCP

MCP server 可能包含写操作、文件读取、命令执行或未脱敏数据。决策层直接连接会绕过 Gateway 的鉴权、scope、限流、timeout、审计和脱敏。因此 MCP 必须先被桥接成 readonly adapter，再由 Gateway 调用。

## D2：MCP adapter 使用 allowlist route

adapter 不自动暴露 MCP `tools/list` 的全部工具。只有配置到 route 的 tool 才会成为 readonly endpoint。

## D3：验证必须跑真实进程

验收不能只靠单测或 mock 函数。必须启动 MCP server、adapter 和 dev-server，并通过 HTTP 调用 Gateway 工具，确认 Gateway -> adapter -> MCP tool 的链路成功。
