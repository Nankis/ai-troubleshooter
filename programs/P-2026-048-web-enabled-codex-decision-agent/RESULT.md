# RESULT

## 结果摘要

- Web 右侧“本地决策 Agent”现在只突出展示可做决策层的 Claude Code / Codex 等 provider；Cursor editor-only 等不可用项被隐藏。
- Web/API 启用 Codex 后，下一个 case 会动态让 Codex 作为 Python Decision Engine 的 `llm_decision_agent` advisor，不需要重启，也不要求主模型切到 `local_agent`。
- 同一 runtime 的本地决策 provider 改为单活；启用 Codex 会关闭 Claude Code。
- Codex CLI 非交互调用改为当前 CLI 兼容参数，并通过真实 `codex exec` 探针验证。
- Agent Run 的 `llm_decision_agent` 模型来源会记录为 `local_agent/codex`，便于 Web 和 MySQL 审计确认。

## 变更范围

- `apps/agent-platform/agent_platform/decision_advisor.py`
- `apps/agent-platform/agent_platform/service.py`
- `apps/agent-platform/agent_platform/local_agents.py`
- `apps/agent-platform/tests/test_agent_platform_fastapi.py`
- `web/static/index.html`
- `README.md`
- `apps/agent-platform/README.md`
- `docs/local-runbook.md`
- `docs/business-onboarding-quickstart.md`
- `docs/architecture-decisions.md`

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| Web 启用本地 agent 动态进入决策层 | done | EV-T2-001, EV-T5-001 |
| 简化本地决策 Agent 展示 | done | EV-T4-001 |
| 修复/验证 Codex CLI 调用参数 | done | EV-T3-001 |
| 补单测 | done | EV-T2-001, EV-T2-002 |
| 启动 MySQL/Gateway/Agent Platform/Web 完整跑 case | done | EV-T5-001, EV-T5-002, EV-T5-003 |
| 更新文档和交接 | done | EV-T6-001 |

## 验证摘要

- Python 编译检查：pass。
- Agent Platform FastAPI 单测：27 tests pass。
- 真实 Codex CLI 探针：pass，返回 wrapper source 为 `local_agent/codex`。
- Web 现场验证：pass，Codex 在页面启用并完成 case `case_20260525_000054`。
- MySQL 落库验证：pass，`llm_decision_agent` 记录为 `model_provider=local_agent`、`model_name=codex`。
- Gateway 审计验证：pass，6 个只读工具调用均 `allowed`。
- `make test`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| 用户在 Web 能看到可启用的本地 agent | pass | EV-T4-001 |
| Codex 可被真实发现、启用并作为决策层 advisor | pass | EV-T3-001, EV-T5-001 |
| 决策层仍通过 Gateway 只读工具取证 | pass | EV-T5-003 |
| 平台持久化和观测可证明使用 Codex | pass | EV-T5-001, EV-T5-002 |

## Commit

- `fc89617 P-2026-048 enable codex local decision agent`

## 残留风险

- 业务 evidence adapter 本轮为 Gateway mock connector，不代表 health-food 生产真实只读接口验收。
- Lark/飞书和 Vision provider 不在本轮范围内。

待补充。
