#!/usr/bin/env python3
"""Readonly adapter backed by a real local health-food service and test DB.

This adapter is for local integration verification and controlled production
readonly log access. It does not synthesize fault scenarios. Every health-food
response is derived from the configured local MySQL database, the health-food
health endpoint, the health-food admin log endpoint, or optional log files.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, time as dt_time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any
from zoneinfo import ZoneInfo


PORT = int(os.getenv("REAL_HEALTH_FOOD_ADAPTER_PORT", "19084"))
API_KEY = os.getenv("CONNECTOR_API_KEY", "")
HEALTH_FOOD_BASE_URL = os.getenv("HEALTH_FOOD_BASE_URL", "http://127.0.0.1:18080").rstrip("/")
MYSQL_HOST = os.getenv("HEALTH_FOOD_MYSQL_HOST", "127.0.0.1")
MYSQL_PORT = os.getenv("HEALTH_FOOD_MYSQL_PORT", "3306")
MYSQL_USER = os.getenv("HEALTH_FOOD_MYSQL_USER", "root")
MYSQL_PASSWORD = os.getenv("HEALTH_FOOD_MYSQL_PASSWORD", "")
MYSQL_DATABASE = os.getenv("HEALTH_FOOD_MYSQL_DATABASE", "").strip()
HEALTH_FOOD_LOG_PATH = os.getenv("HEALTH_FOOD_LOG_PATH", "")
HEALTH_FOOD_ADMIN_BASE_URL = os.getenv("HEALTH_FOOD_ADMIN_BASE_URL", "").rstrip("/")
HEALTH_FOOD_ADMIN_SECRET = os.getenv("HEALTH_FOOD_ADMIN_SECRET", "")
HEALTH_FOOD_ADMIN_SEARCH_LOGS_PATH = os.getenv(
    "HEALTH_FOOD_ADMIN_SEARCH_LOGS_PATH",
    "/food-health/sys/admin/search-logs",
)
HEALTH_FOOD_ALLOWED_SERVICE_NAMES = {
    item.strip()
    for item in os.getenv("HEALTH_FOOD_ALLOWED_SERVICE_NAMES", "health-food,food-health").split(",")
    if item.strip()
}
HEALTH_FOOD_LOG_MAX_LIMIT = int(os.getenv("HEALTH_FOOD_LOG_MAX_LIMIT", "20"))
HEALTH_FOOD_LOG_MAX_RANGE_MINUTES = int(os.getenv("HEALTH_FOOD_LOG_MAX_RANGE_MINUTES", "30"))
HEALTH_FOOD_LOG_TIMEOUT_SECONDS = float(os.getenv("HEALTH_FOOD_LOG_TIMEOUT_SECONDS", "3"))
LOCAL_TZ = ZoneInfo(os.getenv("HEALTH_FOOD_TIMEZONE", "Asia/Shanghai"))

LOG_LEVEL_RE = re.compile(r"\b(ERROR|WARN|INFO|DEBUG|TRACE)\b")
EMAIL_RE = re.compile(r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b")
PHONE_RE = re.compile(r"\b1[3-9]\d{9}\b")
BEARER_RE = re.compile(r"(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+")
SECRET_ASSIGN_RE = re.compile(
    r"(?i)\b(password|passwd|pwd|token|secret|authorization|api[_-]?key)\b(\s*[:=]\s*)([^\s,;}&\"]+)"
)


def now_iso() -> str:
    return datetime.now(LOCAL_TZ).isoformat(timespec="seconds")


def parse_dt(value: str | None) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(LOCAL_TZ)
    except ValueError:
        return None


def normalize_params(params: dict | None) -> dict:
    if not isinstance(params, dict):
        return {}
    out: dict = {}
    for key, value in params.items():
        raw_key = str(key)
        normalized = raw_key.replace("-", "_")
        normalized = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", normalized)
        normalized = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", normalized).lower()
        out.setdefault(normalized, value)
        out.setdefault(raw_key, value)
    return out


def int_param(params: dict, key: str, default: int, minimum: int, maximum: int) -> int:
    try:
        value = int(params.get(key) or default)
    except (TypeError, ValueError):
        value = default
    return max(minimum, min(value, maximum))


def effective_service_name(params: dict) -> str:
    service_name = str(params.get("service_name") or "health-food").strip()
    if HEALTH_FOOD_ALLOWED_SERVICE_NAMES and service_name not in HEALTH_FOOD_ALLOWED_SERVICE_NAMES:
        allowed = ", ".join(sorted(HEALTH_FOOD_ALLOWED_SERVICE_NAMES))
        raise ValueError(f"service_name {service_name!r} is not allowed; allowed={allowed}")
    return service_name


def log_time_window(params: dict) -> tuple[datetime, datetime]:
    end = parse_dt(str(params.get("end_time") or "")) or datetime.now(LOCAL_TZ)
    start = parse_dt(str(params.get("start_time") or "")) or (end - timedelta(minutes=30))
    if end < start:
        raise ValueError("end_time must be after start_time")
    max_range = timedelta(minutes=max(1, HEALTH_FOOD_LOG_MAX_RANGE_MINUTES))
    if end - start > max_range:
        raise ValueError(f"log time range exceeds max {HEALTH_FOOD_LOG_MAX_RANGE_MINUTES} minutes")
    return start, end


def date_strings(start: datetime, end: datetime) -> list[str]:
    dates: list[str] = []
    day = start.date()
    while day <= end.date():
        dates.append(day.isoformat())
        day = day + timedelta(days=1)
    return dates


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


def mysql_query(sql: str, params: tuple[Any, ...] | dict[str, Any] | None = None) -> list[dict[str, str | None]]:
    if not MYSQL_DATABASE:
        raise RuntimeError(
            "HEALTH_FOOD_MYSQL_DATABASE is required and must point to an existing readonly health-food schema; "
            "do not create ad-hoc troubleshooting schemas by default."
        )
    try:
        import pymysql
        import pymysql.cursors
    except ModuleNotFoundError as exc:
        raise RuntimeError(
            "PyMySQL is required for the real health-food readonly adapter; "
            "install with `python3.13 -m pip install PyMySQL`"
        ) from exc
    try:
        port = int(MYSQL_PORT)
    except ValueError as exc:
        raise RuntimeError("HEALTH_FOOD_MYSQL_PORT must be numeric") from exc

    connection = pymysql.connect(
        host=MYSQL_HOST,
        port=port,
        user=MYSQL_USER,
        password=MYSQL_PASSWORD,
        database=MYSQL_DATABASE,
        charset="utf8mb4",
        cursorclass=pymysql.cursors.DictCursor,
        autocommit=True,
        read_timeout=3,
        write_timeout=3,
    )
    try:
        with connection.cursor() as cursor:
            cursor.execute(sql, params or ())
            return [
                {key: None if value is None else str(value) for key, value in row.items()}
                for row in cursor.fetchall()
            ]
    finally:
        connection.close()


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


def mask_log_text(text: str, max_chars: int = 800) -> str:
    masked = text
    if HEALTH_FOOD_ADMIN_SECRET:
        masked = masked.replace(HEALTH_FOOD_ADMIN_SECRET, "<redacted>")
    masked = BEARER_RE.sub("Bearer <redacted>", masked)
    masked = SECRET_ASSIGN_RE.sub(r"\1\2<redacted>", masked)
    masked = EMAIL_RE.sub("<email_redacted>", masked)
    masked = PHONE_RE.sub("<phone_redacted>", masked)
    if len(masked) > max_chars:
        return masked[:max_chars] + "..."
    return masked


def parse_health_food_log_time(value: str | None) -> datetime | None:
    if not value:
        return None
    for fmt in ("%Y-%m-%d %H:%M:%S,%f", "%Y-%m-%d %H:%M:%S"):
        try:
            return datetime.strptime(value, fmt).replace(tzinfo=LOCAL_TZ)
        except ValueError:
            continue
    return parse_dt(value)


def detect_log_level(text: str, fallback: str) -> str:
    match = LOG_LEVEL_RE.search(text or "")
    if match:
        return match.group(1).lower()
    return fallback or "info"


def log_types_for_level(level: str) -> list[str]:
    normalized = level.strip().lower()
    if normalized == "error":
        return ["error"]
    if normalized in {"warn", "warning", "info", "debug", "trace"}:
        return ["info"]
    return ["error", "info"]


def sample_from_admin_line(line: dict, service_name: str, requested_level: str) -> dict | None:
    text = str(line.get("text") or "")
    summary = str(line.get("summary") or "")
    raw_time = str(line.get("time") or "")
    level = detect_log_level(text or summary, requested_level or "info")
    if requested_level:
        normalized = "warn" if requested_level.lower() == "warning" else requested_level.lower()
        if normalized in {"error", "warn", "info", "debug", "trace"} and level != normalized:
            return None
    parsed_time = parse_health_food_log_time(raw_time)
    time_value = parsed_time.isoformat(timespec="milliseconds") if parsed_time else raw_time
    message = summary or (text.splitlines()[0] if text else "")
    sample = {
        "time": time_value,
        "level": level,
        "service": service_name,
        "message": mask_log_text(message, 500),
        "excerpt": mask_log_text(text or summary, 800),
    }
    trace_match = re.search(r"\s-\s([A-Za-z0-9][A-Za-z0-9_-]{7,})\s-\s", text)
    if trace_match:
        sample["trace_id"] = trace_match.group(1)
    return sample


def query_health_food_admin_logs(params: dict, service_name: str, limit: int) -> tuple[list[dict], list[str]]:
    if not (HEALTH_FOOD_ADMIN_BASE_URL and HEALTH_FOOD_ADMIN_SECRET):
        return [], ["health-food admin log upstream is not configured"]

    start, end = log_time_window(params)
    keyword = str(params.get("keyword") or params.get("trace_id") or "").strip()
    requested_level = str(params.get("level") or "").strip().lower()
    warnings: list[str] = []
    samples: list[dict] = []
    seen_samples: set[tuple[str, str, str]] = set()
    path = "/" + HEALTH_FOOD_ADMIN_SEARCH_LOGS_PATH.lstrip("/")
    page_size = min(max(limit * 2, limit, 10), max(1, HEALTH_FOOD_LOG_MAX_LIMIT))

    for date_text in date_strings(start, end):
        for log_type in log_types_for_level(requested_level):
            query = {
                "password": HEALTH_FOOD_ADMIN_SECRET,
                "date": date_text,
                "type": log_type,
                "page": "1",
                "pageSize": str(page_size),
            }
            if params.get("trace_id"):
                query["traceId"] = str(params["trace_id"])
            if params.get("uid"):
                query["uid"] = str(params["uid"])
            elif params.get("user_id"):
                query["uid"] = str(params["user_id"])
            if params.get("api"):
                query["api"] = str(params["api"])
            if keyword:
                query["content"] = keyword

            url = HEALTH_FOOD_ADMIN_BASE_URL + path + "?" + urllib.parse.urlencode(query)
            req = urllib.request.Request(
                url,
                headers={
                    "Accept": "application/json",
                    "User-Agent": "ai-troubleshooter-health-food-readonly-adapter/1.0",
                },
            )
            try:
                with urllib.request.urlopen(req, timeout=HEALTH_FOOD_LOG_TIMEOUT_SECONDS) as resp:
                    body = json.loads(resp.read().decode("utf-8", errors="replace") or "{}")
            except urllib.error.HTTPError as exc:
                warnings.append(f"health-food admin log API returned HTTP {exc.code}")
                continue
            except urllib.error.URLError as exc:
                warnings.append(f"health-food admin log API unavailable: {exc.reason}")
                continue
            except json.JSONDecodeError:
                warnings.append("health-food admin log API returned invalid JSON")
                continue

            if int(body.get("code") or 0) != 0:
                warnings.append(mask_log_text(str(body.get("msg") or "health-food admin log API failed"), 200))
                continue
            data = body.get("data") or {}
            for line in data.get("lines") or []:
                if not isinstance(line, dict):
                    continue
                dedupe_key = (
                    str(line.get("time") or ""),
                    str(line.get("summary") or ""),
                    str(line.get("text") or ""),
                )
                if dedupe_key in seen_samples:
                    continue
                seen_samples.add(dedupe_key)
                line_time = parse_health_food_log_time(str(line.get("time") or ""))
                if line_time and (line_time < start or line_time > end):
                    continue
                sample = sample_from_admin_line(line, service_name, requested_level)
                if sample is None:
                    continue
                samples.append(sample)
                if len(samples) >= limit:
                    return samples, warnings
    return samples, warnings


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
        "WHERE uid=%s AND status=1 AND meal_time_ts BETWEEN %s AND %s "
        "ORDER BY meal_time_ts ASC",
        (uid, start_ms, end_ms),
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
    params = normalize_params(payload.get("params") or {})
    request_id = payload.get("request_id") or f"req_real_hf_{int(time.time() * 1000)}"
    uid = validate_uid(params)
    alive = read_health_food_alive()

    if path == "/v1/readonly/health-food/user/profile":
        users = mysql_query(
            "SELECT uid, nick_name, email, phone, status, "
            "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
            "FROM tb_user_info WHERE uid=%s LIMIT 1",
            (uid,),
        )
        memberships = mysql_query("SELECT level FROM tb_user_ai_membership WHERE uid=%s LIMIT 1", (uid,))
        goals = mysql_query("SELECT goal, dietary_preferences, remark FROM tb_health_goal WHERE uid=%s LIMIT 1", (uid,))
        devices = mysql_query(
            "SELECT device_id, device_name, os_type, os_version, area, "
            "DATE_FORMAT(update_time, '%Y-%m-%dT%H:%i:%s+08:00') AS updated_at "
            "FROM tb_user_device_info WHERE uid=%s ORDER BY update_time DESC LIMIT 1",
            (uid,),
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
            "FROM tb_user_asset_account WHERE uid=%s AND (account_type IN (2, 200) OR LOWER(asset_name)='token') "
            "ORDER BY update_time DESC LIMIT 1",
            (uid,),
        )
        memberships = mysql_query("SELECT level FROM tb_user_ai_membership WHERE uid=%s LIMIT 1", (uid,))
        level = int((memberships[0].get("level") if memberships else 0) or 0)
        quotas = mysql_query("SELECT limit_chat FROM tb_ai_membership_token_quotas WHERE level=%s LIMIT 1", (level,))
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
            "FROM tb_daily_food_recommend WHERE uid=%s AND recommend_date=%s LIMIT 1",
            (uid, date_text),
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


def read_log_samples(keyword: str, limit: int, service_name: str) -> list[dict]:
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
            samples.append(
                {
                    "time": now_iso(),
                    "level": "info",
                    "service": service_name,
                    "message": mask_log_text(line, 500),
                }
            )
            if len(samples) >= limit:
                return samples
    return samples


def handle_ops(path: str, payload: dict) -> tuple[int, dict]:
    params = normalize_params(payload.get("params") or {})
    request_id = payload.get("request_id") or f"req_real_ops_{int(time.time() * 1000)}"
    if path == "/v1/readonly/ops/logs/search":
        service_name = effective_service_name(params)
        keyword = str(params.get("keyword") or params.get("trace_id") or "")
        limit = int_param(params, "limit", 10, 1, HEALTH_FOOD_LOG_MAX_LIMIT)
        samples, warnings = query_health_food_admin_logs(params, service_name, limit)
        if not samples:
            local_keyword = keyword or service_name
            samples = read_log_samples(local_keyword, limit, service_name)
            if samples:
                warnings = [item for item in warnings if item != "health-food admin log upstream is not configured"]
        data = {
            "service_name": service_name,
            "total": len(samples),
            "samples": samples,
        }
        if not samples and not warnings:
            warnings = ["no health-food log upstream/local file configured or no matching log line"]
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
            self.write_json(
                200,
                {
                    "ok": True,
                    "source": "real-local",
                    "health_food": read_health_food_alive(),
                    "admin_log_upstream_configured": bool(
                        HEALTH_FOOD_ADMIN_BASE_URL and HEALTH_FOOD_ADMIN_SECRET
                    ),
                    "local_log_path_configured": bool(HEALTH_FOOD_LOG_PATH),
                },
            )
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
