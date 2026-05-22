#!/usr/bin/env python3
"""Local readonly adapter for health-food integration smoke tests."""

from __future__ import annotations

import json
import os
import time
import urllib.request
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


PORT = int(os.getenv("MOCK_HEALTH_FOOD_ADAPTER_PORT", "19081"))
API_KEY = os.getenv("MOCK_CONNECTOR_API_KEY", "")
SCENARIO = os.getenv("MOCK_HEALTH_FOOD_SCENARIO", "recommendation_missing")
HEALTH_FOOD_BASE_URL = os.getenv("HEALTH_FOOD_BASE_URL", "http://127.0.0.1:18080").rstrip("/")


def now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def read_health_food_alive() -> dict:
    try:
        with urllib.request.urlopen(f"{HEALTH_FOOD_BASE_URL}/food-health/sys/alive", timeout=1.5) as resp:
            body = resp.read().decode("utf-8", errors="replace")
            return {"reachable": True, "status": resp.status, "body": body[:80]}
    except Exception as exc:  # noqa: BLE001 - local smoke adapter should report any failure.
        return {"reachable": False, "error": str(exc)}


def envelope(request_id: str, data: dict, warnings: list[str] | None = None) -> dict:
    return {
        "request_id": request_id,
        "source": "health-food-readonly-adapter/mock",
        "queried_at": now_iso(),
        "data_updated_at": now_iso(),
        "version": "mock-v1",
        "data": data,
        "warnings": warnings or [],
    }


def get_uid(params: dict) -> str:
    return str(params.get("uid") or params.get("user_id") or "hf_user_001")


def recommendation_date(params: dict) -> str:
    return str(params.get("recommendation_date") or params.get("at_time") or now_iso())[:10]


def handle_health_food(path: str, payload: dict) -> tuple[int, dict]:
    params = payload.get("params") or {}
    request_id = payload.get("request_id") or f"req_mock_{int(time.time() * 1000)}"
    uid = get_uid(params)
    alive = read_health_food_alive()
    base = {"uid": uid, "health_food_alive": alive, "scenario": SCENARIO}

    if path == "/v1/readonly/health-food/user/profile":
        data = {
            **base,
            "registered": True,
            "membership_level": 1,
            "health_goal_summary": "fat loss, target 1800 kcal/day",
            "latest_device": {"platform": "ios", "app_version": "1.2.3"},
            "updated_at": now_iso(),
            "source": "health-food",
            "version": "local-mock",
        }
        return 200, envelope(request_id, data)

    if path == "/v1/readonly/health-food/ai/quota":
        abnormal = SCENARIO == "quota_exhausted"
        data = {
            **base,
            "membership_level": 1,
            "available_tokens": "0" if abnormal else "1200",
            "daily_chat_count": 30 if abnormal else 3,
            "limit_chat": 30,
            "last_reset_date": now_iso(),
            "abnormal": abnormal,
            "reason": "daily chat count reached limit while user still has membership" if abnormal else "quota is normal",
            "data_updated_at": now_iso(),
        }
        return 200, envelope(request_id, data)

    if path == "/v1/readonly/health-food/meals/range":
        missing = SCENARIO == "recommendation_missing"
        data = {
            **base,
            "meal_count": 2 if missing else 4,
            "missing_meal_ids": ["meal_missing_dinner"] if missing else [],
            "meal_data_fingerprint": "fingerprint_stale" if missing else "fingerprint_current",
            "meals": [
                {"meal_id": "meal_breakfast", "meal_name": "breakfast", "meal_time": now_iso()},
                {"meal_id": "meal_lunch", "meal_name": "lunch", "meal_time": now_iso()},
            ],
            "data_updated_at": now_iso(),
        }
        return 200, envelope(request_id, data)

    if path == "/v1/readonly/health-food/recommendation/status":
        missing = SCENARIO == "recommendation_missing"
        data = {
            **base,
            "recommend_date": recommendation_date(params),
            "has_recommendation": not missing,
            "job_status": "failed" if missing else "success",
            "meal_count": 2 if missing else 4,
            "meal_data_fingerprint": "fingerprint_stale" if missing else "fingerprint_current",
            "generated_at": None if missing else now_iso(),
            "failure_reason": "meal_data_fingerprint did not refresh after dinner upload" if missing else "",
            "source_meal_ids": ["meal_breakfast", "meal_lunch"],
        }
        return 200, envelope(request_id, data)

    return 404, {"code": "NOT_FOUND", "error": f"unknown path {path}"}


def handle_ops(path: str, payload: dict) -> tuple[int, dict]:
    params = payload.get("params") or {}
    request_id = payload.get("request_id") or f"req_mock_{int(time.time() * 1000)}"
    if path == "/v1/readonly/ops/logs/search":
        service_name = params.get("service_name") or "health-food"
        keyword = params.get("keyword") or ""
        if SCENARIO == "quota_exhausted":
            message = "TokenAccountService rejected request: daily chat count limit reached"
        else:
            message = "RecommendFoodJob skipped generation: meal_data_fingerprint unchanged"
        data = {
            "service_name": service_name,
            "total": 1,
            "samples": [
                {
                    "time": now_iso(),
                    "level": "error",
                    "service": service_name,
                    "message": message,
                    "keyword": keyword,
                }
            ],
        }
        return 200, envelope(request_id, data)
    if path == "/v1/readonly/ops/cases/similar":
        data = {
            "items": [
                {
                    "case_no": "case_hf_mock_001",
                    "issue_domain": "health_food",
                    "issue_type": params.get("issue_type") or "unknown",
                    "summary": "similar health-food mock case",
                    "score": 0.87,
                }
            ]
        }
        return 200, envelope(request_id, data)
    return 404, {"code": "NOT_FOUND", "error": f"unknown path {path}"}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/healthz":
            self.write_json(200, {"ok": True, "scenario": SCENARIO, "health_food": read_health_food_alive()})
            return
        self.write_json(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        if API_KEY:
            expected = f"Bearer {API_KEY}"
            if self.headers.get("Authorization") != expected:
                self.write_json(401, {"code": "UNAUTHORIZED", "error": "invalid adapter token"})
                return
        length = int(self.headers.get("Content-Length") or "0")
        raw = self.rfile.read(length)
        try:
            payload = json.loads(raw or b"{}")
        except json.JSONDecodeError:
            self.write_json(400, {"code": "BAD_JSON", "error": "invalid json"})
            return
        if self.path.startswith("/v1/readonly/health-food/"):
            status, body = handle_health_food(self.path, payload)
        elif self.path.startswith("/v1/readonly/ops/"):
            status, body = handle_ops(self.path, payload)
        else:
            status, body = 404, {"code": "NOT_FOUND", "error": f"unknown path {self.path}"}
        self.write_json(status, body)

    def write_json(self, status: int, body: dict) -> None:
        data = json.dumps(body).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, fmt: str, *args: object) -> None:
        print(f"{self.log_date_time_string()} {fmt % args}")


def main() -> None:
    server = ThreadingHTTPServer(("127.0.0.1", PORT), Handler)
    print(f"mock health-food readonly adapter listening on http://127.0.0.1:{PORT}")
    server.serve_forever()


if __name__ == "__main__":
    main()
