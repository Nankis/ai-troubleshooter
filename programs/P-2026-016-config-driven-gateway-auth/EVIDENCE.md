# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 结论 |
| --- | --- | --- | --- |
| EV-T1-001 | review | T1 | completed：识别并处理 Gateway policy / runner agent id 硬编码 |
| EV-T2-001 | implementation | T2 | completed：支持 JSON/file agent 配置和 token env |
| EV-T3-001 | implementation | T3 | completed：dev-server / worker / baseline runner 使用 `GATEWAY_AGENT_ID` |
| EV-T4-001 | test/docs | T4 | completed：测试、README、Gateway 安全文档和示例配置已更新 |
| EV-T5-001 | command | T5 | passed：完整测试、diff check、secret scan 通过 |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-23 | `go test ./internal/config ./internal/gateway ./internal/policy ./cmd/dev-server ./cmd/worker ./cmd/baseline-orchestrator` | passed | 配置解析、Gateway policy 和命令入口编译测试通过 |
| EV-T5-001 | 2026-05-23 | `make test` | passed | Go 全量测试、Python decision-engine、MCP adapter 单测通过 |
| EV-T5-002 | 2026-05-23 | `git diff --check` | passed | 无 whitespace 错误 |
| EV-T5-003 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | passed | Secret scan passed |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | 硬编码点扫描 | `DefaultAgents()`、cmd runner `AgentID`、文档里的 legacy token 均已识别 | Gateway 权限和 runner agent id 是本轮必须处理项 |
| EV-T2-001 | 2026-05-23 | `configs/gateway-agents.example.json` + env token 启动 dev-server | `health-food-readonly-agent` 使用 `HEALTH_FOOD_AGENT_GATEWAY_TOKEN` 成功认证 | agent/token 配置化可用 |
| EV-T2-002 | 2026-05-23 | health-food agent 调 `get_health_food_recommendation_status` | HTTP 200，summary 返回推荐缺失证据 | 配置化 scope/tool 授权允许正确工具 |
| EV-T2-003 | 2026-05-23 | health-food agent 调 `get_asset_snapshot` | HTTP 403，`scope "asset:read" is not allowed` | 配置化 scope 拒绝生效 |
| EV-T2-004 | 2026-05-23 | health-food agent 使用未授权 chat 调 health-food 工具 | HTTP 403，`chat "oc_not_allowed" is not allowed` | chat allowlist 生效 |
