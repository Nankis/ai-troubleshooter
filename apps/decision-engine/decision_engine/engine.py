from __future__ import annotations

from .models import DecisionRequest, DecisionResponse, ToolPlan


REQUIRED_FIELDS_BY_DOMAIN: dict[str, tuple[str, ...]] = {
    "kline": ("symbol", "interval", "abnormal_time", "issue_type"),
    "asset": ("asset_symbol", "abnormal_time", "issue_type"),
}

DEFAULT_TOOLS_BY_DOMAIN: dict[str, tuple[str, ...]] = {
    "kline": (
        "get_internal_kline",
        "get_external_kline_compare",
        "get_kline_cache_status",
        "get_market_source_status",
        "get_similar_cases",
    ),
    "asset": (
        "get_asset_snapshot",
        "get_asset_events",
        "get_user_recent_errors",
        "get_similar_cases",
    ),
    "default": (
        "search_logs_by_service",
        "get_recent_deployments",
        "get_similar_cases",
    ),
}


class DecisionEngine:
    def plan(self, request: DecisionRequest) -> DecisionResponse:
        missing = self._missing_fields(request)
        if missing:
            return DecisionResponse(
                action="ask_user",
                reason="必要字段不足，先追问再查生产只读证据。",
                missing_fields=missing[:3],
                confidence=0.6,
            )

        knowledge = self._best_knowledge(request)
        if knowledge is not None:
            return DecisionResponse(
                action="answer_from_knowledge",
                reason="平台历史经验置信度高，且不需要实时生产状态验证；直接返回前必须写 AI 决策日志。",
                knowledge_source=knowledge.source or knowledge.title,
                confidence=knowledge.confidence,
            )

        selected_tools = self._select_tools(request)
        if not selected_tools:
            return DecisionResponse(
                action="need_human",
                reason="没有可用只读工具，不能绕过 Gateway 直接查询生产。",
                confidence=0.4,
            )

        return DecisionResponse(
            action="invoke_tools",
            reason="字段已满足最小排障条件，按有限工具计划查询只读证据。",
            tool_plan=[self._tool_plan(name, request) for name in selected_tools],
            confidence=0.72,
        )

    def _missing_fields(self, request: DecisionRequest) -> list[str]:
        domain = request.case.issue_domain or "default"
        required = REQUIRED_FIELDS_BY_DOMAIN.get(domain, ())
        missing = [field for field in required if not request.entities.get(field)]
        if domain == "asset" and not (request.entities.get("user_id") or request.entities.get("account_id")):
            missing.insert(0, "user_id_or_account_id")
        return missing

    def _select_tools(self, request: DecisionRequest) -> list[str]:
        domain = request.case.issue_domain or "default"
        preferred = DEFAULT_TOOLS_BY_DOMAIN.get(domain, DEFAULT_TOOLS_BY_DOMAIN["default"])
        available = {tool.name for tool in request.available_tools if tool.name}
        if available:
            preferred = tuple(name for name in preferred if name in available)
        budget = max(1, min(request.max_tool_calls, 10))
        return list(preferred[:budget])

    def _tool_plan(self, tool_name: str, request: DecisionRequest) -> ToolPlan:
        args = dict(request.entities)
        if request.case.case_no:
            args.setdefault("case_no", request.case.case_no)
        return ToolPlan(
            tool_name=tool_name,
            reason=f"查询 {tool_name} 获取只读证据，调用必须经过 Investigation Gateway。",
            arguments=args,
        )

    def _best_knowledge(self, request: DecisionRequest):
        candidates = sorted(
            request.knowledge_candidates,
            key=lambda item: (item.confidence, item.observed_case_count),
            reverse=True,
        )
        if not candidates:
            return None
        best = candidates[0]
        if best.confidence >= 0.88 and best.observed_case_count >= 2 and not best.requires_realtime_check:
            return best
        return None
