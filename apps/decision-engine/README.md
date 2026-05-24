# Decision Engine

Python 3.13 decision layer and target Agent orchestrator for the troubleshooting agent.

This service is intentionally small for Phase 1:

- It does not connect to production databases, Redis, logs, or business services.
- It only plans readonly tool calls that must go through Investigation Gateway.
- It keeps explicit budgets for tool calls and missing-field follow-up.
- It can run locally with the Go Gateway deployed elsewhere.
- It is the target home for orchestration logic; Python `apps/agent-platform` embeds it in the main path.

Current orchestration shape:

- `Supervisor` routes each request by issue domain and requires every final answer to pass `Verifier`.
- `Knowledge Agent` checks platform experience first. It can answer directly only when confidence is high, observed cases are enough, and realtime validation is not required.
- `Kline Agent` plans bounded K-line readonly tools after `symbol`、`interval`、`abnormal_time`、`issue_type` are present.
- `Asset Agent` plans bounded asset readonly tools after user/account, `asset_symbol`、`abnormal_time`、`issue_type` are present.
- `HealthFood Agent` plans bounded health-food readonly tools after `uid/user_id` and `issue_type` are present.
- `Local Code Agent` is debug-only. It can inspect an allowlisted local repo only when Gateway evidence is insufficient and `debug_local_code=true`; evidence includes keyword hits, language-structure symbols, bounded call graph edges, resolved call targets, and interface implementation relations.
- `Verifier` deduplicates tool plans, filters unavailable tools, caps tool count, and converts unsafe plans into `need_human`.

The HTTP response keeps the old top-level fields (`action`、`reason`、`tool_plan`) and adds:

- `agent_reports`: per-agent route, skip, ask, plan, or knowledge decision.
- `verification`: final verifier checks, violations, budget, and accepted flag.

Debug-only local code inspection:

```bash
export LOCAL_CODE_REPOS_JSON='{
  "health-food": {
    "repo_path": "/path/to/local/health-food",
    "allowed_globs": ["src/main/java/**", "src/main/resources/**"],
    "deny_globs": ["**/application-prod.yml", "**/*.pem", "**/*secret*"],
    "analysis_backend": "auto",
    "lsif_path": "/path/to/local/health-food/index.lsif",
    "lsp_command": ["jdtls", "--stdio"]
  }
}'
```

`analysis_backend` accepts `auto`, `lightweight`, `tree_sitter`, `lsp`, or `lsif`. The current built-in backend always runs a dependency-light analyzer plus cross-module resolver. `tree_sitter` / `lsp` / `lsif` are explicit backend slots so a stronger semantic index can be plugged in later without changing the Agent evidence contract.

Gateway / adapter may provide `service_name`、`repo_hint`、`suspect_area`, but must not provide local paths. The decision engine maps `service_name` locally. A request must include `debug_local_code=true` and an insufficient evidence status:

```json
{
  "entities": {
    "debug_local_code": "true",
    "gateway_evidence_status": "insufficient",
    "service_name": "health-food",
    "suspect_area": "recommendation mealDataFingerprint"
  }
}
```

The response action is `local_code_inspection`. Evidence contains only relative file paths, matched terms, symbols, call edges, resolved symbols, interface implementation relations, and line numbers; no source snippets are returned. The built-in analyzer is intentionally lightweight:

- Python uses stdlib AST for classes, functions, and calls.
- Java / Go / TypeScript / JavaScript use language-aware structure scanning for classes/functions/methods and bounded call edges.
- Java additionally records simple field receiver types, `implements` relations, and resolves receiver calls across interface and implementation classes when possible.
- `analysis_modes` reports which layers contributed: `keyword`, `language_structure_tree`, `symbol_index`, `call_graph`, `cross_module_call_resolution`, `interface_implementation`.
- `analysis_backends` reports the active lightweight resolver and any configured `tree_sitter` / `lsp` / `lsif` backend slot.
- The analyzer interface can later be backed by tree-sitter, LSP, or LSIF without changing the Local Code Agent safety contract.

Run locally:

```bash
cd apps/decision-engine
python3.13 -m decision_engine --host 127.0.0.1 --port 19092
```

Smoke test:

```bash
curl -s localhost:19092/healthz
curl -s localhost:19092/v1/decisions/plan \
  -H 'Content-Type: application/json' \
  -d '{
    "case": {
      "case_no": "case_dev",
      "issue_domain": "kline",
      "issue_type": "price_mismatch",
      "original_text": "BTCUSDT 1m K线价格不一致"
    },
    "entities": {
      "symbol": "BTCUSDT",
      "interval": "1m",
      "abnormal_time": "2026-05-21T20:00:00+08:00",
      "issue_type": "price_mismatch"
    }
  }'
```
