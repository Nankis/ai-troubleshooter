# Evidence

| ID | Type | Status | Evidence |
| --- | --- | --- | --- |
| EV-T1-001 | code | pass | health-food branch `feature/P-2026-009-health-food-readonly` 新增 `/food-health/v1/readonly/**` 接口。 |
| EV-T1-002 | direct-http | pass | `user/profile` 返回 uid `2054603630081875968`：`registered=true`, `health_goal_summary=goal=减脂塑形`。 |
| EV-T1-003 | direct-http | pass | `recommendation/status` 返回 `2026-05-23` 推荐存在但 `job_status=source_date_mismatch`，source meals 为 `2026-05-14`。 |
| EV-T1-004 | direct-http | pass | 错误 token 调用返回 `UNAUTHORIZED`。 |
| EV-T1-005 | direct-http | pass | `ops/logs/search` 查询 `tb_ai_message_log` 返回真实日志样例，并将 URL 脱敏为 `<url_redacted>`。 |
| EV-T2-001 | unit | pass | `go test ./internal/llm ./internal/decisionbaseline` 通过。 |
| EV-T3-001 | runtime | pass | health-food 真实服务启动在 `http://127.0.0.1:18080/food-health`；启动参数中的只读 token 已脱敏记录。 |
| EV-T3-002 | runtime | pass | ai-troubleshooter Web 服务启动在 `http://127.0.0.1:18088/web`；DB DSN 和 connector token 已脱敏记录。 |
| EV-T4-001 | web-ui | pass | Case A `case_20260524_000008`：`uid:2054603630081875968 今日没有每日推荐`，Web 结论为当天无餐食记录，截图：`evidence/screenshots/case-a-daily-missing.png`。 |
| EV-T4-002 | web-ui | pass | Case B `case_20260524_000005`：`uid:999999999999 推荐数据不准`，Web 结论要求反馈方确认正确 uid，截图：`evidence/screenshots/case-b-missing-uid.png`。 |
| EV-T4-003 | web-ui | pass | Case C `case_20260524_000007`：`uid:2054603630081875968 2026-05-23 推荐数据不准`，Web 结论为 `source_date_mismatch`，截图：`evidence/screenshots/case-c-source-date-mismatch.png`。 |
| EV-T4-004 | web-ui | pass | Case D `case_20260524_000009`：`uid:2054603630081875968 今日 token 消耗数量不对`，Web 结论显示真实 token 账户健康，截图：`evidence/screenshots/case-d-token-quota.png`。 |
| EV-T4-005 | mysql | pass | `ai_troubleshooter.tb_troubleshoot_case/message/ai_decision_log` 反查到上述 4 个 case、用户消息、agent 回复、分类/抽取/工具调用/总结决策日志。 |
| EV-T4-006 | mysql | pass | `meow_pas` 真实数据反查：`2054603630081875968` 存在；`999999999999` 不存在；`2026-05-23` 推荐记录引用 `2026-05-14` 餐食 ID；token 余额为真实账户数据。 |
| EV-T4-007 | web-ui/mysql | pass | Case E `case_20260524_000010`：验证 Web 新建 case 后会把抽取到的业务 uid 回写到 `tb_troubleshoot_case.uid=2054603630081875968`，截图：`evidence/screenshots/case-e-uid-persistence.png`。 |
| EV-T5-001 | test | pass | `make test` 通过：Go 全量测试、Python decision-engine 14 个单测、根目录 Python 3 个单测均通过。 |
| EV-T5-002 | compile | pass | health-food `mvn -pl health-food-srv -am -DskipTests compile` 通过。 |
| EV-T5-003 | security | pass | `make secret-scan` 通过；`git diff` 未发现本轮新增密码、readonly token 或 Bearer 明文。 |
| EV-T5-004 | style | pass | 两个代码仓库 `git diff --check` 通过。 |

## 待补

- 提交和推送。
