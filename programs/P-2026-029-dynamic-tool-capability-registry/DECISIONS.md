# DECISIONS

## D1: JSON 导入不等于工具发布

MCP/HTTP 配置导入后只生成候选能力。只有通过只读安全校验、明确 scope/参数边界、并由用户发布后，才会进入 Gateway registry。

## D2: MCP 只能走 readonly adapter

即使 Web 支持粘贴 Claude/Cursor 风格 `mcpServers`，决策层也不能直连 MCP server。生产调用链仍是：

```text
Decision Engine -> Investigation Gateway -> readonly adapter -> allowlisted MCP tool
```

## D3: 不执行任意 stdio command

本轮不在 Web 导入后执行任意 MCP command。`mcpServers` 只入库为 pending discovery；可发布的是明确提供 readonly route/base_url 的能力。

## D4: 使用现有 tool registry 表扩展

`tb_troubleshoot_tool_registry` 已存在但未被运行时使用。本轮将其扩展为动态 capability 表，避免重复建一张语义相近的新表。

## D5: Web 导入支持 YAML，但发布仍使用统一 runtime

业务 manifest 示例已有 YAML，因此导入层支持 JSON/YAML 两种格式。无论来源是 HTTP manifest 还是 MCP route，最终运行时都必须落到 `tb_troubleshoot_tool_registry`，再由 Gateway 统一鉴权、scope、限流、timeout、审计和脱敏。

## D6: 本地默认 agent 允许动态工具，生产仍要求显式 allowlist

为了本地开箱即用，内置默认 agent 支持 `allowed_tools=*` 并新增 `dynamic:read` scope；生产配置若使用 `GATEWAY_AGENT_CONFIG_FILE` / JSON，仍应显式配置新 tool 和 scope。
