#!/usr/bin/env python3
"""A tiny health-food MCP server for local adapter verification."""

from __future__ import annotations

import json
import os
import sys
from datetime import datetime, timezone
from typing import Any


SCENARIO = os.getenv("MOCK_HEALTH_FOOD_SCENARIO", "recommendation_missing")
PROTOCOL_VERSION = os.getenv("MCP_PROTOCOL_VERSION", "2025-06-18")


def now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def uid(args: dict[str, Any]) -> str:
    return str(args.get("uid") or args.get("user_id") or "hf_user_001")


def recommendation_date(args: dict[str, Any]) -> str:
    return str(args.get("recommendation_date") or args.get("at_time") or now_iso())[:10]


def tool_schema(required: list[str]) -> dict[str, Any]:
    return {
        "type": "object",
        "properties": {
            "uid": {"type": "string"},
            "user_id": {"type": "string"},
            "recommendation_date": {"type": "string"},
            "start_time": {"type": "string"},
            "end_time": {"type": "string"},
            "service_name": {"type": "string"},
            "keyword": {"type": "string"},
            "limit": {"type": "integer"},
        },
        "required": required,
    }


TOOLS = [
    {
        "name": "health_food_user_profile",
        "description": "Readonly health-food user profile summary.",
        "inputSchema": tool_schema(["uid"]),
    },
    {
        "name": "health_food_ai_quota",
        "description": "Readonly health-food AI quota summary.",
        "inputSchema": tool_schema(["uid"]),
    },
    {
        "name": "health_food_meal_records",
        "description": "Readonly health-food meal records and fingerprint.",
        "inputSchema": tool_schema(["uid"]),
    },
    {
        "name": "health_food_recommendation_status",
        "description": "Readonly health-food daily recommendation status.",
        "inputSchema": tool_schema(["uid", "recommendation_date"]),
    },
    {
        "name": "health_food_search_logs",
        "description": "Readonly health-food log samples.",
        "inputSchema": tool_schema(["service_name"]),
    },
    {
        "name": "health_food_similar_cases",
        "description": "Readonly health-food similar case search.",
        "inputSchema": tool_schema([]),
    },
]


def call_tool(name: str, args: dict[str, Any]) -> dict[str, Any]:
    missing = SCENARIO == "recommendation_missing"
    quota_bad = SCENARIO == "quota_exhausted"
    base = {"uid": uid(args), "scenario": SCENARIO, "data_updated_at": now_iso()}
    if name == "health_food_user_profile":
        return {
            **base,
            "registered": True,
            "membership_level": 1,
            "health_goal_summary": "fat loss, target 1800 kcal/day",
            "latest_device": {"platform": "ios", "app_version": "1.2.3"},
            "updated_at": now_iso(),
            "source": "health-food",
            "version": "mcp-mock",
        }
    if name == "health_food_ai_quota":
        return {
            **base,
            "membership_level": 1,
            "available_tokens": "0" if quota_bad else "1200",
            "daily_chat_count": 30 if quota_bad else 3,
            "limit_chat": 30,
            "last_reset_date": now_iso(),
            "abnormal": quota_bad,
            "reason": "daily chat count reached limit while user still has membership" if quota_bad else "quota is normal",
        }
    if name == "health_food_meal_records":
        return {
            **base,
            "meal_count": 2 if missing else 4,
            "missing_meal_ids": ["meal_missing_dinner"] if missing else [],
            "meal_data_fingerprint": "fingerprint_stale" if missing else "fingerprint_current",
            "meals": [
                {"meal_id": "meal_breakfast", "meal_name": "breakfast", "meal_time": now_iso()},
                {"meal_id": "meal_lunch", "meal_name": "lunch", "meal_time": now_iso()},
            ],
        }
    if name == "health_food_recommendation_status":
        return {
            **base,
            "recommend_date": recommendation_date(args),
            "has_recommendation": not missing,
            "job_status": "failed" if missing else "success",
            "meal_count": 2 if missing else 4,
            "meal_data_fingerprint": "fingerprint_stale" if missing else "fingerprint_current",
            "generated_at": None if missing else now_iso(),
            "failure_reason": "meal_data_fingerprint did not refresh after dinner upload" if missing else "",
            "source_meal_ids": ["meal_breakfast", "meal_lunch"],
        }
    if name == "health_food_search_logs":
        message = (
            "TokenAccountService rejected request: daily chat count limit reached"
            if quota_bad
            else "RecommendFoodJob skipped generation: meal_data_fingerprint unchanged"
        )
        return {
            "service_name": str(args.get("service_name") or "health-food"),
            "total": 1,
            "samples": [{"time": now_iso(), "level": "error", "service": "health-food", "message": message}],
        }
    if name == "health_food_similar_cases":
        return {
            "items": [
                {
                    "case_no": "case_hf_mcp_001",
                    "issue_domain": "health_food",
                    "issue_type": str(args.get("issue_type") or "recommendation_missing"),
                    "summary": "similar health-food MCP case",
                    "score": 0.9,
                }
            ]
        }
    raise ValueError(f"unknown tool {name}")


def response(request_id: Any, result: dict[str, Any] | None = None, error: dict[str, Any] | None = None) -> None:
    payload: dict[str, Any] = {"jsonrpc": "2.0", "id": request_id}
    if error is not None:
        payload["error"] = error
    else:
        payload["result"] = result or {}
    sys.stdout.write(json.dumps(payload, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def handle(message: dict[str, Any]) -> None:
    method = message.get("method")
    request_id = message.get("id")
    if request_id is None:
        return
    try:
        if method == "initialize":
            response(
                request_id,
                {
                    "protocolVersion": PROTOCOL_VERSION,
                    "capabilities": {"tools": {"listChanged": False}},
                    "serverInfo": {"name": "mock-health-food-mcp-server", "version": "0.1.0"},
                },
            )
            return
        if method == "tools/list":
            response(request_id, {"tools": TOOLS})
            return
        if method == "tools/call":
            params = message.get("params") or {}
            name = str(params.get("name") or "")
            args = params.get("arguments") or {}
            if not isinstance(args, dict):
                args = {}
            data = call_tool(name, args)
            response(
                request_id,
                {
                    "content": [{"type": "text", "text": json.dumps(data, ensure_ascii=False)}],
                    "structuredContent": data,
                    "isError": False,
                },
            )
            return
        response(request_id, error={"code": -32601, "message": f"method not found: {method}"})
    except Exception as exc:  # noqa: BLE001 - mock server should report protocol errors.
        response(request_id, error={"code": -32000, "message": str(exc)})


def main() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            message = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(message, dict):
            handle(message)


if __name__ == "__main__":
    main()
