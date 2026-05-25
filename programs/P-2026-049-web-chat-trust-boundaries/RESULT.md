# RESULT

## 结果摘要

- 问候语、闲聊、无有效排障线索的输入不再命中平台经验，也不调用 Gateway 工具；系统会追问具体生产问题。
- 有效问题在未启用本地决策 Agent 且主模型为 `local_rules` 时仍可走规则编排，但最终回复会明确说明来源。
- Gateway mock evidence 会在最终回复中前置警告，避免被当成真实业务数据。
- Web 输入框支持 Enter 发送、Shift+Enter 换行。

## 变更范围

- `apps/decision-engine/decision_engine/agent_team.py`
- `apps/decision-engine/tests/test_engine.py`
- `apps/agent-platform/agent_platform/service.py`
- `apps/agent-platform/tests/test_agent_platform_fastapi.py`
- `web/static/index.html`
- `programs/P-2026-049-web-chat-trust-boundaries/*`

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| 修复低信号误答 | done | EV-T2-001, EV-T4-001 |
| 标注 mock/local_rules 来源 | done | EV-T3-001, EV-T5-001 |
| Enter 发送 | done | EV-T4-001 |
| 单测和 Web/MySQL 验证 | done | EV-T2-001, EV-T3-001, EV-T5-001, EV-T6-001 |

## 验证摘要

- `make test`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。
- Web 现场验证：pass，case `case_20260525_000056` 和 `case_20260525_000057`。
- MySQL 验证：pass，case 56 无 tool audit；case 57 有 6 个 mock Gateway tool audit。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| 问候语不会再得到平台经验假答 | pass | EV-T2-001, EV-T4-001 |
| 关闭本地决策 Agent 后仍回复时会说明规则来源 | pass | EV-T3-001, EV-T5-001 |
| mock 数据必须显式标注 | pass | EV-T3-001, EV-T5-001 |
| Enter 可发送 | pass | EV-T4-001 |

## Commit

- `cab210b P-2026-049 harden web chat trust boundaries`

## 残留风险

- 本轮没有替换 mock Gateway connector；真实 health-food adapter 仍需另起 Program 验证。
