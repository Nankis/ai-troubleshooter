# RESULT

## 结果摘要

- 已确认本地多 schema 的根因不是 MySQL 自动行为，而是历史 Program 和 adapter 验证中使用了临时库名，加上 migration 脚本允许任意 `MYSQL_DATABASE` 自动建库。
- 本地平台库收敛为 `ai_troubleshooter`；migration、Python Agent Platform、Go Gateway 对本地非 canonical 平台库默认 fail-fast。
- health-food readonly adapter 不再默认 `hf_troubleshoot_*` 临时业务库，必须显式指定已有只读业务库。
- 新增本地 schema audit 脚本，只列出重复 schema 和建议 DROP SQL，默认不删除。

## 变更范围

- `scripts/mysql-migrate.sh`：本地 canonical schema 守卫、库名和 migration 文件名校验。
- `apps/agent-platform/agent_platform/config.py`：MYSQL/DB_DSN 加载时校验本地 schema。
- `internal/storage/storage.go`：Go Gateway MySQL DSN 本地 schema guard。
- `scripts/real-health-food-readonly-adapter.py`：移除临时库默认值。
- `scripts/mysql-local-schema-audit.sh`：非破坏性盘点和显式清理脚本。
- `AGENTS.md`、`docs/LESSONS.md`、`README.md`、`docs/local-runbook.md`、`docs/business-onboarding-quickstart.md`：写入防复发规则。

## 验证摘要

- `bash -n scripts/mysql-migrate.sh scripts/mysql-local-schema-audit.sh` 通过。
- `MYSQL_DATABASE=ai_troubleshooter_itest scripts/mysql-migrate.sh` 按预期 exit 2，拒绝本地非 canonical schema。
- `MYSQL_DATABASE=ai_troubleshooter scripts/mysql-migrate.sh` 通过，migration 均 skip。
- `scripts/mysql-local-schema-audit.sh` 列出 5 个历史非 canonical schema，且默认没有删除。
- `go test ./internal/storage` 通过。
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py` 通过。
- `make test` 通过。
- `make secret-scan` 通过。
- `git diff --check` 通过。

## Commit

- Commit message: `P-2026-045 local mysql schema discipline`。

## 残留风险

- 现有历史 schema 未自动删除，避免破坏用户本地证据；需要用户确认后再用 audit 脚本显式清理。
- 历史临时库数据没有自动迁移到 `ai_troubleshooter`。
