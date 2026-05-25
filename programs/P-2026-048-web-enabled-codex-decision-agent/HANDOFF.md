# HANDOFF

当前目标：让 Web 启用的 Codex 本地 agent 真实进入 Python 决策层，并完成真实流程验证。

已完成：

- 新增 `RuntimeLLMDecisionAdvisor`，Web/API 启用的本地 provider 会在下一个 case 动态进入 Python Decision Engine。
- 本地决策 provider 改为单活，避免 Claude Code / Codex 同时 enabled 后选择不可预期。
- Web 右侧区域改为“本地决策 Agent”，只突出已安装且 `llm_capable=true` 的 provider。
- 修复 Codex CLI 参数兼容：移除通用 `--output-schema`，按当前 CLI help 决定是否追加 `--ask-for-approval never`。
- `llm_decision_agent` 的 Agent Run 模型来源会记录为 `local_agent/codex`。
- README、Agent Platform README、本地运行手册、业务接入文档和架构决策文档已同步。

证据路径：

- `programs/P-2026-048-web-enabled-codex-decision-agent/EVIDENCE.md`
- `programs/P-2026-048-web-enabled-codex-decision-agent/RESULT.md`
- `programs/P-2026-048-web-enabled-codex-decision-agent/web-codex-enabled.png`
- `programs/P-2026-048-web-enabled-codex-decision-agent/web-codex-provider-result.png`

已运行命令：

- `python3.13 -m py_compile apps/agent-platform/agent_platform/decision_advisor.py apps/agent-platform/agent_platform/service.py apps/agent-platform/agent_platform/local_agents.py`
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_agent_platform_fastapi.py'`
- `complete_json_with_local_agent(... provider_id='codex' ...)`
- `MYSQL_DATABASE=ai_troubleshooter make migrate-mysql`
- `go run ./cmd/investigation-gateway` with `CONNECTOR_MODE=mock`
- `.venv/bin/python -m agent_platform` with MySQL + local Gateway
- Browser opened `http://127.0.0.1:19148/web`, enabled Codex, submitted case `case_20260525_000054`
- MySQL checked `tb_troubleshoot_case`、`tb_troubleshoot_agent_run`、`tb_troubleshoot_agent_runtime`、`tb_troubleshoot_tool_call_audit`
- `make test`
- `make secret-scan`
- `git diff --check`

当前工作树：

- 主变更已提交并推送为 `fc89617 P-2026-048 enable codex local decision agent`；收口提交已推送为 `d83a53d docs: close P-2026-048 handoff`。

下一步：

- 无。后续可基于真实 health-food adapter、Vision 或 Lark/飞书真实回调另起 Program。

风险：

- 业务证据侧本轮是 Gateway mock connector，只能证明平台链路、真实 Codex advisor、MySQL 和 Gateway 审计；不能声称 health-food 生产真实 adapter 已验证。
- Lark/飞书和 Vision provider 不在本轮范围内。
