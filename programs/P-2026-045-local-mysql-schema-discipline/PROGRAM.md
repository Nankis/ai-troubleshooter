# P-2026-045 Local MySQL Schema Discipline

## 背景

用户发现本地 MySQL 里出现多个 `ai_troubleshooter*` 和 `hf_troubleshoot*` schema。根因需要排查清楚，并阻止后续验证无限创建新 schema。

## 目标

- 盘点当前本地 schema 来源。
- 平台本地验证统一使用 canonical schema：`ai_troubleshooter`。
- 迁移脚本、Python Agent Platform、Go Gateway 对本地非 canonical 平台 schema 做 fail-fast。
- health-food readonly adapter 不再默认鼓励 `hf_troubleshoot_codex` 这类临时库名。
- 提供非破坏性 schema 审计和显式确认清理脚本。

## 非目标

- 本轮不自动删除用户本地已有 schema。
- 本轮不合并/迁移历史临时库里的数据。
