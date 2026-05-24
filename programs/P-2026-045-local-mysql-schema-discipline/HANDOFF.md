# HANDOFF

当前目标：阻止本地 MySQL 无限重复创建排障平台 schema。

已完成：

- 盘点本机相关 schema：`ai_troubleshooter`、`ai_troubleshooter_hf_codex`、`ai_troubleshooter_hf_real`、`ai_troubleshooter_itest`、`ai_troubleshooter_p2026008`、`hf_troubleshoot_codex`。
- 确认根因：迁移脚本允许任意 `MYSQL_DATABASE` 创建、历史 Program 使用临时 schema、health-food adapter 默认临时 schema。
- 已实现守卫：migration、Python Agent Platform、Go Gateway 都会拒绝本地非 canonical 平台库，除非显式 `ALLOW_NON_CANONICAL_LOCAL_DB=true`。
- 已移除 health-food adapter 的临时 schema 默认值。
- 已新增 `scripts/mysql-local-schema-audit.sh`，默认只列出重复 schema 和 DROP 建议，不删除。
- 已更新 `AGENTS.md`、`docs/LESSONS.md`、`README.md`、本地运行和业务接入文档。
- 已验证：`make test`、`make secret-scan`、`git diff --check` 通过；详情见 `EVIDENCE.md`。

下一步：

- 无；如需清理历史 schema，先由用户确认哪些库可以删除，再执行审计脚本的显式清理模式。

风险：

- 不自动 DROP 现有 schema，避免破坏用户本地数据；只提供显式确认清理脚本。
- 历史临时库数据未迁移，后续如果用户确认不需要保留，可按审计脚本输出清理。
