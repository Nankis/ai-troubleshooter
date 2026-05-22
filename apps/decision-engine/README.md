# Decision Engine

Python 3.13 decision layer and target Agent orchestrator for the troubleshooting agent.

This service is intentionally small for Phase 1:

- It does not connect to production databases, Redis, logs, or business services.
- It only plans readonly tool calls that must go through Investigation Gateway.
- It keeps explicit budgets for tool calls and missing-field follow-up.
- It can run locally with the Go Gateway deployed elsewhere.
- It is the target home for orchestration logic; Go `internal/decisionbaseline` is only a local fallback.

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
