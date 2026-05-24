# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-ROOT-001 | 现场排查 | T1 | 找到本地重复 schema 和历史来源 | pass |
| EV-GUARD-001 | 脚本验证 | T2 | 本地非 canonical migration fail-fast | pass |
| EV-GUARD-002 | MySQL 验证 | T2/T5 | canonical schema 可迁移，重复 schema 只审计不删除 | pass |
| EV-TEST-001 | 单元测试 | T3 | Python / Go 启动配置拒绝本地非 canonical schema | pass |
| EV-REG-001 | 回归验证 | T6 | 全量测试、secret scan、diff check | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-GUARD-001 | 2026-05-25 | `bash -n scripts/mysql-migrate.sh scripts/mysql-local-schema-audit.sh` | exit 0 | shell 语法通过。 |
| EV-GUARD-001 | 2026-05-25 | `MYSQL_DATABASE=ai_troubleshooter_itest scripts/mysql-migrate.sh` | exit 2 | 按预期拒绝本地非 canonical schema，未连接 MySQL、未创建库。 |
| EV-GUARD-002 | 2026-05-25 | `MYSQL_DATABASE=ai_troubleshooter scripts/mysql-migrate.sh` | exit 0 | 7 个 migration 均 skip，canonical schema 正常可用。 |
| EV-GUARD-002 | 2026-05-25 | `scripts/mysql-local-schema-audit.sh` | exit 0 | 列出 5 个非 canonical troubleshooting schema 和建议 DROP SQL，明确 `No schema was dropped`。 |
| EV-TEST-001 | 2026-05-25 | `go test ./internal/storage` | exit 0 | 覆盖 Go Gateway local DSN guard，包括 `127.0.0.1`、`localhost`、`[::1]`。 |
| EV-TEST-001 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py` | exit 0 | 20 tests OK；覆盖 Agent Platform local DB guard。 |
| EV-REG-001 | 2026-05-25 | `make test` | exit 0 | Go 全仓、Decision Engine、Agent Platform、顶层 tests 全部通过。 |
| EV-REG-001 | 2026-05-25 | `make secret-scan` | exit 0 | Secret scan passed。 |
| EV-REG-001 | 2026-05-25 | `git diff --check` | exit 0 | 无 whitespace error。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-ROOT-001 | 2026-05-25 | 本机 schema 盘点 | 查询到 `ai_troubleshooter`、`ai_troubleshooter_hf_codex`、`ai_troubleshooter_hf_real`、`ai_troubleshooter_itest`、`ai_troubleshooter_p2026008`、`hf_troubleshoot_codex`。 | pass |
| EV-ROOT-002 | 2026-05-25 | 本机 schema 表数量和大小盘点 | `ai_troubleshooter` 17 tables；5 个非 canonical schema 分别 12-21 tables，确认是历史验证残留，不是当前脚本继续创建。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 找到本地重复 schema 来源 | T1 | EV-ROOT-001, EV-ROOT-002 | pass |
| 阻止 migration 继续创建本地临时平台库 | T2 | EV-GUARD-001, EV-GUARD-002 | pass |
| 服务启动配置阻止本地临时平台库 | T3 | EV-TEST-001 | pass |
| health-food adapter 不再默认临时业务库 | T4 | EV-REG-001 | pass |
| schema 审计不破坏用户数据 | T5 | EV-GUARD-002 | pass |
| 提交前回归 | T6 | EV-REG-001 | pass |

## 未验证项

- 没有自动删除现有本地 schema；需要用户确认里面没有要保留的证据后，再执行显式 DROP。
- 未迁移历史临时库数据到 `ai_troubleshooter`。

## 已知噪音

- `scripts/mysql-local-schema-audit.sh` 会打印 DROP 建议 SQL，但默认不执行。
- fail-fast 验证命令返回 exit 2 属于预期结果。
