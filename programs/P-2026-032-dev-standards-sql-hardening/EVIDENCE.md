# Evidence

| ID | Type | Status | Evidence |
| --- | --- | --- | --- |
| EV-001 | research | pass | 参考 OWASP SQL Injection Prevention、GitHub CodeQL SQL injection、Uber Go Style Guide、Google Python Style Guide，结论写入 `AGENTS.md`。 |
| EV-002 | audit | pass | `rg` 扫描 Go/Python SQL、`fmt.Sprintf`、f-string SQL、动态 `strings.Join` 查询。 |
| EV-003 | code | pass | `scripts/real-health-food-readonly-adapter.py` 已从 MySQL CLI/f-string SQL 改为 PyMySQL 参数绑定。 |
| EV-004 | code | pass | Go 动态查询抽出 builder，新增测试确保注入 payload 只进入 args。 |
| EV-005 | test | pass | `make test` 通过：Go 全量测试、Python decision-engine 14 个单测、根目录 Python 4 个单测。 |
| EV-006 | test | pass | `python3.13 -m py_compile scripts/real-health-food-readonly-adapter.py && python3.13 -m unittest tests/test_real_health_food_readonly_adapter.py` 通过。 |
| EV-007 | security | pass | `make secret-scan` 通过。 |
| EV-008 | static | pass | `go vet ./...`、`git diff --check` 通过。 |
| EV-009 | audit | pass | `rg` 未发现 `mysql_query(f"...")`、SQL f-string、SQL `fmt.Sprintf` 或 `mysql -e` 模式。 |

## 待补

- 无。
