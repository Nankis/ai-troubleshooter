# P-2026-029 Dynamic Tool Capability Registry

## 背景

用户希望业务方像 Claude/Cursor 接 MCP 一样，把别人提供的 MCP JSON 或 readonly HTTP manifest 直接录入 Web，就能生成可用的排障工具；同时必须避免危险操作、任意命令执行、任意写接口或未经 allowlist 的 MCP tool 进入生产排障链路。

当前 Gateway 默认工具由 Go 代码注册，MCP route allowlist 由 JSON/env 配置驱动，Web 只能展示工具，不能录入、校验、发布能力。

## 目标

- 增加平台能力注册数据模型，记录业务服务、MCP server、Gateway tool capability 和验证结果。
- 支持 Web/API 导入 MCP JSON、MCP route JSON 和 readonly HTTP manifest JSON。
- 对候选能力做只读安全校验：危险动作默认拒绝，未明确 readonly 的能力不能自动启用。
- Web 端提供“能力接入”入口，支持粘贴配置、查看候选能力、发布安全能力。
- Gateway 在本地 dev-server 中能从已启用能力动态补充工具，发布后无需改 Go 代码。

## 非目标

- 不执行任意 MCP stdio command。
- 不让决策层直连 MCP。
- 不把 MCP tools/list 返回的所有 tool 自动开放。
- 不实现生产级 secret manager，只保存 `secret_ref`，真实 token 仍通过环境变量或公司密钥系统注入。
- 不完成复杂多实例热更新；本轮先保证单进程 dev-server 发布后 reload registry。

## 验收标准

- 新 migration 可在本地 MySQL 执行。
- MCP/HTTP import API 单测覆盖 readonly candidate、dangerous rejected、Claude/Cursor mcpServers pending discovery。
- Web 端可以录入配置，展示能力候选，发布安全能力。
- 发布后的 HTTP readonly capability 能出现在 `/tools` 和 Web tools 列表。
- 危险 method/name/path 被拒绝且不能发布。
- `make test`、`make secret-scan`、`git diff --check` 通过。
