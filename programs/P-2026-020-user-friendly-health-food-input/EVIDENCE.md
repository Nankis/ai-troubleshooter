# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `internal/caseflow/state.go`：health-food 必填字段去掉 `abnormal_time`。 |
| EV-T2-001 | unit | `internal/llm/rules_test.go` 覆盖 `uid:123456 用户反馈 今日 token消耗 数量不对` 归类为 `AI配额异常`。 |
| EV-T3-001 | unit | `internal/decisionbaseline/orchestrator_test.go` 覆盖缺时间追问不包含 `timezone` / `Asia/Shanghai`，并说明默认北京时间 UTC+8。 |
| EV-T4-001 | unit | `internal/decisionbaseline/orchestrator_test.go` 覆盖 health-food 点查工具不携带 `start_time/end_time`，日志窗口不超过 30 分钟。 |
| EV-T5-001 | unit | `go test ./internal/llm ./internal/caseflow ./internal/decisionbaseline` 通过。 |
| EV-T5-002 | browser | 本地启动 dev-server，Web UI 打开用户示例 case，状态 `NEED_HUMAN_CONFIRMATION`，无 timezone 追问，profile/quota/logs/similar tools 均成功。 |

## 本地 Web 验证

启动命令：

```text
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=memory CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules go run ./cmd/dev-server
```

输入：

```text
uid:123456 用户反馈 今日 token消耗 数量不对
```

结果：

- case `case_20260523_000001` 进入 `NEED_HUMAN_CONFIRMATION`。
- 问题类型识别为 `AI配额异常`，领域为 `health_food`。
- 没有出现 `timezone`、`Asia/Shanghai` 或 `请补充 异常发生`。
- 工具证据：
  - `get_health_food_user_profile`：`registered=true membership_level=1`
  - `get_health_food_ai_quota`：`abnormal: tokens=0 daily_chat=30/30`
  - `search_logs_by_service`：`found 3 log samples`
  - `get_similar_cases`：`found 5 similar cases`
