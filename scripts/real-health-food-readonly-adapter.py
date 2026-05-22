#!/usr/bin/env python3
"""Readonly adapter backed by a real local health-food service and test DB.

This adapter is for local integration verification. It does not synthesize
fault scenarios. Every health-food response is derived from the configured
local MySQL database, the health-food health endpoint, or optional log files.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import subprocess
import time
import urllib.request
from datetime import datetime, time as dt_time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from zoneinfo import ZoneInfo


PORT = int(os.getenv("REAL_HEALTH_FOOD_ADAPTER_PORT", "19084"))
API_KEY = os.getenv("CONNECTOR_API_KEY", "")
HEALTH_FOOD_BASE_URL = os.getenv("HEALTH_FOOD_BASE_URL", "http://127.0.0.1:18080").rstrip("/")
MYSQL_HOST = os.getenv("HEALTH_FOOD_MYSQL_HOST", "127.0.0.1")
MYSQL_PORT = os.getenv("HEALTH_FOOD_MYSQL_PORT", "3306")
MYSQL_USER = os.getenv("HEALTH_FOOD_MYSQL_USER", "root")
MYSQL_PASSWORD = os.getenv("HEALTH_FOOD_MYSQL_PASSWORD", "")
MYSQL_DATABASE = os.getenv("HEALTH_FOOD_MYSQL_DATABASE", "hf_troubleshoot_codex")
HEALTH_FOOD_LOG_PATH = os.getenv("HEALTH_FOOD_LOG_PATH", "")
LOCAL_TZ = ZoneInfo(os.getenv("HEALTH_FOOD_TIMEZONE", "Asia/Shanghai"))


def now_iso() -> str:
    return datetime.now(LOCAL_TZ).isoformat(timespec="seconds")


def parse_dt(value: str | None) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(LOCAL_TZ)
    except ValueError:
        return None


def day_bounds(params: dict) -> tuple[int, int, str]:
    at = parse_dt(params.get("at_time")) or datetime.now(LOCAL_TZ)
    date_text = str(params.get("recommendation_date") or at.date().isoformat())[:10]
    try:
        day = datetime.fromisoformat(date_text).date()
    except ValueError:
        day = at.date()
        date_text = day.isoformat()
    start_dt = datetime.combine(day, dt_time.min, tzinfo=LOCAL_TZ)
    end_dt = datetime.combine(day, dt_time.max, tzinfo=LOCAL_TZ)
    start = parse_dt(params.get("start_time"))
    end = parse_dt(params.get("end_time"))
    if start:
        start_dt = start
    if end:
        end_dt = end
    return int(start_dt.timestamp() * 1000), int(end_dt.timestamp() * 1000), date_text


def validate_uid(params: dict) -> str:
    uid = str(params.get("uid") or params.get("user_id") or "").strip()
    if not re.fullmatch(r"\d{1,32}", uid):
        raise ValueError("health-food uid must be a numeric local test uid")
    return uid


def mysql_query(sql: str) -> list[dict[str, str | None]]:
    env = os.environ.copy()
    if MYSQL_PASSWORD:
        env["MYSQL_PWD"] = MYSQL_PASSWORD
    cmd = [
        "mysql",
        "-h",
        MYSQL_HOST,
        "-P",
        MYSQL_PORT,
        "-u",
        MYSQL_USER,
        "--default-character-set=utf8mb4",
        "--batch",
        "--raw",
        MYSQL_DATABASE,
        "-e",
        sql,
    ]
    proc = subprocess.run(cmd, env=env, text=True, capture_output=True, check=False)
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or "mysql query failed")
    lines = proc.stdout.splitlines()
    if not lines:
        return []
    headers = lines[0].split("\t")
    rows: list[dict[str, str | None]] = []
    for line in lines[1:]:
        values = line.split("\t")
        row: dict[str, str | None] = {}
        for idx, header in enumerate(headers):
            value = values[idx] if idx < len(values) else None
            row[header] = None if value == "NULL" else value
        rows.append(row)
    return rows


def read_health_food_alive() -> dict:
    try:
        with urllib.request.urlopen(f"{HEALTH_FOOD_BASE_URL}/food-health/sys/alive", timeout=1.5) as resp:
            body = resp.read().decode("utf-8", errors="replace")
            return {"reachable": True, "status": resp.status, "body": body[:80]}
    except Exception as exc:  # noqa: BLE001 - adapter reports local dependency state.
        return {"reachable": False, "error": str(exc)}


def envelope(request_id: str, data: dict, warnings: list[str] | None = None) -> dict:
    return {
        "request_id": request_id,
        "source": "health-food-readonly-adapter/real-local",
        "queried_at": now_iso(),
        "data_updated_at": data.get("data_updated_at") or data.get("updated_at") or now_iso(),
        "version": "real-local-v1",
        "data": data,
        "warnings": warnings or [],
    }


def health_goal_summary(row: dict[str, str | None] | None) -> str:
    if not row:
        return ""
    parts = []
    for key, label in (("goal", "goal"), ("dietary_preferences", "dietary"), ("remark", "remark")):
        value = row.get(key)
        if value:
            parts.append(f"{label}={value}")
    return "; ".join(parts)


def query_meals(uid: str, params: dict) -> tuple[list[dict], str, str]:
    start_ms, end_ms, date_text = day_bounds(params)
    rows = mysql_query(
        "SELECT meal_id, meal_name, meal_time_ts, mood, status, "
        "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
        "FROM tb_meal_record "
        f"WHERE uid={uid} AND status=1 AND meal_time_ts BETWEEN {start_ms} AND {end_ms} "
        "ORDER BY meal_time_ts ASC"
    )
    meal_ids = [str(row["meal_id"]) for row in rows if row.get("meal_id")]
    fingerprint = hashlib.md5(",".join(sorted(meal_ids)).encode("utf-8")).hexdigest() if meal_ids else ""
    meals = [
        {
            "meal_id": row.get("meal_id"),
            "meal_name": row.get("meal_name"),
            "meal_time_ts": int(row.get("meal_time_ts") or 0),
            "mood": row.get("mood") or "",
            "status": int(row.get("status") or 0),
        }
        for row in rows
    ]
    return meals, fingerprint, date_text


def handle_health_food(path: str, payload: dict) -> tuple[int, dict]:
    params = payload.get("params") or {}
    request_id = payload.get("request_id") or f"req_real_hf_{int(time.time() * 1000)}"
    uid = validate_uid(params)
    alive = read_health_food_alive()

    if path == "/v1/readonly/health-food/user/profile":
        users = mysql_query(
            "SELECT uid, nick_name, email, phone, status, "
            "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
            f"FROM tb_user_info WHERE uid={uid} LIMIT 1"
        )
        memberships = mysql_query(f"SELECT level FROM tb_user_ai_membership WHERE uid={uid} LIMIT 1")
        goals = mysql_query(f"SELECT goal, dietary_preferences, remark FROM tb_health_goal WHERE uid={uid} LIMIT 1")
        devices = mysql_query(
            "SELECT device_id, device_name, os_type, os_version, area, "
            "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
            f"FROM tb_user_device_info WHERE uid={uid} ORDER BY update_time DESC LIMIT 1"
        )
        user = users[0] if users else {}
        data = {
            "uid": uid,
            "registered": bool(users),
            "membership_level": int((memberships[0].get("level") if memberships else 0) or 0),
            "health_goal_summary": health_goal_summary(goals[0] if goals else None),
            "latest_device": devices[0] if devices else {},
            "updated_at": user.get("updated_at") or now_iso(),
            "source": "health-food-db",
            "version": "real-local-db",
            "health_food_alive": alive,
        }
        return 200, envelope(request_id, data)

    if path == "/v1/readonly/health-food/ai/quota":
        accounts = mysql_query(
            "SELECT available_balance, daily_chat_count, "
            "DATE_FORMAT(last_reset_date, '%Y-%m-%dT%H:%i:%s+08:00') AS last_reset_date, "
            "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
            f"FROM tb_user_asset_account WHERE uid={uid} AND (account_type IN (2, 200) OR LOWER(asset_name)='token') "
            "ORDER BY update_time DESC LIMIT 1"
        )
        memberships = mysql_query(f"SELECT level FROM tb_user_ai_membership WHERE uid={uid} LIMIT 1")
        level = int((memberships[0].get("level") if memberships else 0) or 0)
        quotas = mysql_query(f"SELECT limit_chat FROM tb_ai_membership_token_quotas WHERE level={level} LIMIT 1")
        account = accounts[0] if accounts else {}
        available = account.get("available_balance") or "0"
        daily = int(account.get("daily_chat_count") or 0)
        limit_chat = int((quotas[0].get("limit_chat") if quotas else daily) or 0)
        abnormal = (not accounts) or float(available) <= 0 or daily <= 0
        reason = "real token account is healthy"
        if not accounts:
            reason = "real DB has no token account row for uid"
        elif float(available) <= 0:
            reason = "real token account available_balance is zero"
        elif daily <= 0:
            reason = "real token account daily_chat_count is zero"
        data = {
            "uid": uid,
            "membership_level": level,
            "available_tokens": available,
            "daily_chat_count": daily,
            "limit_chat": limit_chat,
            "last_reset_date": account.get("last_reset_date") or now_iso(),
            "abnormal": abnormal,
            "reason": reason,
            "data_updated_at": account.get("updated_at") or now_iso(),
        }
        return 200, envelope(request_id, data)

    if path == "/v1/readonly/health-food/meals/range":
        meals, fingerprint, _ = query_meals(uid, params)
        data = {
            "uid": uid,
            "meal_count": len(meals),
            "missing_meal_ids": [],
            "meal_data_fingerprint": fingerprint,
            "meals": meals,
            "data_updated_at": now_iso(),
        }
        return 200, envelope(request_id, data)

    if path == "/v1/readonly/health-food/recommendation/status":
        meals, current_fingerprint, date_text = query_meals(uid, params)
        rows = mysql_query(
            "SELECT food_json, source_meal_ids, meal_data_fingerprint, meal_count, "
            "DATE_FORMAT(create_time, '%Y-%m-%dT%H:%i:%s+08:00') AS generated_at, "
            "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
            f"FROM tb_daily_food_recommend WHERE uid={uid} AND recommend_date='{date_text}' LIMIT 1"
        )
        row = rows[0] if rows else {}
        has_recommendation = bool(rows and (row.get("food_json") or ""))
        source_meal_ids = []
        if row.get("source_meal_ids"):
            try:
                source_meal_ids = json.loads(str(row["source_meal_ids"]))
            except json.JSONDecodeError:
                source_meal_ids = [str(row["source_meal_ids"])]
        if not source_meal_ids:
            source_meal_ids = [str(item["meal_id"]) for item in meals if item.get("meal_id")]
        stored_fingerprint = row.get("meal_data_fingerprint") or ""
        job_status = "success" if has_recommendation else "missing"
        failure_reason = ""
        if not has_recommendation:
            if meals:
                failure_reason = (
                    f"real DB has {len(meals)} meal record(s) for {date_text} but no "
                    "tb_daily_food_recommend row; inspect RecommendFoodJob/FoodServiceImpl"
                )
            else:
                failure_reason = f"real DB has no meal records for {date_text}"
        elif stored_fingerprint and current_fingerprint and stored_fingerprint != current_fingerprint:
            job_status = "stale"
            failure_reason = "stored recommendation fingerprint differs from current meal fingerprint"
        data = {
            "uid": uid,
            "recommend_date": date_text,
            "has_recommendation": has_recommendation,
            "job_status": job_status,
            "meal_count": int(row.get("meal_count") or len(meals)),
            "meal_data_fingerprint": stored_fingerprint or current_fingerprint,
            "generated_at": row.get("generated_at"),
            "failure_reason": failure_reason,
            "source_meal_ids": source_meal_ids,
            "service_name": "health-food",
            "suspect_area": "RecommendFoodJob FoodServiceImpl meal_data_fingerprint",
            "data_updated_at": row.get("updated_at") or now_iso(),
        }
        return 200, envelope(request_id, data)

    return 404, {"code": "NOT_FOUND", "error": f"unknown path {path}"}


def read_log_samples(keyword: str, limit: int) -> list[dict]:
    if not HEALTH_FOOD_LOG_PATH:
        return []
    root = Path(HEALTH_FOOD_LOG_PATH)
    paths: list[Path]
    if root.is_dir():
        paths = sorted(root.rglob("*.log"), key=lambda item: item.stat().st_mtime, reverse=True)
    else:
        paths = [root]
    samples: list[dict] = []
    lowered = keyword.lower()
    for path in paths[:8]:
        try:
            lines = path.read_text(encoding="utf-8", errors="ignore").splitlines()
        except OSError:
            continue
        for line in reversed(lines[-800:]):
            if lowered and lowered not in line.lower():
                continue
            samples.append({"time": now_iso(), "level": "info", "service": "health-food", "message": line[:500]})
            if len(samples) >= limit:
                return samples
    return samples


def handle_ops(path: str, payload: dict) -> tuple[int, dict]:
    params = payload.get("params") or {}
    request_id = payload.get("request_id") or f"req_real_ops_{int(time.time() * 1000)}"
    if path == "/v1/readonly/ops/logs/search":
        service_name = str(params.get("service_name") or "health-food")
        keyword = str(params.get("keyword") or params.get("trace_id") or "health-food")
        limit = int(params.get("limit") or 10)
        samples = read_log_samples(keyword, max(1, min(limit, 20)))
        data = {
            "service_name": service_name,
            "total": len(samples),
            "samples": samples,
        }
        warnings = [] if samples else ["no local health-food log file configured or no matching log line"]
        return 200, envelope(request_id, data, warnings)
    if path == "/v1/readonly/ops/cases/similar":
        data = {"items": []}
        return 200, envelope(request_id, data)
    if path == "/v1/readonly/ops/deployments/recent":
        data = {"service_name": params.get("service_name") or "health-food", "items": []}
        return 200, envelope(request_id, data)
    return 404, {"code": "NOT_FOUND", "error": f"unknown path {path}"}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/healthz":
            self.write_json(200, {"ok": True, "source": "real-local", "health_food": read_health_food_alive()})
            return
        self.write_json(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        if API_KEY and self.headers.get("Authorization") != f"Bearer {API_KEY}":
            self.write_json(401, {"code": "UNAUTHORIZED", "error": "invalid adapter token"})
            return
        length = int(self.headers.get("Content-Length") or "0")
        raw = self.rfile.read(length)
        try:
            payload = json.loads(raw or b"{}")
            if self.path.startswith("/v1/readonly/health-food/"):
                status, body = handle_health_food(self.path, payload)
            elif self.path.startswith("/v1/readonly/ops/"):
                status, body = handle_ops(self.path, payload)
            else:
                status, body = 404, {"code": "NOT_FOUND", "error": f"unknown path {self.path}"}
        except ValueError as exc:
            status, body = 400, {"code": "BAD_REQUEST", "error": str(exc)}
        except Exception as exc:  # noqa: BLE001 - adapter should surface readonly failures.
            status, body = 500, {"code": "ADAPTER_ERROR", "error": str(exc)}
        self.write_json(status, body)

    def write_json(self, status: int, body: dict) -> None:
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, fmt: str, *args: object) -> None:
        print(f"{self.log_date_time_string()} {fmt % args}")


def main() -> None:
    server = ThreadingHTTPServer(("127.0.0.1", PORT), Handler)
    print(f"real health-food readonly adapter listening on http://127.0.0.1:{PORT}")
    server.serve_forever()


if __name__ == "__main__":
    main()
