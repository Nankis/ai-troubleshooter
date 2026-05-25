# Evidence

## Index

| ID | Type | Result | Evidence |
| --- | --- | --- | --- |
| E1 | Unit / integration tests | PASS | `make test` |
| E2 | Secret / whitespace checks | PASS | `make secret-scan`, `git diff --check` |
| E3 | MySQL migration | PASS | canonical schema `ai_troubleshooter`, migrations already applied |
| E4 | Web no-agent guard | PASS | case `case_20260525_000062`, screenshot `artifacts/web-no-agent-blocked-case-62.png` |
| E5 | Web Codex-enabled path | PASS | case `case_20260525_000061`, screenshot `artifacts/web-codex-enabled-case-61.png` |

## Commands

```bash
make test
make secret-scan
git diff --check
```

Results:

- Go tests: PASS.
- Decision Engine Python tests: 20 PASS.
- Agent Platform Python tests: 41 PASS.
- Root Python tests: 4 PASS.
- Secret scan: PASS.
- Diff check: PASS.

## Local Services

Started for validation:

```bash
DB_DRIVER=memory HTTP_PORT=18150 CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false go run ./cmd/investigation-gateway

MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=<local secret> MYSQL_DATABASE=ai_troubleshooter \
AGENT_PLATFORM_PORT=19150 GATEWAY_ENDPOINT=http://127.0.0.1:18150 AI_MODEL_PROFILE=local_rules \
LOCAL_AGENT_WORKSPACE_ROOT=/Users/ginseng/Documents/AI工作区/ai-troubleshooter \
PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m agent_platform
```

Migration:

```bash
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=<local secret> MYSQL_DATABASE=ai_troubleshooter scripts/mysql-migrate.sh
```

Result: migrations skipped/applied successfully on canonical schema `ai_troubleshooter`.

## Web Validation

### E4: No Agent Must Block

Browser flow:

1. Opened `http://127.0.0.1:19150/web`.
2. Disabled the enabled local decision Agent from the right-side panel.
3. Created a new conversation.
4. Submitted: `health-food uid hf-no-agent-050c token quota wrong`.

Observed reply:

```text
[case_20260525_000062] 当前未启用真实决策 Agent，已停止排障。我不会查询 Gateway、平台经验或用 local_rules 给排障结论。请先在右侧启用 Codex/Claude Code 等本地决策 Agent，或配置 Qwen/GPT/Claude/公司模型网关并设置 DECISION_LLM_ENABLED=true 后重新提交。
```

DB verification:

```text
case_no=case_20260525_000062 status=WAITING_USER_REPLY
decision logs:
- classify_extract / success
- decision_agent_ready / blocked
gateway_or_knowledge_or_tools=0
```

Screenshot: `programs/P-2026-050-require-real-decision-agent/artifacts/web-no-agent-blocked-case-62.png`.

### E5: Codex Enabled Can Investigate

Browser flow:

1. Enabled Codex CLI from the right-side local decision Agent panel.
2. Created a new conversation.
3. Submitted: `health-food uid hf-codex-agent-050b token quota wrong`.

Observed:

- Case `case_20260525_000061` reached `NEED_HUMAN_CONFIRMATION`.
- `decision_agent_ready` output recorded `source=local_agent provider=codex`.
- `llm_decision_agent` Agent Run recorded `model_provider=local_agent`, `model_name=codex`.
- Gateway tool calls executed only after the Codex decision Agent was enabled.
- Final answer correctly included mock boundary warning because Gateway was started in mock connector mode.

DB verification summary:

```text
case_no=case_20260525_000061 status=NEED_HUMAN_CONFIRMATION
decision_agent_ready=success source=local_agent provider=codex
agent_run llm_decision_agent model_provider=local_agent model_name=codex
tool_invocation count=6
verifier_final_answer=success
```

Screenshot: `programs/P-2026-050-require-real-decision-agent/artifacts/web-codex-enabled-case-61.png`.

## Known Noise

- Browser plugin text input had virtual clipboard limitations, so ASCII text was entered through DOM keyboard events on the real opened page. The page, UI state, form submission, backend calls and screenshots were still real.
- One intermediate validation case accidentally enabled Codex through an imprecise coordinate click. That case was discarded; clean blocked evidence is `case_20260525_000062` after explicitly disabling the Agent.
- Business Gateway evidence is L2 because the Gateway connector was `CONNECTOR_MODE=mock`; this Program validates the decision-Agent guard, not real health-food production data.
