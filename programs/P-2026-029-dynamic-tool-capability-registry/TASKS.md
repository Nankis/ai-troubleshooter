# TASKS

## Task 1: [x] 建立 Program

- 记录背景、目标、非目标和验收标准。

## Task 2: [x] 能力注册模型和 DDL

- 新增业务服务 / MCP server 表。
- 扩展 tool registry 表为动态 capability runtime source。
- 实现 Go model、MySQL store 和内存 store。

## Task 3: [x] 导入与安全校验

- 支持 MCP route JSON、Claude/Cursor mcpServers JSON、readonly HTTP manifest JSON/YAML。
- 阻断危险 method/name/path/description。
- 只生成 draft/rejected/pending discovery，不自动执行 command。

## Task 4: [x] Gateway 动态注册

- 已启用 capability 转成 `tool.Spec`。
- HTTP readonly adapter capability 可通过通用 handler 调用。
- 发布后单进程 dev-server reload registry。

## Task 5: [x] Web 接入

- 新增能力接入表单。
- 展示候选能力、风险和状态。
- 支持发布安全能力。

## Task 6: [x] 验证与交付

- 单测、secret scan、diff check。
- 本地 MySQL migration 和 API/UI smoke。
- 更新文档、RESULT、EVIDENCE，commit + push。
