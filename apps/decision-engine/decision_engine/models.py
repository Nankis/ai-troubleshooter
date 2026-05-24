from __future__ import annotations

from dataclasses import asdict, dataclass, field
from typing import Any


@dataclass(slots=True)
class CaseSnapshot:
    case_no: str
    issue_domain: str = ""
    issue_type: str = ""
    original_text: str = ""
    ocr_text: str = ""
    source: str = ""
    uid: str = ""
    reporter_user_id: str = ""
    chat_id: str = ""

    @classmethod
    def from_dict(cls, value: dict[str, Any]) -> "CaseSnapshot":
        return cls(
            case_no=str(value.get("case_no", "")),
            issue_domain=str(value.get("issue_domain", "")),
            issue_type=str(value.get("issue_type", "")),
            original_text=str(value.get("original_text", "")),
            ocr_text=str(value.get("ocr_text", "")),
            source=str(value.get("source", "")),
            uid=str(value.get("uid", "")),
            reporter_user_id=str(value.get("reporter_user_id", "")),
            chat_id=str(value.get("chat_id", "")),
        )


@dataclass(slots=True)
class ToolSpec:
    name: str
    required_scope: str = ""
    max_time_range_minutes: int = 0
    max_limit: int = 0

    @classmethod
    def from_dict(cls, value: dict[str, Any]) -> "ToolSpec":
        return cls(
            name=str(value.get("name", "")),
            required_scope=str(value.get("required_scope", "")),
            max_time_range_minutes=int(value.get("max_time_range_minutes") or 0),
            max_limit=int(value.get("max_limit") or 0),
        )


@dataclass(slots=True)
class KnowledgeCandidate:
    title: str
    confidence: float = 0.0
    observed_case_count: int = 0
    requires_realtime_check: bool = False
    source: str = ""
    summary: str = ""

    @classmethod
    def from_dict(cls, value: dict[str, Any]) -> "KnowledgeCandidate":
        return cls(
            title=str(value.get("title", "")),
            confidence=float(value.get("confidence") or 0.0),
            observed_case_count=int(value.get("observed_case_count") or 0),
            requires_realtime_check=bool(value.get("requires_realtime_check") or False),
            source=str(value.get("source", "")),
            summary=str(value.get("summary", "")),
        )


@dataclass(slots=True)
class ContextLedgerItem:
    ledger_type: str
    ledger_key: str = ""
    summary: str = ""
    evidence_refs: list[dict[str, Any]] = field(default_factory=list)
    source_agent: str = ""

    @classmethod
    def from_dict(cls, value: dict[str, Any]) -> "ContextLedgerItem":
        refs = value.get("evidence_refs") or []
        return cls(
            ledger_type=str(value.get("ledger_type", "")),
            ledger_key=str(value.get("ledger_key", "")),
            summary=str(value.get("summary", "")),
            evidence_refs=[dict(item) for item in refs if isinstance(item, dict)],
            source_agent=str(value.get("source_agent", "")),
        )


@dataclass(slots=True)
class DecisionRequest:
    case: CaseSnapshot
    entities: dict[str, str] = field(default_factory=dict)
    available_tools: list[ToolSpec] = field(default_factory=list)
    knowledge_candidates: list[KnowledgeCandidate] = field(default_factory=list)
    context_ledger: list[ContextLedgerItem] = field(default_factory=list)
    max_tool_calls: int = 10

    @classmethod
    def from_dict(cls, value: dict[str, Any]) -> "DecisionRequest":
        raw_tools = value.get("available_tools") or []
        raw_knowledge = value.get("knowledge_candidates") or []
        raw_context_ledger = value.get("context_ledger") or []
        return cls(
            case=CaseSnapshot.from_dict(value.get("case") or {}),
            entities={str(k): str(v) for k, v in (value.get("entities") or {}).items()},
            available_tools=[ToolSpec.from_dict(v) for v in raw_tools if isinstance(v, dict)],
            knowledge_candidates=[KnowledgeCandidate.from_dict(v) for v in raw_knowledge if isinstance(v, dict)],
            context_ledger=[ContextLedgerItem.from_dict(v) for v in raw_context_ledger if isinstance(v, dict)],
            max_tool_calls=int(value.get("max_tool_calls") or 10),
        )


@dataclass(slots=True)
class ToolPlan:
    tool_name: str
    reason: str
    arguments: dict[str, Any] = field(default_factory=dict)


@dataclass(slots=True)
class AgentReport:
    agent_name: str
    action: str
    reason: str
    missing_fields: list[str] = field(default_factory=list)
    tool_plan: list[ToolPlan] = field(default_factory=list)
    knowledge_source: str = ""
    confidence: float = 0.0
    observations: list[str] = field(default_factory=list)
    risks: list[str] = field(default_factory=list)
    evidence: list[dict[str, Any]] = field(default_factory=list)


@dataclass(slots=True)
class VerificationReport:
    accepted: bool
    reason: str
    checks: list[str] = field(default_factory=list)
    violations: list[str] = field(default_factory=list)
    tool_budget: int = 0
    tool_count: int = 0


@dataclass(slots=True)
class DecisionResponse:
    action: str
    reason: str
    missing_fields: list[str] = field(default_factory=list)
    tool_plan: list[ToolPlan] = field(default_factory=list)
    knowledge_source: str = ""
    confidence: float = 0.0
    agent_reports: list[AgentReport] = field(default_factory=list)
    verification: VerificationReport | None = None

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)
