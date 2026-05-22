from __future__ import annotations

from .agent_team import DEFAULT_TOOLS_BY_DOMAIN, REQUIRED_FIELDS_BY_DOMAIN, SupervisorAgentTeam
from .models import DecisionRequest, DecisionResponse, ToolPlan


class DecisionEngine:
    def __init__(self, supervisor: SupervisorAgentTeam | None = None) -> None:
        self.supervisor = supervisor or SupervisorAgentTeam()

    def plan(self, request: DecisionRequest) -> DecisionResponse:
        return self.supervisor.plan(request)

    def _missing_fields(self, request: DecisionRequest) -> list[str]:
        return self.supervisor.missing_fields(request)

    def _select_tools(self, request: DecisionRequest) -> list[str]:
        return self.supervisor.select_tools(request)

    def _tool_plan(self, tool_name: str, request: DecisionRequest) -> ToolPlan:
        return self.supervisor.build_tool_plan(tool_name, request)

    def _best_knowledge(self, request: DecisionRequest):
        return self.supervisor.knowledge_agent.best_knowledge(request)
