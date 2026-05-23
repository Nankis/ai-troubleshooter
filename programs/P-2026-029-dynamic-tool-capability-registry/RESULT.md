# RESULT

已完成动态能力注册中心一期：

- 新增 MySQL DDL：业务服务、MCP server、tool validation run，并扩展 `tb_troubleshoot_tool_registry` 为动态 capability runtime source。
- 新增 capability 导入层：支持 readonly HTTP manifest JSON/YAML、MCP routes JSON、Claude/Cursor `mcpServers` JSON。
- 安全校验默认保守：写操作、危险关键词、非 `/readonly/` 路径、任意 SQL/command 类能力会 rejected；`mcpServers` 只 pending discovery，不执行 command。
- Gateway 支持从 enabled capability 热加载动态工具，通用 readonly HTTP handler 继续走 Gateway 鉴权、scope、limit、timeout、audit、masking。
- Web 工作台新增“能力接入”：可录入配置、查看候选能力状态、发布/停用只读能力；发布后 Gateway Tools 面板可立即看到新增工具。
- 文档已更新 README、local runbook、business service registration、MCP adapter、gateway security 和 deployment checklist。

验证结论：本地 MySQL migration、Web/API 导入发布、Gateway 动态工具调用、危险能力拒绝、审计落库、`make test`、`make secret-scan`、`git diff --check` 均通过。

后续建议：

- 生产多实例需要 registry reload 事件或轮询刷新。
- 若公司要求更严格，可把 Web 发布动作接入内部审批流。
- MCP `mcpServers` pending discovery 后续可增加受控的 sandbox discovery job，但仍不能自动发布全部 tools。
