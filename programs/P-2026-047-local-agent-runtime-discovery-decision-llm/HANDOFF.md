# HANDOFF

当前目标：支持 Local Agent Runtime Discovery，并让本地非交互 agent 可作为 Python 决策层 LLM advisor。

已完成：

- 参考 Multica runtime/provider 模式，新增本机 provider descriptor：Claude Code、Codex、Cursor、Cursor Agent。
- 新增 API：`GET/POST /api/v1/local-agents/discover`、`POST /api/v1/local-agents/enable`、`POST /api/v1/local-agents/probe`，Web 前缀同名。
- 新增 `local_agent` LLM provider，支持 `LOCAL_AGENT_COMMAND` 测试入口、Claude Code、Codex、Cursor Agent 非交互 JSON 调用路径。
- Python Decision Engine 新增 `llm_decision_agent` advisor；advisor 输出仍经过 Verifier。
- Agent Platform 默认只在 `AI_MODEL_PROFILE=local_agent` 或 `DECISION_LLM_ENABLED=true` 时挂载 advisor。
- Web 右侧新增“本地 Agent”：自动发现、手动发现、启用/停用，editor-only provider 禁用。
- README、local runbook、业务接入手册和架构 ADR 已更新。

证据路径：

- `programs/P-2026-047-local-agent-runtime-discovery-decision-llm/EVIDENCE.md`
- `programs/P-2026-047-local-agent-runtime-discovery-decision-llm/RESULT.md`
- `programs/P-2026-047-local-agent-runtime-discovery-decision-llm/web-local-agent-discovery.png`

已运行命令：

- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_agent_platform_fastapi.py'`
- `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest discover -s apps/decision-engine/tests -p 'test_engine.py'`
- `make test`
- `make secret-scan`
- `git diff --check`
- 临时 Web 服务 `127.0.0.1:19147` 浏览器验证后已停止。

commit/push 状态：

- commit: `P-2026-047 local agent runtime discovery`
- push: main。

工作树状态：

- 包含 P-2026-047 代码、文档、Program 证据和截图。
- 未知是否有用户并行改动；提交前需再看 `git status --short`。

下一步：

- 如后续继续真实 `execute=true` 验收，需要新开或继续 Program，并记录真实模型额度/登录态风险。

风险/阻塞：

- 真实 `execute=true` 未跑，避免消耗本机模型额度；如后续要求真实 Claude/Codex 推理验收，需要单独明确。
- Cursor Agent 本机未安装，只验证 editor-only 禁用和代码路径。
