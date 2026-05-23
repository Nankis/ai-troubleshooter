# HANDOFF

## 当前状态

- 已修正 storage 打开策略。
- 已补充 storage 单元测试。
- 已更新 README/CONTRIBUTING/Gateway/Web 文档。
- 已执行本地 MySQL migration。
- 已用 Web UI 录入平台经验，并从 MySQL `tb_troubleshoot_knowledge_item` 查询到记录。
- 已重启服务并确认 `/web/api/overview` 仍可读取该经验。

## 恢复步骤

1. 本地服务仍在 `http://127.0.0.1:18088/web` 运行，使用 MySQL store。
2. 如需重启，继续使用本机临时 env 注入 `DB_DRIVER=mysql` 和 `DB_DSN`，不要把 DSN 写入仓库。
3. 涉及平台经验沉淀的后续验证必须重复“Web UI 保存 + MySQL 表查询 + 重启后读取”这组 evidence。
