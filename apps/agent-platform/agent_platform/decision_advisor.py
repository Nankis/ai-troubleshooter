from __future__ import annotations

from dataclasses import asdict
from typing import Any

from decision_engine.models import AgentReport, DecisionRequest, DecisionResponse, ToolPlan

from .llm import LLMClient


class LLMDecisionAdvisor:
    name = "llm_decision_agent"

    def __init__(self, llm: LLMClient) -> None:
        self.llm = llm

    def evaluate(
        self,
        request: DecisionRequest,
        agent_reports: list[AgentReport] | tuple[AgentReport, ...],
        default_proposal: DecisionResponse,
    ) -> AgentReport | None:
        if not self.llm.is_real:
            return None
        payload = self.llm.advise_decision(_request_payload(request), default_proposal.to_dict()).payload
        action = _safe_action(payload.get("action"), default_proposal.action)
        if action == "skip":
            return AgentReport(
                agent_name=self.name,
                action="skip",
                reason=str(payload.get("reason") or "LLM advisor skipped; deterministic proposal kept."),
                confidence=float(payload.get("confidence") or 0.0),
            )
        selected_tools = _selected_tools(payload.get("selected_tools") or payload.get("tools"), request)
        tool_plan = [_tool_plan(name, request, str(payload.get("reason") or default_proposal.reason)) for name in selected_tools]
        if action == "invoke_tools" and not tool_plan:
            action = default_proposal.action
            tool_plan = list(default_proposal.tool_plan)
        return AgentReport(
            agent_name=self.name,
            action=action,
            reason=str(payload.get("reason") or default_proposal.reason)[:1000],
            missing_fields=[str(item) for item in (payload.get("missing_fields") or default_proposal.missing_fields)][:5],
            tool_plan=tool_plan,
            knowledge_source=str(payload.get("knowledge_source") or default_proposal.knowledge_source or ""),
            confidence=_confidence(payload.get("confidence"), default_proposal.confidence),
            observations=[
                f"provider={self.llm.config.provider}",
                f"model={self.llm.config.model}",
                f"default_action={default_proposal.action}",
                f"selected_tool_count={len(tool_plan)}",
            ],
            risks=["LLM advisor output is advisory only; verifier enforces tool availability and budget"],
        )


def _request_payload(request: DecisionRequest) -> dict[str, Any]:
    payload = asdict(request)
    payload["available_tools"] = [
        {
            "name": tool.name,
            "required_scope": tool.required_scope,
            "max_time_range_minutes": tool.max_time_range_minutes,
            "max_limit": tool.max_limit,
        }
        for tool in request.available_tools
    ]
    return payload


def _safe_action(value: Any, default: str) -> str:
    action = str(value or default or "need_human").strip()
    allowed = {"ask_user", "answer_from_knowledge", "invoke_tools", "need_human", "local_code_inspection", "skip"}
    return action if action in allowed else default


def _selected_tools(value: Any, request: DecisionRequest) -> list[str]:
    raw = value if isinstance(value, list) else []
    available = {tool.name for tool in request.available_tools if tool.name}
    out: list[str] = []
    for item in raw:
        name = str(item).strip()
        if name and (not available or name in available) and name not in out:
            out.append(name)
    return out[: max(1, min(request.max_tool_calls, 10))]


def _tool_plan(tool_name: str, request: DecisionRequest, reason: str) -> ToolPlan:
    args = dict(request.entities)
    if request.case.case_no:
        args.setdefault("case_no", request.case.case_no)
    return ToolPlan(tool_name=tool_name, reason=reason or f"LLM advisor selected {tool_name}", arguments=args)


def _confidence(value: Any, default: float) -> float:
    try:
        confidence = float(value)
    except (TypeError, ValueError):
        confidence = float(default or 0.5)
    return max(0.0, min(confidence, 1.0))
