# P-2026-027 MySQL Persistence Hardening

## 背景

用户在 Web 工作台手动录入“平台经验沉淀”时，发现记录没有写入本地 MySQL。排查后确认：本地 UI 验证期间服务被启动为 `DB_DRIVER=memory`，并且代码存在 `DB_DRIVER=mysql` 但 `DB_DSN` 为空时静默回退到内存 store 的历史逻辑。这个行为违反“平台经验沉淀必须落库验证”的强约束。

## 目标

- `DB_DRIVER=mysql` 时没有 `DB_DSN` 必须启动失败，不能静默退回 memory。
- 只有显式 `DB_DRIVER=memory` 且没有 `DB_DSN` 时才允许内存 store。
- 文档明确：经验沉淀、case、消息、审计和 AI 决策日志验收必须使用 MySQL。
- 本地启动 Web 工作台连接 MySQL，实际通过 UI 手动录入经验，并从 MySQL 表验证写入。
- 记录错误复盘，避免后续 Program 再用 memory 结果冒充持久化验收。

## 非目标

- 不改知识表 DDL。
- 不改 Web 工作台交互样式。
- 不接真实 Lark/Feishu。

## 验收标准

- `go test ./...` 通过。
- `git diff --check` 通过。
- MySQL migration 可执行。
- Web 工作台以 MySQL store 启动。
- 通过 Web UI 录入一条平台经验后，`tb_troubleshoot_knowledge_item` 能查询到对应记录。
- 重启服务后，该经验仍可从平台接口或页面读取。
