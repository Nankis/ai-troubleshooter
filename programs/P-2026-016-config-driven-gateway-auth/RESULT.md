# RESULT

已完成。

## 交付内容

- Gateway 支持 `GATEWAY_AGENT_CONFIG_JSON` 和 `GATEWAY_AGENT_CONFIG_FILE` 配置 agents。
- agent 配置支持 `agent_id`、`status`、`bearer_token_env`、`allowed_scopes`、`allowed_tools`、`allowed_chat_ids`、`rate_limit_qps`。
- `GATEWAY_BEARER_TOKENS=agent_id:token` 继续兼容，便于旧部署和平滑迁移。
- `GATEWAY_AGENT_ID` 控制 dev-server / worker / baseline-orchestrator 的 runner agent id，默认仍是 `business-troubleshooter-v1`。
- 新增 `configs/gateway-agents.example.json`。
- README、`docs/gateway-security.md`、`docs/deployment-checklist.md`、`configs/config.example.yaml` 已同步。

## 其它硬编码点检查

- 已修复：Gateway policy agent/scopes/tools 从代码默认值升级为配置。
- 已修复：cmd runner agent id 从代码常量升级为 `GATEWAY_AGENT_ID`。
- 保留：Tool Registry 仍由 Go 注册默认工具。原因是工具 handler、connector 类型和参数边界仍在代码内，完全动态注册需要单独 Program。
- 保留：`CONNECTOR_MODE=mock` 和 `LLM_PROVIDER=local_rules` 是本地 demo fallback，不是生产接入阻塞；文档已明确生产需要接真实 readonly adapter / LLM provider。

## 验收结果

- 实际启动 `cmd/dev-server`，加载 `configs/gateway-agents.example.json`。
- `health-food-readonly-agent` 使用环境变量 token 成功调用 health-food 工具。
- 同一 agent 调资产工具被 scope 拒绝。
- 同一 agent 从未授权 chat 调 health-food 工具被 chat allowlist 拒绝。

## 验证命令

- `go test ./internal/config ./internal/gateway ./internal/policy ./cmd/dev-server ./cmd/worker ./cmd/baseline-orchestrator`：通过。
- `make test`：通过。
- `git diff --check`：通过。
- `python3.13 scripts/secret-scan.py --mode all`：通过。
