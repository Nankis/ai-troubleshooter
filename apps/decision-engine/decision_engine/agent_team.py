from __future__ import annotations

from collections.abc import Sequence

from .models import (
    AgentReport,
    DecisionRequest,
    DecisionResponse,
    KnowledgeCandidate,
    ToolPlan,
    VerificationReport,
)
from .local_code import LocalCodeInspector


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


class KnowledgeAgent:
    name = "knowledge_agent"

    def evaluate(self, request: DecisionRequest) -> AgentReport:
        best = self.best_knowledge(request)
        if best is None:
            observation = "knowledge_candidates=0"
            if request.knowledge_candidates:
                observation = "knowledge_confidence_or_realtime_gate_not_met"
            return AgentReport(
                agent_name=self.name,
                action="skip",
                reason="没有满足直接复用条件的平台经验，继续走实时只读证据排查。",
                observations=[observation],
            )

        source = best.source or best.title
        return AgentReport(
            agent_name=self.name,
            action="answer_from_knowledge",
            reason="平台历史经验置信度高、样本数足够且不要求实时校验，可作为优先答案来源。",
            knowledge_source=source,
            confidence=best.confidence,
            observations=[f"observed_case_count={best.observed_case_count}", f"source={source}"],
        )

    def best_knowledge(self, request: DecisionRequest) -> KnowledgeCandidate | None:
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


class DomainAgent:
    def __init__(self, domain: str, name: str) -> None:
        self.domain = domain
        self.name = name

    def evaluate(self, request: DecisionRequest, supervisor: "SupervisorAgentTeam") -> AgentReport:
        missing = supervisor.missing_fields(request, self.domain)
        if missing:
            return AgentReport(
                agent_name=self.name,
                action="ask_user",
                reason=f"{self.domain} 排障必要字段不足，先追问再查询下游只读工具。",
                missing_fields=missing[:3],
                confidence=0.6,
                risks=["缺字段时不允许调用生产只读工具"],
            )

        selected = supervisor.select_tools(request, self.domain)
        if not selected:
            return AgentReport(
                agent_name=self.name,
                action="need_human",
                reason=f"{self.domain} 没有可用的 Gateway 只读工具，不能绕过 Gateway 直接查询。",
                confidence=0.4,
                risks=["no_available_gateway_tool"],
            )

        return AgentReport(
            agent_name=self.name,
            action="invoke_tools",
            reason=f"{self.domain} 字段已满足最小排障条件，生成有限只读工具计划。",
            tool_plan=[supervisor.build_tool_plan(name, request) for name in selected],
            confidence=0.72,
            observations=[f"tool_count={len(selected)}"],
        )


class FallbackAgent:
    name = "fallback_agent"

    def evaluate(self, request: DecisionRequest, supervisor: "SupervisorAgentTeam") -> AgentReport:
        selected = supervisor.select_tools(request, "default")
        if not selected:
            return AgentReport(
                agent_name=self.name,
                action="need_human",
                reason="当前问题域没有 specialist，且没有可用的通用只读工具。",
                confidence=0.4,
                risks=["unsupported_domain", "no_available_gateway_tool"],
            )

        return AgentReport(
            agent_name=self.name,
            action="invoke_tools",
            reason="当前问题域没有专属 specialist，先按通用日志、发布记录和相似 case 做有限排查。",
            tool_plan=[supervisor.build_tool_plan(name, request) for name in selected],
            confidence=0.62,
            observations=[
                f"issue_domain={request.case.issue_domain or 'default'}",
                f"tool_count={len(selected)}",
            ],
        )


class LocalCodeAgent:
    name = "local_code_agent"

    def __init__(self, inspector: LocalCodeInspector | None = None) -> None:
        self.inspector = inspector or LocalCodeInspector.from_env()

    def should_run(self, request: DecisionRequest) -> bool:
        if not self._truthy(request.entities.get("debug_local_code", "")):
            return False
        status = (
            request.entities.get("gateway_evidence_status")
            or request.entities.get("tool_evidence_status")
            or request.entities.get("evidence_status")
        )
        return str(status).lower() in {
            "insufficient",
            "inconclusive",
            "no_answer",
            "no_match",
            "need_code_inspection",
            "needs_code_inspection",
        }

    def evaluate(self, request: DecisionRequest) -> AgentReport:
        service_name = request.entities.get("service_name", "")
        repo_hint = request.entities.get("repo_hint", "")
        if not service_name and not repo_hint:
            return AgentReport(
                agent_name=self.name,
                action="need_human",
                reason="Gateway 证据不足，但没有提供 service_name 或 repo_hint，无法选择本地仓库。",
                confidence=0.35,
                risks=["local_code_service_hint_missing"],
            )

        query_text = self._query_text(request)
        result = self.inspector.inspect(service_name=service_name, repo_hint=repo_hint, query_text=query_text)
        observations = [
            f"repo_id={result.repo_id}",
            f"status={result.status}",
            f"scanned_files={result.scanned_files}",
            f"skipped_denied_files={result.skipped_denied_files}",
            f"symbol_count={result.symbol_count}",
            f"call_edge_count={result.call_edge_count}",
            f"resolved_call_edge_count={result.resolved_call_edge_count}",
            f"implement_relation_count={result.implement_relation_count}",
            f"analysis_modes={','.join(result.analysis_modes) or 'none'}",
            f"analysis_backends={','.join(result.analysis_backends) or 'none'}",
        ]
        if result.status == "matched":
            return AgentReport(
                agent_name=self.name,
                action="local_code_inspection",
                reason=result.summary,
                confidence=0.56,
                observations=observations,
                risks=["本地代码证据只能作为调试辅助，不能替代生产只读证据"],
                evidence=result.evidence(),
            )

        return AgentReport(
            agent_name=self.name,
            action="need_human",
            reason=result.summary,
            confidence=0.35,
            observations=observations,
            risks=result.risks or ["local_code_no_supporting_evidence"],
            evidence=result.evidence(),
        )

    def _query_text(self, request: DecisionRequest) -> str:
        parts = [
            request.entities.get("suspect_area", ""),
            request.entities.get("issue_type", ""),
            request.case.issue_type,
            request.case.original_text,
            request.case.ocr_text,
        ]
        return " ".join(item for item in parts if item)

    def _truthy(self, value: str) -> bool:
        return str(value).lower() in {"1", "true", "yes", "y", "on", "enabled"}


class Verifier:
    name = "verifier"

    def verify(
        self,
        request: DecisionRequest,
        proposal: DecisionResponse,
        agent_reports: Sequence[AgentReport],
    ) -> DecisionResponse:
        budget = self._tool_budget(request.max_tool_calls)
        checks = ["tool_budget_bounded", "gateway_only_tools", "dedupe_tool_plan"]
        violations: list[str] = []

        if proposal.action == "ask_user":
            proposal.tool_plan = []
            proposal.agent_reports = list(agent_reports)
            proposal.verification = VerificationReport(
                accepted=True,
                reason="缺字段追问不允许带工具调用计划。",
                checks=checks + ["ask_user_has_no_tool_plan"],
                tool_budget=budget,
                tool_count=0,
            )
            return proposal

        if proposal.action == "answer_from_knowledge":
            if not proposal.knowledge_source:
                violations.append("knowledge_source_required")
                return self._need_human(request, agent_reports, budget, checks, violations)
            proposal.tool_plan = []
            proposal.agent_reports = list(agent_reports)
            proposal.verification = VerificationReport(
                accepted=True,
                reason="高置信经验直答已通过来源校验，且不调用工具。",
                checks=checks + ["knowledge_source_present", "answer_has_no_tool_plan"],
                tool_budget=budget,
                tool_count=0,
            )
            return proposal

        if proposal.action == "invoke_tools":
            normalized, normalize_violations = self._normalize_tool_plan(request, proposal.tool_plan, budget)
            violations.extend(normalize_violations)
            if not normalized:
                violations.append("no_verified_tool_plan")
                return self._need_human(request, agent_reports, budget, checks, violations)

            proposal.tool_plan = normalized
            proposal.agent_reports = list(agent_reports)
            proposal.verification = VerificationReport(
                accepted=True,
                reason="工具计划通过预算、去重和可用工具校验。",
                checks=checks,
                violations=violations,
                tool_budget=budget,
                tool_count=len(normalized),
            )
            return proposal

        if proposal.action == "need_human":
            proposal.tool_plan = []
            proposal.agent_reports = list(agent_reports)
            proposal.verification = VerificationReport(
                accepted=True,
                reason="无需工具调用，转人工确认。",
                checks=checks,
                tool_budget=budget,
                tool_count=0,
            )
            return proposal

        if proposal.action == "local_code_inspection":
            proposal.tool_plan = []
            proposal.agent_reports = list(agent_reports)
            proposal.verification = VerificationReport(
                accepted=True,
                reason="本地代码检查已通过 debug-only、无工具调用和 allowlist 约束。",
                checks=checks + ["debug_local_code_explicit", "local_repo_allowlist", "no_source_snippets"],
                tool_budget=budget,
                tool_count=0,
            )
            return proposal

        violations.append(f"unsupported_action={proposal.action}")
        return self._need_human(request, agent_reports, budget, checks, violations)

    def _normalize_tool_plan(
        self,
        request: DecisionRequest,
        tool_plan: Sequence[ToolPlan],
        budget: int,
    ) -> tuple[list[ToolPlan], list[str]]:
        violations: list[str] = []
        available = {tool.name for tool in request.available_tools if tool.name}
        normalized: list[ToolPlan] = []
        seen: set[str] = set()
        for item in tool_plan:
            if not item.tool_name:
                violations.append("empty_tool_name")
                continue
            if item.tool_name in seen:
                violations.append(f"duplicate_tool={item.tool_name}")
                continue
            if available and item.tool_name not in available:
                violations.append(f"unavailable_tool={item.tool_name}")
                continue
            seen.add(item.tool_name)
            normalized.append(item)
            if len(normalized) >= budget:
                if len(tool_plan) > budget:
                    violations.append("tool_plan_truncated_by_budget")
                break
        return normalized, violations

    def _need_human(
        self,
        request: DecisionRequest,
        agent_reports: Sequence[AgentReport],
        budget: int,
        checks: Sequence[str],
        violations: Sequence[str],
    ) -> DecisionResponse:
        return DecisionResponse(
            action="need_human",
            reason="Verifier 未能确认安全可执行的工具计划，需要人工介入或补充 Gateway 能力。",
            confidence=0.38,
            agent_reports=list(agent_reports),
            verification=VerificationReport(
                accepted=False,
                reason="工具计划未通过 verifier。",
                checks=list(checks),
                violations=list(violations),
                tool_budget=budget,
                tool_count=0,
            ),
        )

    def _tool_budget(self, max_tool_calls: int) -> int:
        return max(1, min(max_tool_calls, 10))


class SupervisorAgentTeam:
    def __init__(self, local_code_agent: LocalCodeAgent | None = None) -> None:
        self.knowledge_agent = KnowledgeAgent()
        self.kline_agent = DomainAgent("kline", "kline_agent")
        self.asset_agent = DomainAgent("asset", "asset_agent")
        self.fallback_agent = FallbackAgent()
        self.local_code_agent = local_code_agent or LocalCodeAgent()
        self.verifier = Verifier()

    def plan(self, request: DecisionRequest) -> DecisionResponse:
        reports: list[AgentReport] = [
            AgentReport(
                agent_name="supervisor",
                action="route",
                reason="Supervisor 按问题域选择 specialist，并要求最终输出经过 Verifier。",
                observations=[f"issue_domain={request.case.issue_domain or 'default'}"],
            )
        ]

        knowledge_report = self.knowledge_agent.evaluate(request)
        reports.append(knowledge_report)
        if knowledge_report.action == "answer_from_knowledge":
            proposal = DecisionResponse(
                action="answer_from_knowledge",
                reason=knowledge_report.reason,
                knowledge_source=knowledge_report.knowledge_source,
                confidence=knowledge_report.confidence,
            )
            return self.verifier.verify(request, proposal, reports)

        if self.local_code_agent.should_run(request):
            local_code_report = self.local_code_agent.evaluate(request)
            reports.append(local_code_report)
            proposal = self._response_from_report(local_code_report)
            return self.verifier.verify(request, proposal, reports)

        specialist = self._select_specialist(request.case.issue_domain)
        specialist_report = specialist.evaluate(request, self)
        reports.append(specialist_report)
        proposal = self._response_from_report(specialist_report)
        return self.verifier.verify(request, proposal, reports)

    def missing_fields(self, request: DecisionRequest, domain: str | None = None) -> list[str]:
        selected_domain = domain or request.case.issue_domain or "default"
        required = REQUIRED_FIELDS_BY_DOMAIN.get(selected_domain, ())
        missing = [field for field in required if not request.entities.get(field)]
        if selected_domain == "asset" and not (request.entities.get("user_id") or request.entities.get("account_id")):
            missing.insert(0, "user_id_or_account_id")
        return missing

    def select_tools(self, request: DecisionRequest, domain: str | None = None) -> list[str]:
        selected_domain = domain or request.case.issue_domain or "default"
        preferred = DEFAULT_TOOLS_BY_DOMAIN.get(selected_domain, DEFAULT_TOOLS_BY_DOMAIN["default"])
        available = {tool.name for tool in request.available_tools if tool.name}
        if available:
            preferred = tuple(name for name in preferred if name in available)
        budget = max(1, min(request.max_tool_calls, 10))
        return list(preferred[:budget])

    def build_tool_plan(self, tool_name: str, request: DecisionRequest) -> ToolPlan:
        args = dict(request.entities)
        if request.case.case_no:
            args.setdefault("case_no", request.case.case_no)
        return ToolPlan(
            tool_name=tool_name,
            reason=f"查询 {tool_name} 获取只读证据，调用必须经过 Investigation Gateway。",
            arguments=args,
        )

    def _select_specialist(self, domain: str) -> DomainAgent | FallbackAgent:
        if domain == "kline":
            return self.kline_agent
        if domain == "asset":
            return self.asset_agent
        return self.fallback_agent

    def _response_from_report(self, report: AgentReport) -> DecisionResponse:
        return DecisionResponse(
            action=report.action,
            reason=report.reason,
            missing_fields=list(report.missing_fields),
            tool_plan=list(report.tool_plan),
            knowledge_source=report.knowledge_source,
            confidence=report.confidence,
        )
