from __future__ import annotations

import json
import re
from typing import Any
from urllib.parse import urlparse

import yaml

from .repository import Repository


DANGER_WORDS = {
    "delete",
    "remove",
    "drop",
    "truncate",
    "insert",
    "update",
    "upsert",
    "create",
    "write",
    "execute",
    "exec",
    "run",
    "shell",
    "command",
    "deploy",
    "restart",
    "approve",
    "refund",
    "pay",
    "transfer",
    "send",
    "grant",
    "revoke",
    "disable",
    "enable",
}
READ_WORDS = {"get", "list", "search", "query", "read", "find", "describe", "profile", "status", "quota", "logs", "snapshot", "events", "compare"}


def import_capabilities(repository: Repository, payload: dict[str, Any]) -> dict[str, Any]:
    raw_config = str(payload.get("raw_config") or "").strip()
    if not raw_config:
        raise ValueError("raw_config is required")
    data = _parse_config(raw_config)
    if "capabilities" not in data:
        raise ValueError("only HTTP readonly capability manifest is supported by Python Agent Platform importer")
    service_block = data.get("service") if isinstance(data.get("service"), dict) else {}
    service_name = str(payload.get("service_name") or service_block.get("service_name") or data.get("service_name") or "").strip()
    if not service_name:
        raise ValueError("service_name is required")
    base_url = str(payload.get("base_url") or service_block.get("base_url") or data.get("base_url") or "").strip()
    if base_url:
        _validate_base_url(base_url)
    secret_ref = str(payload.get("secret_ref") or _nested(service_block, "auth", "secret_ref") or _nested(service_block, "auth", "token_env") or "").strip()
    service = repository.upsert_business_service(
        {
            "service_name": service_name,
            "owner_team": service_block.get("owner_team") or "",
            "environment": service_block.get("environment") or "local",
            "base_url": base_url,
            "health_check_path": _nested(service_block, "health_check", "path") or "",
            "auth_type": _nested(service_block, "auth", "type") or "bearer",
            "secret_ref": secret_ref,
            "service_status": "enabled",
        }
    )
    imported = []
    warnings = []
    for raw_item in data.get("capabilities") or []:
        if not isinstance(raw_item, dict):
            continue
        tool_name = _normalize_tool_name(str(raw_item.get("tool_name") or ""))
        if not tool_name:
            warnings.append("skip capability without tool_name")
            continue
        method = str(raw_item.get("method") or "POST").upper()
        path = _normalize_path(str(raw_item.get("path") or ""))
        safety_status, reasons = assess_safety(tool_name, str(raw_item.get("description") or ""), method, path, str(raw_item.get("scope") or ""))
        if "/readonly/" not in path:
            safety_status = "rejected"
            reasons.append("readonly http path must be under /readonly/")
        tool_status = "draft"
        approval_status = "pending"
        if safety_status == "rejected":
            tool_status = "rejected"
            approval_status = "rejected"
        required = [str(v) for v in raw_item.get("required_params") or [] if str(v).strip()]
        optional = [str(v) for v in raw_item.get("optional_params") or [] if str(v).strip()]
        imported.append(
            repository.upsert_tool_capability(
                {
                    "tool_name": tool_name,
                    "description": str(raw_item.get("description") or tool_name),
                    "service_name": service_name,
                    "source_type": "http_adapter",
                    "readonly_base_url": base_url,
                    "readonly_path": path,
                    "http_method": method,
                    "secret_ref": secret_ref,
                    "required_scope": str(raw_item.get("scope") or _scope_from_tool_name(tool_name)),
                    "required_params": required,
                    "optional_params": optional,
                    "max_time_range_minutes": int(raw_item.get("max_time_range_minutes") or 0),
                    "max_limit": int(raw_item.get("max_limit") or 0),
                    "timeout_ms": int(raw_item.get("timeout_ms") or 5000),
                    "safety_status": safety_status,
                    "safety_reasons": reasons,
                    "approval_status": approval_status,
                    "tool_status": tool_status,
                    "created_by": payload.get("created_by") or "web",
                }
            )
        )
    return {"services": [service], "capabilities": imported, "warnings": warnings, "mcp_servers": [], "validation_runs": []}


def assess_safety(tool_name: str, description: str, method: str, path: str, scope: str) -> tuple[str, list[str]]:
    if method not in {"GET", "POST"}:
        return "rejected", [f"method {method} is not allowed for readonly capability"]
    signal = " ".join([tool_name, description, path, scope]).lower()
    for word in DANGER_WORDS:
        if re.search(rf"(^|[^a-z0-9]){re.escape(word)}([^a-z0-9]|$)", signal):
            return "rejected", [f"dangerous action keyword: {word}"]
    for word in READ_WORDS:
        if word in signal:
            return "readonly_candidate", [f"readonly signal: {word}"]
    return "needs_review", ["no clear readonly signal; manual review required"]


def _parse_config(raw: str) -> dict[str, Any]:
    try:
        value = json.loads(raw)
    except json.JSONDecodeError:
        value = yaml.safe_load(raw)
    if not isinstance(value, dict):
        raise ValueError("raw_config must be a JSON/YAML object")
    return value


def _validate_base_url(value: str) -> None:
    parsed = urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise ValueError("base_url must be an http(s) URL")


def _normalize_path(value: str) -> str:
    value = "/" + value.strip().lstrip("/")
    return re.sub(r"/+", "/", value)


def _normalize_tool_name(value: str) -> str:
    value = value.strip().lower().replace("-", "_").replace("/", "_")
    value = re.sub(r"[^a-z0-9_]+", "_", value)
    return value.strip("_")


def _scope_from_tool_name(name: str) -> str:
    parts = [part for part in name.split("_") if part]
    service = parts[1] if len(parts) > 1 else "dynamic"
    return f"{service}:read"


def _nested(value: dict[str, Any], *keys: str) -> Any:
    current: Any = value
    for key in keys:
        if not isinstance(current, dict):
            return ""
        current = current.get(key)
    return current or ""
