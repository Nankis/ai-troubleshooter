# RESULT

## 结果摘要

已支持 Local Agent Runtime Discovery，并把本地非交互 agent 接入为 Python 决策层可选 LLM advisor：

- Web/API 可发现本机 Claude Code、Codex、Cursor、Cursor Agent。
- Web/API 可显式启用 llm-capable provider；Cursor editor-only provider 不能启用为决策 LLM。
- 新增 `AI_MODEL_PROFILE=local_agent`，可通过本地 CLI 获取 JSON 输出。
- Python Decision Engine 新增 `llm_decision_agent` advisor 路径，输出仍必须经过 Verifier。
- 文档补齐本地 agent 配置、API、边界和 Multica 借鉴点。

## 变更范围

- `apps/agent-platform`：新增 `local_agents.py`、`decision_advisor.py`，扩展 config/LLM/server/service。
- `apps/decision-engine`：Supervisor 支持可选 `DecisionAdvisor`。
- `web/static/index.html`：右侧新增“本地 Agent”发现和启用 UI。
- `README.md`、`docs/local-runbook.md`、`docs/business-onboarding-quickstart.md`、`docs/architecture-decisions.md`：更新架构和运行说明。
- `programs/P-2026-047-*`：记录决策、证据和交接。

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| 梳理现有 LLM、Decision Engine 和 runtime 主路径 | 完成 | EV-001 |
| 实现 Local Agent Discovery 和 runtime 注册 API | 完成 | EV-001, EV-005 |
| 实现 `local_agent` LLM provider | 完成 | EV-001 |
| 实现 Decision Engine `llm_decision_agent` advisory 路径 | 完成 | EV-001 |
| 更新 Web/API 文档和本地运行手册 | 完成 | EV-004 |
| 增加单测、本地 discovery/probe 验证和回归 | 完成 | EV-001, EV-002, EV-003, EV-004, EV-005 |

## 验证摘要

- `make test`：PASS。
- `make secret-scan`：PASS。
- `git diff --check`：PASS。
- Browser：PASS，`/web` 实际展示本机 providers，Codex 可启用，Cursor editor-only 禁用。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| 可自动发现本机 agent/AI 工具 | 通过 | EV-004, EV-005 |
| 可显式启用本地 LLM provider | 通过 | EV-001, EV-004 |
| 本地 LLM 参与决策层但不能绕过 Verifier | 通过 | EV-001 |
| Go Gateway 不新增 LLM/决策职责 | 通过 | EV-002 |
| 敏感信息不入仓 | 通过 | EV-003 |

## Commit

- commit: `P-2026-047 local agent runtime discovery`
- push: main。

## 残留风险

- 真实 `execute=true` 会消耗本机 Claude/Codex 登录态或模型额度，本轮未执行真实模型调用；后续要做真实本地 LLM 验收时单独记录。
- Cursor Agent 的真实 CLI 参数可能随版本变化；当前本机未安装，仅保留支持路径和失败显式报错。
- Web 启用 provider 是平台登记动作；生产运行是否使用它仍由 `AI_MODEL_PROFILE=local_agent`、`LOCAL_AGENT_PROVIDER` 和 `DECISION_LLM_ENABLED` 控制，避免运行时误切模型。
