# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 结论 |
| --- | --- | --- | --- |
| EV-T1-001 | implementation | T1 | completed：MCP readonly adapter 已实现 |
| EV-T2-001 | implementation | T2 | completed：health-food MCP mock server 已实现 |
| EV-T3-001 | documentation | T3 | completed：README、MCP adapter 文档、配置样例已更新 |
| EV-T4-001 | runtime | T4 | passed：实际启动 MCP adapter 和 dev-server，Gateway 调用 MCP 成功 |
| EV-T5-001 | command | T5 | passed：完整测试、diff check、secret scan 通过 |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | `python3.13 -m py_compile scripts/mcp-readonly-adapter.py scripts/mock-health-food-mcp-server.py` | passed | adapter 和 MCP mock server 语法检查通过 |
| EV-T4-001 | 2026-05-23 | `MCP_ADAPTER_API_KEY=... MCP_READONLY_ADAPTER_PORT=19085 python3.13 scripts/mcp-readonly-adapter.py` | passed | adapter 启动并完成 MCP initialize / tools/list |
| EV-T4-002 | 2026-05-23 | `HTTP_PORT=18086 CONNECTOR_MODE=http ... go run ./cmd/dev-server` | passed | dev-server 实际启动 |
| EV-T4-003 | 2026-05-23 | `curl POST /tools/get_health_food_recommendation_status/invoke` | passed | Gateway -> readonly adapter -> MCP `tools/call` 成功 |
| EV-T4-004 | 2026-05-23 | `curl POST /web/api/chat` | passed | Web Chat 实际排查 health-food 推荐缺失，5 个工具均成功 |
| EV-T5-001 | 2026-05-23 | `make test` | passed | Go 全量测试、decision-engine 14 个单测、MCP adapter 1 个单测通过 |
| EV-T5-002 | 2026-05-23 | `git diff --check` | passed | 无 whitespace 错误 |
| EV-T5-003 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | passed | Secret scan passed |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-23 | MCP adapter `/healthz` | 返回 `health_food_recommendation_status`、`health_food_search_logs` 等 6 个 MCP tools 和 6 个 readonly routes | MCP tools/list 接入成功 |
| EV-T4-003 | 2026-05-23 | Gateway 直接调用 `get_health_food_recommendation_status` | `status=success`，`has_recommendation=false`，`job_status=failed`，`meal_data_fingerprint=fingerprint_stale`，summary 包含 `meal_data_fingerprint did not refresh` | Gateway 到 MCP tool 链路成功 |
| EV-T4-004 | 2026-05-23 | Web Chat 输入 `health-food uid:hf_user_001 2026-05-23 10:00 今日推荐没有生成，请排查` | AI decision logs 中 `get_health_food_user_profile`、`get_health_food_meal_records`、`get_health_food_recommendation_status`、`search_logs_by_service`、`get_similar_cases` 均为 success | 用户入口到 MCP tool 链路成功 |
