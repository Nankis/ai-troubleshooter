# Result

## Summary

已修复“普通/平台咨询继承旧 case 上下文继续查 Gateway”的问题。

现在 Web Chat 会先看最新用户消息：

- 模型/Agent/平台配置/用户纠错/普通咨询：走 `llm_decision_agent` direct chat。
- 真实生产问题或补充 uid/时间等字段：继续走排障流程。
- direct chat 路径禁止 Gateway、平台经验和工具计划。

## Changed

- `apps/agent-platform/agent_platform/service.py`
  - `_upsert_case` 在用户追加消息时重新打开非 active case，允许处理 follow-up。
  - 新增最新用户消息路由。
  - 新增 `decision_agent_direct_answer` 路径和 Agent Run。
- `apps/agent-platform/agent_platform/llm.py`
  - 新增 `answer_chat()`，供决策层 Agent 直接回答非排障消息。
- `apps/agent-platform/tests/test_agent_platform_fastapi.py`
  - 覆盖已有业务 case 内问模型状态。
  - 覆盖已有业务 case 内吐槽 Claude Code，用 Codex direct answer，且不查 Gateway。
- 文档和复盘
  - `AGENTS.md`、`README.md`、`apps/agent-platform/README.md`、`docs/LESSONS.md`。

## Validation

- `make test`: PASS.
- `make secret-scan`: PASS.
- `git diff --check`: PASS.
- Web + MySQL: `case_20260525_000063` 通过。
  - first message: health-food issue used Gateway mock evidence.
  - follow-up: `my claude code cannot work` used `llm_decision_agent` direct chat.
  - follow-up did not add Gateway/Knowledge/Tool records.
- Web + MySQL: `case_20260525_000064` 通过。
  - message: `现在是用什么模型`.
  - answer came from `llm_decision_agent / direct_chat / local_agent / codex`.
  - decision logs show `decision_agent_direct_answer=1`, `tool_invocation=0`.
- Web + MySQL: `case_20260525_000065` 通过最终话术验收。
  - answer explicitly separates real decision Agent `codex/codex` from platform main LLM profile `local_rules/rules-v1`.
  - decision logs show `decision_agent_direct_answer=1`, `tool_invocation=0`.

## Residual Risk

- 本轮 Gateway 仍是 mock connector；真实业务证据不是本 Program 的验收目标。
- direct chat 依赖本地 Codex/Claude Code 或真实 LLM provider 的 JSON 输出能力；失败时平台会显式返回 Agent 失败，不会 fallback 到规则或 Gateway。
