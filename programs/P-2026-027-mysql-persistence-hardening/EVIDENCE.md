# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | code | Task 1 | mysql 缺 DSN fail-fast | pass |
| EV-T2-001 | unit | Task 2 | storage 打开策略测试 | pass |
| EV-T3-001 | docs | Task 3 | 文档不再允许隐式 memory | pass |
| EV-T4-001 | command | Task 4 | MySQL migration 可执行 | pass |
| EV-T4-002 | browser/db | Task 4 | UI 录入经验写入 MySQL | pass |
| EV-T4-003 | browser/api | Task 4 | 重启后经验仍可读取 | pass |
| EV-T5-001 | command | Task 5 | secret scan / diff check | pass |
| EV-T5-002 | command | Task 5 | go vet | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | 2026-05-23 | `go test ./...` | pass | 全量 Go 单测通过，新增 `internal/storage` fail-fast 测试通过。 |
| EV-T4-001 | 2026-05-23 | `MYSQL_DATABASE=ai_troubleshooter make migrate-mysql` | pass | 本地 MySQL 应用 `001_initial.sql` 到 `004_case_idempotency.sql`。密码仅通过环境变量传入。 |
| EV-T5-001 | 2026-05-23 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-T5-001 | 2026-05-23 | `git diff --check` | pass | 无输出。 |
| EV-T5-002 | 2026-05-23 | `go vet ./...` | pass | 无输出。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-002 | 2026-05-23 | mysql 缺 DSN fail-fast | `APP_ENV=dev DB_DRIVER=mysql DB_DSN='' /tmp/ai-troubleshooter-dev-server` 退出码 1，输出 `DB_DSN is required when DB_DRIVER=mysql`。 | pass |
| EV-T4-002 | 2026-05-23 | Web UI 手动录入平台经验并查表 | Chrome 打开 `http://127.0.0.1:18088/web`，点击录入并保存标题 `验证MySQL落库-1779548259929`；截图 `/tmp/ai-troubleshooter-mysql-knowledge-ui.png`；MySQL 查询 `tb_troubleshoot_knowledge_item` 返回 id=1、`knowledge_status=active`、`recommended_steps_json` 有 3 条步骤。 | pass |
| EV-T4-003 | 2026-05-23 | 重启 MySQL-backed Web 后读取经验 | 杀掉旧服务进程后重新用 `DB_DRIVER=mysql` 启动；`/web/api/overview` 返回 `knowledge_count=1` 且找到 `验证MySQL落库-1779548259929`。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| `go test ./...` 通过 | Task 2 | EV-T2-001 | pass |
| `git diff --check` 通过 | Task 5 | EV-T5-001 | pass |
| MySQL migration 可执行 | Task 4 | EV-T4-001 | pass |
| Web 工作台以 MySQL store 启动 | Task 4 | EV-T4-002 | pass |
| UI 录入经验后 MySQL 表能查到 | Task 4 | EV-T4-002 | pass |
| 重启后经验仍可读取 | Task 4 | EV-T4-003 | pass |

## 未验证项

- 无。

## 已知噪音

- Browser 插件当前缺少虚拟剪贴板，不能稳定向表单输入文字；本轮使用本机 Chrome 调试端口完成真实页面录入和保存，未降级成后端 curl 写入。
