# Evidence

## Index

| ID | Type | Result | Evidence |
| --- | --- | --- | --- |
| E1 | Unit / integration tests | PASS | `make test` |
| E2 | Web + MySQL | PASS | case `case_20260525_000063` |
| E3 | Screenshot | PASS | `artifacts/web-direct-agent-followup-case-63.png`, `artifacts/web-runtime-status-direct-agent-case-64.png`, `artifacts/web-runtime-status-direct-agent-case-65.png` |
| E4 | Secret / whitespace checks | PASS | `make secret-scan`, `git diff --check` |

## Test Commands

```bash
PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py
make test
make secret-scan
git diff --check
```

Observed:

- Target Agent Platform test file: 32 PASS after implementation.
- Full `make test`: Go + Decision Engine + Agent Platform + root tests PASS. Agent Platform total now 43 tests.
- `make secret-scan`: PASS.
- `git diff --check`: PASS.

## Web + MySQL Validation

Services:

```bash
DB_DRIVER=memory HTTP_PORT=18151 CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false go run ./cmd/investigation-gateway

MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=<local secret> MYSQL_DATABASE=ai_troubleshooter \
AGENT_PLATFORM_PORT=19151 GATEWAY_ENDPOINT=http://127.0.0.1:18151 AI_MODEL_PROFILE=local_rules \
LOCAL_AGENT_WORKSPACE_ROOT=/Users/ginseng/Documents/AI工作区/ai-troubleshooter \
PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m agent_platform
```

Browser flow:

1. Opened `http://127.0.0.1:19151/web`.
2. Enabled Codex CLI as local decision Agent in the right panel.
3. Created a new case and submitted `health-food uid hf-direct-routing-051 token quota wrong`.
4. Waited for the first production-style investigation to finish. It used mock Gateway, as expected for this local validation.
5. In the same case, submitted `my claude code cannot work`.
6. Confirmed the follow-up was answered by `llm_decision_agent` direct chat, not by Gateway.

Observed Web reply for follow-up:

```text
[case_20260525_000063] 当前输入显示决策 Agent 是本地 Codex，不是 Claude Code；主 LLM 是 local_rules/rules-v1。如果你希望使用 Claude Code，需要先在本机修复或启用 Claude Code 本地 Agent，包括确认已安装、已登录、可运行，并更新平台配置指向 Claude Code。
```

DB verification:

```text
case_no=case_20260525_000063 status=WAITING_USER_REPLY
decision_agent_direct_answer success count=1
tool_invocation success count=6
```

The six `tool_invocation` rows are from the first health-food investigation only. The follow-up added exactly one `decision_agent_direct_answer` and no extra Gateway/Knowledge/Tool records.

Agent run verification:

```text
llm_decision_agent / direct_chat / model_provider=local_agent / model_name=codex
```

Screenshot:

- `programs/P-2026-051-direct-agent-chat-routing/artifacts/web-direct-agent-followup-case-63.png`

Additional runtime-status validation after tightening routing:

1. Restarted Python Agent Platform on `127.0.0.1:19151` with platform MySQL.
2. Confirmed local Codex provider was installed, LLM-capable and enabled in Local Agent Runtime.
3. Submitted `现在是用什么模型` through `/web/api/chat`.
4. Observed case `case_20260525_000064`:

```text
case_no=case_20260525_000064 status=WAITING_USER_REPLY
decision_agent_direct_answer=1
tool_invocation=0
agent_runs:
  supervisor / case_process / completed / local_rules / rules-v1
  llm_decision_agent / direct_chat / completed / local_agent / codex
```

Web screenshot:

- `programs/P-2026-051-direct-agent-chat-routing/artifacts/web-runtime-status-direct-agent-case-64.png`

Final wording validation after prompt tightening:

```text
case_no=case_20260525_000065 status=WAITING_USER_REPLY
decision_agent_direct_answer=1
tool_invocation=0
agent_runs:
  supervisor / case_process / completed / local_rules / rules-v1
  llm_decision_agent / direct_chat / completed / local_agent / codex
reply explicitly says provider=codex/model=codex is the real decision Agent for this answer,
and local_rules/rules-v1 is only the platform main LLM profile.
```

Web screenshot:

- `programs/P-2026-051-direct-agent-chat-routing/artifacts/web-runtime-status-direct-agent-case-65.png`

## Known Noise

- Gateway used `CONNECTOR_MODE=mock`; this Program validates routing and direct answer isolation, not real health-food production evidence.
- Browser plugin keyboard input still required DOM keyboard events for ASCII text. The page, UI events, backend calls, MySQL records and screenshot are real.
