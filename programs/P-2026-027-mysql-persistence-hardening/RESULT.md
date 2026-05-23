# RESULT

## 结果摘要

- 修复了 `DB_DRIVER=mysql` 缺少 `DB_DSN` 时静默回退内存 store 的问题。
- 平台经验沉淀的验收标准升级为必须 MySQL 落库验证。
- 本地 Web 工作台已连接 MySQL，真实通过页面录入经验，并从 `tb_troubleshoot_knowledge_item` 查到记录；重启服务后仍能读取。

## 变更范围

- `internal/storage/storage.go`
- `internal/storage/storage_test.go`
- `README.md`
- `CONTRIBUTING.md`
- `docs/gateway-security.md`
- `docs/web-workbench.md`
- `programs/P-2026-027-mysql-persistence-hardening/`

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| Task 1 | done | EV-T1-001 |
| Task 2 | done | EV-T2-001 |
| Task 3 | done | EV-T3-001 |
| Task 4 | done | EV-T4-001, EV-T4-002, EV-T4-003 |
| Task 5 | done | EV-T5-001, EV-T5-002 |

## 验证摘要

- `go test ./...`：pass。
- `go vet ./...`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。
- MySQL migration：pass。
- Web UI 经验录入 + MySQL 查表：pass。
- 服务重启后读取经验：pass。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| MySQL 持久化验收不再允许隐式 memory | pass | EV-T1-001 |
| UI 经验录入实际落 MySQL | pass | EV-T4-002 |
| 重启后经验仍可读取 | pass | EV-T4-003 |

## Commit

- 本轮提交到 main，最终 commit hash 以 `git log -1` 为准。

## 残留风险

- 本地验证记录 `验证MySQL落库-1779548259929` 保留在 `ai_troubleshooter.tb_troubleshoot_knowledge_item`，用于证明重启后可读；如不需要可通过 Web UI 软删除。
