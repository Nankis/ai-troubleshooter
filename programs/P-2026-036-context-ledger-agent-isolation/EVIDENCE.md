# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | docs | T1 | Program 建立 | pass |
| EV-T5-001 | command | T5 | Agent Platform 单测覆盖 Context Ledger | pass |
| EV-T5-002 | command | T5 | Decision Engine 单测覆盖 context ledger snapshot | pass |
| EV-T6-001 | command | T6 | 全量测试 | pass |
| EV-T6-002 | command | T6 | secret scan | pass |
| EV-T6-003 | command | T6 | diff whitespace check | pass |
| EV-T6-004 | command | T6 | MySQL migration 应用 Context Ledger DDL | pass |
| EV-T6-005 | local-smoke | T6 | Agent Platform + Gateway API 写入并查询 MySQL Context Ledger | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T5-001 | 2026-05-24 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py` | pass | 10 tests pass。 |
| EV-T5-002 | 2026-05-24 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py` | pass | 17 tests pass。 |
| EV-T6-001 | 2026-05-24 | `make test` | pass | Go tests pass；Decision Engine 17 tests pass；Agent Platform 11 tests pass；root tests 4 pass。 |
| EV-T6-002 | 2026-05-24 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-T6-003 | 2026-05-24 | `git diff --check` | pass | 无输出。 |
| EV-T6-004 | 2026-05-24 | `MYSQL_PASSWORD=<local> MYSQL_DATABASE=ai_troubleshooter make migrate-mysql` | pass | `007_context_ledger.sql` applied。 |
| EV-T6-005a | 2026-05-24 | `DB_DRIVER=mysql ... .venv/bin/python -m agent_platform` | pass | Agent Platform 启动在 `127.0.0.1:19091`，平台持久化使用 MySQL。 |
| EV-T6-005b | 2026-05-24 | `DB_DRIVER=memory CONNECTOR_MODE=mock go run ./cmd/investigation-gateway` | pass | Gateway 启动在 `127.0.0.1:18080`；本项只作为 mock 只读工具链路，不作为 Gateway 审计持久化验收。 |
| EV-T6-005c | 2026-05-24 | `curl -s -X POST http://127.0.0.1:19091/api/v1/chat -F message=... -F async=0` | pass | 生成 `case_20260524_000022`，状态 `NEED_HUMAN_CONFIRMATION`，调用 6 个 Gateway 工具。 |
| EV-T6-005d | 2026-05-24 | `mysql ... SELECT ledger_type, COUNT(*) ... WHERE case_id=22` | pass | ledger：`case_state=1`、`gateway_tools=1`、`knowledge_retrieval=1`、`agent_report=1`、`tool_evidence=6`、`final_summary=1`。 |
| EV-T6-005e | 2026-05-24 | `mysql ... SELECT SUM(payload_json LIKE '%"data":%'), SUM(JSON_LENGTH(evidence_refs_json)>0) ...` | pass | `raw_data_key_count=0`，`with_refs=8`，`verifier_final_answer=success`。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T6-005 | 2026-05-24 | 本地 MySQL + Python Agent Platform + mock Gateway API 完整链路 | `case_20260524_000022` 和 MySQL 查询结果 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 新增 Context Ledger DDL | T2 | EV-T6-004 | pass |
| LLM summarize 不接收原始 Gateway `data` | T3/T5 | EV-T5-001 | pass |
| Decision Engine 支持 context ledger 摘要 | T4/T5 | EV-T5-002 | pass |
| 全量测试与扫描 | T6 | EV-T6-001 / EV-T6-002 / EV-T6-003 | pass |
| MySQL 真实写入查询 | T6 | EV-T6-005 | pass |

## 未验证项

- 本轮没有接真实 LLM/Vision，也没有接真实 health-food 生产 readonly adapter；API 验证使用 `local_rules` 和 mock Gateway，只能证明平台上下文隔离、ledger 持久化和 Gateway 调用契约。
- Gateway smoke 使用 `DB_DRIVER=memory`，不声明 Gateway 审计持久化验收；平台 MySQL 持久化已验证。

## 已知噪音

- 首次 API 验证失败：MySQL knowledge `confidence` 以 `Decimal` 返回，Context Ledger payload JSON 序列化失败。已将 repository JSON 序列化改为 `default=str`，并补单测 `test_repository_json_serializes_database_scalar_types`，重新执行 `make test` 通过。
