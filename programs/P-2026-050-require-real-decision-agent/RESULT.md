# Result

## Summary

已把“必须使用真实决策 Agent 排查问题”落成硬约束。

核心行为：

- 无本地 Codex/Claude Code 等决策 Agent，且未开启真实 LLM Decision advisor 时，平台只允许 intake 补充询问。
- 输入已经足够进入排障时，平台在查询 Gateway、平台经验和工具调用前阻断。
- 启用 Codex 本地 Agent 后，Decision Engine 才继续执行，并记录 `llm_decision_agent model_provider=local_agent model_name=codex`。

## Changed

- `apps/agent-platform/agent_platform/service.py`
  - 新增 `decision_agent_ready` 守门。
  - 新增平台运行时状态问答，避免“你现在模型用什么”误进入排障。
  - 移除“local_rules 继续排障但做披露”的边界说明。
- `apps/agent-platform/tests/test_agent_platform_fastapi.py`
  - 覆盖无 Agent 阻断、状态询问不排障、启用 Codex 后才可进入工具排查。
- 文档
  - `AGENTS.md`、`README.md`、`apps/agent-platform/README.md`、`docs/local-runbook.md`、`docs/business-onboarding-quickstart.md`、`docs/LESSONS.md` 同步规则。

## Validation

- `make test`: PASS.
- `make secret-scan`: PASS.
- `git diff --check`: PASS.
- Web + MySQL validation:
  - `case_20260525_000062`: no Agent，阻断，Gateway/Knowledge/Tool 日志数量为 0。
  - `case_20260525_000061`: 启用 Codex 后进入排障，`llm_decision_agent` 记录为 `local_agent/codex`。

## Residual Risk

- Gateway 本轮为 mock connector；真实 health-food 生产证据不是本 Program 的验收对象。
- 真实 Qwen/GPT/公司模型网关路径已有单测覆盖配置，完整外部模型验收需在有真实 key 和允许调用时另跑。
