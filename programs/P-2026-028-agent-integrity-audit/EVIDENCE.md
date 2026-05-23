# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | repo scan | T1 | 历史错误已盘点 | pass |
| EV-T2-001 | repo scan | T2 | mock/memory/未验证项已盘点 | pass |
| EV-T3-001 | browser/db | T3 | Web 知识 CRUD MySQL-backed | pass |
| EV-T3-002 | api/db | T3 | Web Chat case/decision/tool audit MySQL-backed | pass |
| EV-T3-003 | api/db | T3 | root cause 自进化 MySQL-backed | pass |
| EV-T4-001 | docs | T4 | AGENTS/LESSONS/README 已更新 | pass |
| EV-T5-001 | command | T5 | 全量测试和安全扫描 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | `find programs -maxdepth 2 -name 'ERRORS.md' ... rg` | pass | 汇总 P-2026-001/006/008/009/010/012/015/017/027 等错误。 |
| EV-T2-001 | 2026-05-23 | `find programs -maxdepth 2 -name 'EVIDENCE.md' ... rg 'DB_DRIVER=memory'` | pass | 命中 P-2026-018、020、021、022、023、024、025、026。 |
| EV-T2-002 | 2026-05-23 | `find programs -maxdepth 2 -name 'EVIDENCE.md' ... rg '未验证项|SKIP|pending|blocked'` | pass | 命中 10 个 Program，已在 `FEATURE_AUDIT.md` 归类。 |
| EV-T3-002 | 2026-05-23 | `curl -F message=... http://127.0.0.1:18088/web/api/chat` | pass | 创建 `case_20260523_000001`，返回 health-food 推荐缺失排查结论。 |
| EV-T3-003 | 2026-05-23 | `POST /cases/case_20260523_000001/root-cause` | pass | case 转 `DONE`，返回 `knowledge_id=3` 和 `evolution_decision=upserted`。 |
| EV-T5-001 | 2026-05-23 | `make test` | pass | Go 全量测试通过；Python decision-engine 14 个单测通过；repo Python tests 3 个通过。 |
| EV-T5-002 | 2026-05-23 | `go vet ./...` | pass | 无输出。 |
| EV-T5-003 | 2026-05-23 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-T5-004 | 2026-05-23 | `git diff --check` | pass | 无输出。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T3-001 | 2026-05-23 | MySQL-backed Web UI 经验创建、预览、编辑、删除 | Chrome 调试端口打开 `http://127.0.0.1:18088/web`；页面事件创建 `审计-经验编辑删除-1779548798868`，预览 modal 成功，编辑为 `审计-经验编辑删除-1779548798868-已编辑`，删除后列表隐藏；截图 `/tmp/ai-troubleshooter-audit-knowledge-crud.png`。 | pass |
| EV-T3-001 | 2026-05-23 | 查询 MySQL 经验表 | `tb_troubleshoot_knowledge_item` 中 id=2，title=`审计-经验编辑删除-1779548798868-已编辑`，`knowledge_status=deleted`，`typical_description` 为编辑后内容。 | pass |
| EV-T3-002 | 2026-05-23 | 查询 Web Chat 平台表 | `tb_troubleshoot_case` 有 `case_20260523_000001`；`tb_troubleshoot_case_message` 3 条；`tb_troubleshoot_ai_decision_log` 10 条；`tb_troubleshoot_tool_call_audit` 5 条且均 `allowed`。 | pass |
| EV-T3-003 | 2026-05-23 | 查询 root cause 和 evolution 表 | `tb_troubleshoot_root_cause` 记录 `recommendation_job_skipped`；`tb_troubleshoot_knowledge_item` id=3 active；`tb_troubleshoot_knowledge_evolution_run` run_no=`ke_20260523_000001` decision=`upserted`。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 功能证据矩阵覆盖主要功能域 | T1/T2 | EV-T1-001, EV-T2-001, EV-T2-002 | pass |
| 历史错误已整理 | T1 | EV-T1-001 | pass |
| MySQL-backed Web 知识 CRUD | T3 | EV-T3-001 | pass |
| Web Chat / decision / audit 落库 | T3 | EV-T3-002 | pass |
| Root cause 自进化落库 | T3 | EV-T3-003 | pass |
| AGENTS/LESSONS/README 更新 | T4 | EV-T4-001 | pass |
| 全量验证通过 | T5 | EV-T5-001, EV-T5-002, EV-T5-003, EV-T5-004 | pass |

## 未验证项

- 未接真实 Lark/Feishu bot。
- 未接真实生产 health-food 日志接口。
- 未连接真实 DMS 实例或 DMS MCP server。
- 未重跑真实 Qwen/Qwen-VL smoke；当前本地服务使用 `local_rules`。
- 未让 Go worker 切到 Python decision-engine。

## 已知噪音

- Browser 插件虚拟剪贴板不可用；本轮 UI 验收改用本机 Chrome 调试端口触发真实页面事件。
- Web Chat 业务证据来自 mock connector；平台数据落库为真实 MySQL。
