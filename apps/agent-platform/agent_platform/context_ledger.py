from __future__ import annotations

from typing import Any


MAX_SUMMARY_CHARS = 600


def compact_tool_observation(tool_name: str, status: str, response: dict[str, Any]) -> dict[str, Any]:
    """Return the small evidence envelope that is allowed into LLM context."""
    evidence_refs = evidence_refs_from_tool_response(tool_name, response)
    data = response.get("data")
    observation: dict[str, Any] = {
        "tool_name": tool_name,
        "status": status or response.get("status") or "success",
        "summary": _clip(str(response.get("summary") or response.get("error") or "")),
        "evidence_refs": evidence_refs,
        "result_count": _result_count(response),
        "data_shape": _data_shape(data),
    }
    if response.get("error"):
        observation["error"] = _clip(str(response.get("error")))
    return observation


def compact_failed_observation(tool_name: str, error: str) -> dict[str, Any]:
    return {
        "tool_name": tool_name,
        "status": "failed",
        "summary": _clip(error),
        "evidence_refs": [],
        "result_count": 0,
        "data_shape": {"type": "error"},
        "error": _clip(error),
    }


def compact_decision_payload(decision: Any) -> dict[str, Any]:
    return {
        "action": getattr(decision, "action", ""),
        "reason": _clip(str(getattr(decision, "reason", ""))),
        "confidence": getattr(decision, "confidence", 0.0),
        "missing_fields": list(getattr(decision, "missing_fields", []) or []),
        "tool_plan": [
            {
                "tool_name": item.tool_name,
                "reason": _clip(item.reason),
                "argument_keys": sorted(str(key) for key in item.arguments.keys()),
            }
            for item in getattr(decision, "tool_plan", []) or []
        ],
        "agent_reports": [
            {
                "agent_name": report.agent_name,
                "action": report.action,
                "reason": _clip(report.reason),
                "observations": [_clip(str(item)) for item in report.observations[:12]],
                "risks": list(report.risks[:8]),
                "evidence_count": len(report.evidence),
            }
            for report in getattr(decision, "agent_reports", []) or []
        ],
        "verification": _compact_verification(getattr(decision, "verification", None)),
    }


def compact_knowledge_payload(rows: list[dict[str, Any]]) -> dict[str, Any]:
    return {
        "matched_count": len(rows),
        "candidates": [
            {
                "id": item.get("id"),
                "title": _clip(str(item.get("title") or ""), 160),
                "confidence": item.get("confidence"),
                "observed_case_count": item.get("observed_case_count"),
                "status": item.get("status"),
            }
            for item in rows[:10]
        ],
    }


def compact_gateway_tools_payload(tools: list[dict[str, Any]]) -> dict[str, Any]:
    services: dict[str, int] = {}
    for item in tools:
        service_name = str(item.get("service_name") or item.get("service") or "unassigned")
        services[service_name] = services.get(service_name, 0) + 1
    return {
        "tool_count": len(tools),
        "services": services,
        "tools": [
            {
                "name": item.get("name") or item.get("tool_name"),
                "service_name": item.get("service_name") or item.get("service") or "",
                "required_scope": item.get("required_scope") or "",
            }
            for item in tools[:50]
        ],
    }


def final_answer_verification(observations: list[dict[str, Any]], confidence: float) -> dict[str, Any]:
    evidence_refs = evidence_refs_from_observations(observations)
    successful = [item for item in observations if item.get("status") == "success"]
    accepted = bool(successful and evidence_refs)
    adjusted_confidence = confidence if accepted else min(confidence, 0.45)
    reason = "最终结论有 Gateway 工具证据引用。" if accepted else "最终结论缺少可回查证据引用，只能作为低置信转人工结论。"
    return {
        "accepted": accepted,
        "reason": reason,
        "evidence_ref_count": len(evidence_refs),
        "successful_observation_count": len(successful),
        "adjusted_confidence": adjusted_confidence,
        "evidence_refs": evidence_refs,
    }


def evidence_refs_from_observations(observations: list[dict[str, Any]]) -> list[dict[str, Any]]:
    refs: list[dict[str, Any]] = []
    seen: set[tuple[str, str, str]] = set()
    for item in observations:
        for ref in item.get("evidence_refs") or []:
            if not isinstance(ref, dict):
                continue
            key = (
                str(ref.get("ref_type") or ""),
                str(ref.get("ref_id") or ""),
                str(ref.get("tool_name") or item.get("tool_name") or ""),
            )
            if key in seen or not key[1]:
                continue
            seen.add(key)
            refs.append(dict(ref))
    return refs


def evidence_refs_from_tool_response(tool_name: str, response: dict[str, Any]) -> list[dict[str, Any]]:
    refs: list[dict[str, Any]] = []
    tool_call_id = response.get("tool_call_id")
    if tool_call_id:
        refs.append({"ref_type": "gateway_tool_call", "ref_id": str(tool_call_id), "tool_name": tool_name})
    query_id = response.get("query_id")
    if not query_id and isinstance(response.get("data"), dict):
        query_id = response["data"].get("query_id")
    if query_id:
        refs.append({"ref_type": "downstream_query", "ref_id": str(query_id), "tool_name": tool_name})
    return refs


def _compact_verification(value: Any) -> dict[str, Any]:
    if value is None:
        return {}
    return {
        "accepted": getattr(value, "accepted", False),
        "reason": _clip(str(getattr(value, "reason", ""))),
        "checks": list(getattr(value, "checks", []) or []),
        "violations": list(getattr(value, "violations", []) or []),
        "tool_budget": getattr(value, "tool_budget", 0),
        "tool_count": getattr(value, "tool_count", 0),
    }


def _result_count(response: dict[str, Any]) -> int:
    value = response.get("result_count")
    if value is None and isinstance(response.get("data"), dict):
        value = response["data"].get("result_count")
    if isinstance(value, int):
        return max(value, 0)
    data = response.get("data")
    if isinstance(data, list):
        return len(data)
    if isinstance(data, dict):
        for key in ("items", "rows", "records", "events"):
            nested = data.get(key)
            if isinstance(nested, list):
                return len(nested)
    return 0


def _data_shape(value: Any) -> dict[str, Any]:
    if isinstance(value, dict):
        return {"type": "object", "keys": sorted(str(key) for key in value.keys())[:30]}
    if isinstance(value, list):
        first = value[0] if value else None
        shape = {"type": "array", "count": len(value)}
        if isinstance(first, dict):
            shape["item_keys"] = sorted(str(key) for key in first.keys())[:30]
        return shape
    if value is None:
        return {"type": "null"}
    return {"type": type(value).__name__}


def _clip(value: str, limit: int = MAX_SUMMARY_CHARS) -> str:
    value = " ".join(value.split())
    if len(value) <= limit:
        return value
    return value[: limit - 3] + "..."
