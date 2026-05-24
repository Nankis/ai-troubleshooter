# ERRORS

## E1. 本地 schema sprawl

- 现象：本地 MySQL 出现 `ai_troubleshooter_hf_codex`、`ai_troubleshooter_hf_real`、`ai_troubleshooter_itest`、`ai_troubleshooter_p2026008`、`hf_troubleshoot_codex` 等 schema。
- 根因：
  - 早期 Program 为隔离验证直接改 `MYSQL_DATABASE`，例如 P-2026-008 使用 `ai_troubleshooter_p2026008`。
  - `scripts/mysql-migrate.sh` 对任何 `MYSQL_DATABASE` 都直接 `CREATE DATABASE IF NOT EXISTS`。
  - `scripts/real-health-food-readonly-adapter.py` 默认 `HEALTH_FOOD_MYSQL_DATABASE=hf_troubleshoot_codex`，文档也复用了这个临时库名。
  - 缺少“本地平台 schema 固定、临时 schema 必须清理”的硬规则。
- 修复：本 Program 增加三层 fail-fast、移除临时默认库名、补审计/清理脚本和文档规则。
