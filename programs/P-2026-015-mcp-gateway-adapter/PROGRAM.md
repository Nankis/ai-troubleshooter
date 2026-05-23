# P-2026-015 MCP Gateway Adapter

## 背景

用户要求系统支持快速接入外部 MCP。当前架构要求决策层不能直连 MCP 或业务服务，生产只读证据必须经过 Investigation Gateway。

## 目标

- 新增 MCP readonly adapter，把 MCP `tools/list` / `tools/call` 映射为本系统标准 readonly HTTP adapter。
- Gateway 和 Python 决策层保持不变，仍只看到受控工具。
- 支持 allowlist route 配置，只有显式声明的 MCP tool 才能被 Gateway 调用。
- 用 health-food MCP mock server 实际启动 adapter 和 dev-server，验证 Gateway 能成功调用 MCP tool 并得到预期证据。

## 非目标

- 不让决策层直接连接 MCP。
- 不开放 MCP 写工具。
- 不自动信任 MCP server 的所有工具。
