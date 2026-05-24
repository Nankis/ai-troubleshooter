from __future__ import annotations

import json
import re
import time
import uuid
from datetime import datetime
from typing import Any

from decision_engine import CaseSnapshot, DecisionEngine, DecisionRequest
from decision_engine.models import KnowledgeCandidate, ToolPlan, ToolSpec

from .capabilities import import_capabilities
from .classifier import classify_and_extract_rules, merge_llm_result
from .config import Config
from .context_ledger import (
    compact_decision_payload,
    compact_failed_observation,
    compact_gateway_tools_payload,
    compact_knowledge_payload,
    compact_tool_observation,
    evidence_refs_from_observations,
    final_answer_verification,
)
from .gateway import GatewayHTTPClient
from .llm import LLMClient
from .repository import Repository
from .vision import ImageInput, LocalVisionClient


ENTRY_STATUSES = {"NEW", "NEED_MORE_INFO", "WAITING_USER_REPLY", "READY_TO_INVESTIGATE"}
ACTIVE_STATUSES = {"READY_TO_INVESTIGATE", "INVESTIGATING", "WAITING_TOOL_RESULT"}


class AgentPlatform:
    def __init__(
        self,
        config: Config,
        repository: Repository,
        gateway: GatewayHTTPClient | None = None,
        decision_engine: DecisionEngine | None = None,
        llm_client: LLMClient | None = None,
        vision_client: LocalVisionClient | None = None,
    ) -> None:
        self.config = config
        self.repository = repository
        self.gateway = gateway or GatewayHTTPClient(config.gateway_endpoint, config.gateway_bearer_token)
        self.decision_engine = decision_engine or DecisionEngine()
        self.llm = llm_client or LLMClient(config.llm)
        self.vision = vision_client or LocalVisionClient()

    def close(self) -> None:
        self.repository.close()

    def health(self) -> dict[str, Any]:
        return {
            "ok": True,
            "service": "agent-platform",
            "decision_layer": "python",
            "gateway_endpoint": self.config.gateway_endpoint,
            "llm_provider": self.config.llm.provider,
            "llm_model": self.config.llm.model,
        }

    def submit_chat(
        self,
        *,
        message: str,
        title: str = "",
        case_no: str = "",
        images: list[ImageInput] | None = None,
        async_process: bool = False,
    ) -> dict[str, Any]:
        text = message.strip()
        images = images or []
        if not text and not images:
            raise ValueError("message or image is required")
        vision_result = self.vision.analyze(text, images)
        case = self._upsert_case(case_no=case_no, title=title, user_text=text, ocr_text=vision_result.ocr_text)
        if async_process:
            case = self._transition(case, "READY_TO_INVESTIGATE")
            return self.case_payload(case, {"case_id": case["id"], "case_no": case["case_no"], "status": case["status"], "reply": f"[{case['case_no']}] 已开始排查。"}, processing=True)
        result = self.process_case(int(case["id"]))
        return self.case_payload(self._require_case(result["case_no"]), result, processing=False)

    def submit_channel_message(
        self,
        *,
        source: str,
        text: str,
        chat_id: str,
        thread_id: str,
        message_id: str,
        reporter_user_id: str,
        ocr_text: str = "",
        images: list[ImageInput] | None = None,
        async_process: bool = True,
    ) -> dict[str, Any]:
        images = images or []
        vision_result = self.vision.analyze(text, images)
        combined_ocr = _join_non_empty(ocr_text, vision_result.ocr_text)
        if not text.strip() and not combined_ocr.strip() and not images:
            raise ValueError("text or image is required")
        if message_id:
            existing = self.repository.find_case_by_message_id(source, message_id)
            if existing is not None:
                payload = self.case_payload(
                    existing,
                    {
                        "case_id": existing["id"],
                        "case_no": existing["case_no"],
                        "status": existing["status"],
                        "reply": f"[{existing['case_no']}] 该消息已接收，跳过重复排查。",
                        "duplicate": True,
                    },
                    processing=str(existing.get("status") or "") in ACTIVE_STATUSES,
                )
                payload["duplicate"] = True
                return payload
        case = self.repository.create_case(
            {
                "title": _trim_title(text),
                "uid": reporter_user_id,
                "source": source,
                "chat_id": chat_id,
                "thread_id": thread_id,
                "message_id": message_id,
                "reporter_user_id": reporter_user_id,
                "original_text": text,
                "ocr_text": combined_ocr,
                "timezone": "Asia/Shanghai",
            }
        )
        self.repository.add_message(int(case["id"]), "user", _message_content(text, combined_ocr))
        if async_process:
            case = self._transition(case, "READY_TO_INVESTIGATE")
            return self.case_payload(case, {"case_id": case["id"], "case_no": case["case_no"], "status": case["status"], "reply": f"[{case['case_no']}] 已开始排查。"}, processing=True)
        result = self.process_case(int(case["id"]))
        return self.case_payload(self._require_case(result["case_no"]), result, processing=False)

    def process_case(self, case_id: int) -> dict[str, Any]:
        case = self._require_case_by_id(case_id)
        if case["status"] not in ENTRY_STATUSES:
            self._record_decision(case, None, "process_skipped", "case is already being processed or in terminal status", {"status": case["status"]}, {}, "skipped")
            return {"case_id": case["id"], "case_no": case["case_no"], "status": case["status"], "reply": f"[{case['case_no']}] 当前状态为 {case['status']}，跳过重复排障。"}

        start_time = time.monotonic()
        investigation: dict[str, Any] | None = None
        try:
            self._transition(case, "READY_TO_INVESTIGATE")
            case = self._require_case_by_id(case_id)
            classification = self._classify_and_extract(case)
            self.repository.add_entities(case_id, _entities_to_rows(classification.get("entities") or {}))
            entities = self.entity_map(case_id)
            issue_domain = str(classification.get("issue_domain") or "")
            issue_type = str(classification.get("issue_type") or "")
            case = self.repository.update_case_fields(
                case_id,
                {
                    "issue_domain": issue_domain,
                    "issue_type": issue_type,
                    "uid": entities.get("uid") or entities.get("user_id") or case.get("uid") or "",
                },
            )
            self._record_context_ledger(
                case,
                "case_state",
                "classified_entities",
                f"domain={issue_domain or 'unknown'} issue_type={issue_type or 'unknown'} entity_keys={','.join(sorted(entities))}",
                [],
                {
                    "case": _safe_case(case),
                    "entity_keys": sorted(entities.keys()),
                    "max_tool_calls": self.config.max_tool_calls_per_case,
                    "max_tool_failures": self.config.max_tool_failures_per_case,
                },
                "agent-platform",
            )

            tools = self._list_gateway_tools(case)
            knowledge = self._knowledge_candidates(case)
            request = self._decision_request(case, entities, tools, knowledge)
            step_start = time.monotonic()
            decision = self.decision_engine.plan(request)
            self._record_decision(
                case,
                None,
                "orchestrator_plan",
                "Python Supervisor selected next action and verifier checked tool budget/safety",
                {"case": _safe_case(case), "entities": entities, "available_tool_count": len(tools)},
                decision.to_dict(),
                "success",
                latency_ms=_elapsed_ms(step_start),
                selected_tools=[item.tool_name for item in decision.tool_plan],
            )
            self._record_context_ledger(
                case,
                "agent_report",
                "orchestrator_plan",
                decision.reason,
                [],
                compact_decision_payload(decision),
                "supervisor",
            )

            if decision.action == "ask_user":
                return self._ask_user(case, decision.missing_fields)
            if decision.action == "answer_from_knowledge":
                return self._answer_from_knowledge(case, decision.knowledge_source, decision.reason, decision.confidence)
            if decision.action == "local_code_inspection":
                return self._need_human(case, decision.reason, decision.confidence)
            if decision.action != "invoke_tools":
                return self._need_human(case, decision.reason, decision.confidence)

            self._transition(case, "INVESTIGATING")
            case = self._require_case_by_id(case_id)
            investigation = self.repository.create_investigation(
                {
                    "case_id": case_id,
                    "agent_id": self.config.gateway_agent_id,
                    "agent_version": "python-agent-platform-v1",
                    "model_provider": self.config.llm.provider,
                    "model_name": self.config.llm.model,
                    "initial_hypothesis": f"domain={case.get('issue_domain')} issue_type={case.get('issue_type')}",
                }
            )
            tool_call_ids, observations = self._invoke_tools(case, investigation, decision.tool_plan)
            summary, confidence = self._summarize(case, observations)
            self.repository.finish_investigation(int(investigation["id"]), "finished", summary, confidence)
            self.repository.add_message(case_id, "agent", f"[{case['case_no']}] {summary}")
            self._transition(case, "NEED_HUMAN_CONFIRMATION")
            latest = self._require_case_by_id(case_id)
            return {"case_id": case_id, "case_no": case["case_no"], "status": latest["status"], "reply": f"[{case['case_no']}] {summary}", "tool_call_ids": tool_call_ids}
        except Exception as exc:
            latest = self.repository.get_case_by_id(case_id) or case
            self._record_decision(latest, investigation, "process_failure", "Python Agent Platform stopped and finalized the case", {}, {"error": str(exc)}, "failed", error_message=str(exc))
            self.repository.add_message(case_id, "system", f"排查失败：{exc}")
            self.repository.update_case_fields(case_id, {"case_status": "FAILED"})
            raise
        finally:
            elapsed = time.monotonic() - start_time
            if elapsed > self.config.max_investigation_seconds:
                self.repository.update_case_fields(case_id, {"case_status": "FAILED"})

    def overview(self) -> dict[str, Any]:
        tools: list[dict[str, Any]] = []
        tool_warning = ""
        try:
            tools = self.gateway.list_tools()
        except Exception as exc:
            tool_warning = str(exc)
        return {
            "cases": self.repository.list_recent_cases(30),
            "tools": tools,
            "capabilities": self.repository.list_capabilities(200),
            "knowledge": self.repository.list_knowledge(30),
            "tool_warning": tool_warning,
            "now": datetime.now(),
        }

    def case_payload(self, case: dict[str, Any], result: dict[str, Any] | None = None, processing: bool = False) -> dict[str, Any]:
        result = result or {"case_id": case["id"], "case_no": case["case_no"], "status": case["status"], "reply": ""}
        logs = self.repository.list_decision_logs(int(case["id"]), 100)
        status = str(case.get("status") or "")
        return {
            "case": case,
            "reply": result.get("reply", ""),
            "tool_call_ids": result.get("tool_call_ids", []),
            "missing_fields": result.get("missing_fields", []),
            "entities": self.repository.list_entities(int(case["id"])),
            "messages": self.repository.list_messages(int(case["id"])),
            "ai_decision_logs": logs,
            "context_ledger": self.repository.list_context_ledger(int(case["id"]), 100),
            "evolution_runs": [],
            "progress": build_progress(status, logs),
            "processing": processing or status in ACTIVE_STATUSES,
        }

    def get_case_payload(self, ref: str) -> dict[str, Any]:
        case = self._require_case(ref)
        return self.case_payload(case)

    def rename_case(self, ref: str, title: str) -> dict[str, Any]:
        case = self._require_case(ref)
        if not title.strip():
            raise ValueError("title is required")
        return self.repository.update_case_fields(int(case["id"]), {"case_title": title.strip()})

    def delete_case(self, ref: str) -> dict[str, Any]:
        case = self._require_case(ref)
        self.repository.delete_case(int(case["id"]))
        return {"deleted": True, "case_no": case["case_no"]}

    def save_knowledge(self, item: dict[str, Any], knowledge_id: int = 0) -> dict[str, Any]:
        if knowledge_id:
            item["id"] = knowledge_id
        if not str(item.get("title") or "").strip():
            raise ValueError("title is required")
        if not str(item.get("issue_domain") or "").strip():
            raise ValueError("issue_domain is required")
        return self.repository.upsert_knowledge(item)

    def import_capabilities(self, payload: dict[str, Any]) -> dict[str, Any]:
        payload.setdefault("created_by", "web")
        return import_capabilities(self.repository, payload)

    def publish_capability(self, capability_id: int) -> dict[str, Any]:
        item = self.repository.update_tool_capability_status(capability_id, "enabled", "web")
        item["gateway_reload"] = self._reload_gateway_capabilities()
        return item

    def disable_capability(self, capability_id: int) -> dict[str, Any]:
        item = self.repository.update_tool_capability_status(capability_id, "disabled", "web")
        item["gateway_reload"] = self._reload_gateway_capabilities()
        return item

    def _upsert_case(self, *, case_no: str, title: str, user_text: str, ocr_text: str) -> dict[str, Any]:
        if case_no:
            case = self._require_case(case_no)
            self.repository.add_message(int(case["id"]), "user", _message_content(user_text, ocr_text))
            updated = self.repository.update_case_fields(
                int(case["id"]),
                {
                    "original_text": _append_block(case.get("original_text") or "", "用户补充", user_text),
                    "ocr_text": _append_block(case.get("ocr_text") or "", "图片识别补充", ocr_text),
                    "case_status": "WAITING_USER_REPLY" if case.get("status") == "NEED_MORE_INFO" else case.get("status"),
                },
            )
            return updated
        case = self.repository.create_case(
            {
                "title": _trim_title(title or user_text),
                "uid": "web_user",
                "source": "web",
                "chat_id": "web-local",
                "thread_id": "thread_" + uuid.uuid4().hex,
                "message_id": "webmsg_" + uuid.uuid4().hex,
                "reporter_user_id": "web_user",
                "original_text": user_text,
                "ocr_text": ocr_text,
                "timezone": "Asia/Shanghai",
            }
        )
        self.repository.add_message(int(case["id"]), "user", _message_content(user_text, ocr_text))
        return case

    def _classify_and_extract(self, case: dict[str, Any]) -> dict[str, object]:
        text = f"{case.get('original_text') or ''}\n{case.get('ocr_text') or ''}"
        step_start = time.monotonic()
        rule_result = classify_and_extract_rules(text)
        llm_payload: dict[str, object] = {}
        llm_error = ""
        try:
            llm_payload = self.llm.classify_and_extract(text).payload
        except Exception as exc:
            llm_error = str(exc)
            if self.llm.is_real and not self.config.llm.allow_rule_fallback:
                self._record_decision(case, None, "classify_extract", "LLM classification failed and fallback is disabled", {"text": text}, {"rule_result": rule_result}, "failed", latency_ms=_elapsed_ms(step_start), error_message=llm_error)
                raise
        merged = merge_llm_result(rule_result, llm_payload)
        self._record_decision(
            case,
            None,
            "classify_extract",
            "classify issue domain and extract minimum troubleshooting entities in Python orchestrator",
            {"case_no": case["case_no"], "original_text": case.get("original_text"), "ocr_text": case.get("ocr_text")},
            {"rule_result": rule_result, "llm_result": llm_payload, "merged": merged, "llm_error": llm_error},
            "success" if not llm_error else "fallback",
            latency_ms=_elapsed_ms(step_start),
        )
        return merged

    def _decision_request(self, case: dict[str, Any], entities: dict[str, str], tools: list[dict[str, Any]], knowledge: list[KnowledgeCandidate]) -> DecisionRequest:
        return DecisionRequest(
            case=CaseSnapshot(
                case_no=case["case_no"],
                issue_domain=case.get("issue_domain") or "",
                issue_type=case.get("issue_type") or "",
                original_text=case.get("original_text") or "",
                ocr_text=case.get("ocr_text") or "",
                source=case.get("source") or "",
                uid=case.get("uid") or "",
                reporter_user_id=case.get("reporter_user_id") or "",
                chat_id=case.get("chat_id") or "",
            ),
            entities=entities,
            available_tools=[
                ToolSpec(
                    name=str(item.get("name") or item.get("tool_name") or ""),
                    required_scope=str(item.get("required_scope") or ""),
                    max_time_range_minutes=int(item.get("max_time_range_minutes") or 0),
                    max_limit=int(item.get("max_limit") or 0),
                )
                for item in tools
            ],
            knowledge_candidates=knowledge,
            context_ledger=self.repository.list_context_ledger(int(case["id"]), 50),
            max_tool_calls=self.config.max_tool_calls_per_case,
        )

    def _list_gateway_tools(self, case: dict[str, Any]) -> list[dict[str, Any]]:
        step_start = time.monotonic()
        try:
            tools = self.gateway.list_tools()
            self._record_decision(case, None, "gateway_tool_discovery", "list registered readonly tools from Go Investigation Gateway", {}, {"tool_count": len(tools)}, "success", latency_ms=_elapsed_ms(step_start))
            self._record_context_ledger(
                case,
                "gateway_tools",
                "registered_readonly_tools",
                f"Gateway registered readonly tool count={len(tools)}",
                [],
                compact_gateway_tools_payload(tools),
                "gateway-discovery",
            )
            return tools
        except Exception as exc:
            self._record_decision(case, None, "gateway_tool_discovery", "failed to list Gateway tools", {}, {"error": str(exc)}, "failed", latency_ms=_elapsed_ms(step_start), error_message=str(exc))
            return []

    def _knowledge_candidates(self, case: dict[str, Any]) -> list[KnowledgeCandidate]:
        rows = self.repository.list_knowledge(3, str(case.get("issue_domain") or ""), str(case.get("issue_type") or ""), "active")
        self._record_decision(case, None, "knowledge_retrieval", "retrieve platform knowledge before querying downstream business tools", {"issue_domain": case.get("issue_domain"), "issue_type": case.get("issue_type")}, {"matched_count": len(rows)}, "success")
        self._record_context_ledger(
            case,
            "knowledge_retrieval",
            "platform_knowledge_candidates",
            f"matched platform knowledge count={len(rows)}",
            [{"ref_type": "knowledge_item", "ref_id": str(row.get("id")), "tool_name": "platform_knowledge"} for row in rows if row.get("id")],
            compact_knowledge_payload(rows),
            "knowledge_agent",
        )
        return [
            KnowledgeCandidate(
                title=row.get("title") or "",
                confidence=float(row.get("confidence") or 0),
                observed_case_count=int(row.get("observed_case_count") or 0),
                requires_realtime_check=_requires_realtime(case),
                source=f"knowledge:{row.get('id')}",
                summary=row.get("typical_description") or "",
            )
            for row in rows
        ]

    def _invoke_tools(self, case: dict[str, Any], investigation: dict[str, Any], tool_plan: list[ToolPlan]) -> tuple[list[str], list[dict[str, Any]]]:
        self._transition(case, "WAITING_TOOL_RESULT")
        tool_call_ids: list[str] = []
        observations: list[dict[str, Any]] = []
        failures = 0
        for plan in tool_plan[: self.config.max_tool_calls_per_case]:
            if failures >= self.config.max_tool_failures_per_case:
                self._record_decision(case, investigation, "tool_query_stopped", "stopped tool queries after failure budget was exhausted", {"max_tool_failures": self.config.max_tool_failures_per_case}, {"tool_failures": failures}, "stopped")
                break
            step_start = time.monotonic()
            try:
                response = self.gateway.invoke_tool(
                    plan.tool_name,
                    case_no=case["case_no"],
                    agent_id=self.config.gateway_agent_id,
                    caller_user_id=case.get("uid") or case.get("reporter_user_id") or "web_user",
                    chat_id=case.get("chat_id") or "",
                    arguments=plan.arguments,
                )
                status = response.get("status") or "success"
                if status != "success":
                    failures += 1
                tool_call_ids.append(str(response.get("tool_call_id") or ""))
                observation = compact_tool_observation(plan.tool_name, str(status), response)
                observations.append(observation)
                self._record_context_ledger(
                    case,
                    "tool_evidence",
                    plan.tool_name,
                    observation["summary"],
                    observation.get("evidence_refs") or [],
                    observation,
                    f"tool:{plan.tool_name}",
                )
                self._record_decision(case, investigation, "tool_invocation", plan.reason, {"tool_name": plan.tool_name, "arguments": plan.arguments}, response, status, latency_ms=_elapsed_ms(step_start), selected_tools=[plan.tool_name])
            except Exception as exc:
                failures += 1
                observation = compact_failed_observation(plan.tool_name, str(exc))
                observations.append(observation)
                self._record_context_ledger(
                    case,
                    "tool_evidence",
                    plan.tool_name,
                    observation["summary"],
                    [],
                    observation,
                    f"tool:{plan.tool_name}",
                )
                self._record_decision(case, investigation, "tool_invocation", plan.reason, {"tool_name": plan.tool_name, "arguments": plan.arguments}, {"error": str(exc)}, "failed", latency_ms=_elapsed_ms(step_start), error_message=str(exc), selected_tools=[plan.tool_name])
        return tool_call_ids, observations

    def _summarize(self, case: dict[str, Any], observations: list[dict[str, Any]]) -> tuple[str, float]:
        step_start = time.monotonic()
        summary = _deterministic_summary(case, observations)
        confidence = 0.72 if observations else 0.38
        llm_payload: dict[str, object] = {}
        try:
            llm_payload = self.llm.summarize(_safe_case(case), observations).payload
            if llm_payload.get("summary"):
                summary = str(llm_payload["summary"])
            if isinstance(llm_payload.get("confidence"), int | float):
                confidence = float(llm_payload["confidence"])
        except Exception as exc:
            if self.llm.is_real and not self.config.llm.allow_rule_fallback:
                self._record_decision(case, None, "summarize_findings", "LLM summary failed and fallback is disabled", {"observations": observations}, {"error": str(exc)}, "failed", latency_ms=_elapsed_ms(step_start), error_message=str(exc))
                raise
        verification = final_answer_verification(observations, confidence)
        confidence = float(verification["adjusted_confidence"])
        self._record_decision(case, None, "summarize_findings", "summarize bounded tool observations and ask human owner for confirmation", {"observations": observations}, {"summary": summary, "confidence": confidence, "llm_result": llm_payload}, "success", latency_ms=_elapsed_ms(step_start))
        self._record_decision(
            case,
            None,
            "verifier_final_answer",
            verification["reason"],
            {"observation_count": len(observations)},
            verification,
            "success" if verification["accepted"] else "needs_human",
            latency_ms=0,
        )
        self._record_context_ledger(
            case,
            "final_summary",
            "bounded_evidence_summary",
            summary,
            evidence_refs_from_observations(observations),
            {
                "summary": summary,
                "confidence": confidence,
                "verification": verification,
                "observation_count": len(observations),
            },
            "verifier",
        )
        return summary, confidence

    def _ask_user(self, case: dict[str, Any], missing_fields: list[str]) -> dict[str, Any]:
        self._transition(case, "NEED_MORE_INFO")
        reply = _missing_reply(case["case_no"], missing_fields)
        self.repository.add_message(int(case["id"]), "bot", reply)
        latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "WAITING_USER_REPLY"})
        return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply, "missing_fields": missing_fields}

    def _answer_from_knowledge(self, case: dict[str, Any], source: str, reason: str, confidence: float) -> dict[str, Any]:
        self._transition(case, "INVESTIGATING")
        inv = self.repository.create_investigation({"case_id": case["id"], "agent_id": self.config.gateway_agent_id, "agent_version": "python-agent-platform-v1", "model_provider": self.config.llm.provider, "model_name": self.config.llm.model, "initial_hypothesis": "high-confidence platform knowledge matched"})
        reply = f"[{case['case_no']}] 平台经验命中：{source}。{reason} 请业务 Owner 确认根因后沉淀为正式经验。"
        self.repository.finish_investigation(int(inv["id"]), "finished", reply, confidence or 0.8)
        self.repository.add_message(int(case["id"]), "agent", reply)
        self._record_context_ledger(
            case,
            "final_summary",
            "knowledge_answer",
            reply,
            [{"ref_type": "knowledge_item", "ref_id": source, "tool_name": "platform_knowledge"}] if source else [],
            {"summary": reply, "confidence": confidence or 0.8, "knowledge_source": source},
            "knowledge_agent",
        )
        latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "NEED_HUMAN_CONFIRMATION"})
        return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply}

    def _need_human(self, case: dict[str, Any], reason: str, confidence: float) -> dict[str, Any]:
        reply = f"[{case['case_no']}] 当前无法形成安全可执行的只读工具计划：{reason}"
        self.repository.add_message(int(case["id"]), "agent", reply)
        latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "NEED_HUMAN_CONFIRMATION"})
        return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply, "confidence": confidence}

    def _transition(self, case: dict[str, Any], status: str) -> dict[str, Any]:
        return self.repository.update_case_fields(int(case["id"]), {"case_status": status})

    def _record_decision(
        self,
        case: dict[str, Any],
        investigation: dict[str, Any] | None,
        decision_type: str,
        reason: str,
        input_snapshot: Any,
        output_snapshot: Any,
        status: str,
        *,
        latency_ms: int = 0,
        error_message: str = "",
        selected_tools: list[str] | None = None,
    ) -> None:
        self.repository.add_decision_log(
            {
                "case_id": int(case["id"]),
                "investigation_id": int(investigation["id"]) if investigation and investigation.get("id") else 0,
                "agent_id": self.config.gateway_agent_id,
                "decision_type": decision_type,
                "reason": reason,
                "input": _mask(input_snapshot),
                "output": _mask(output_snapshot),
                "selected_tools": selected_tools or [],
                "status": status,
                "latency_ms": latency_ms,
                "error_message": error_message,
            }
        )

    def _record_context_ledger(
        self,
        case: dict[str, Any],
        ledger_type: str,
        ledger_key: str,
        summary: str,
        evidence_refs: list[dict[str, Any]],
        payload: Any,
        source_agent: str,
    ) -> None:
        self.repository.add_context_ledger(
            {
                "case_id": int(case["id"]),
                "ledger_type": ledger_type,
                "ledger_key": ledger_key,
                "summary": summary,
                "evidence_refs": _mask(evidence_refs),
                "payload": _mask(payload),
                "source_agent": source_agent,
            }
        )

    def _require_case(self, ref: str) -> dict[str, Any]:
        case = self.repository.get_case_by_no(ref)
        if case is None and ref.isdigit():
            case = self.repository.get_case_by_id(int(ref))
        if case is None:
            raise KeyError("case not found")
        return case

    def _require_case_by_id(self, case_id: int) -> dict[str, Any]:
        case = self.repository.get_case_by_id(case_id)
        if case is None:
            raise KeyError("case not found")
        return case

    def entity_map(self, case_id: int) -> dict[str, str]:
        out: dict[str, str] = {}
        for entity in self.repository.list_entities(case_id):
            key = str(entity.get("entity_type") or "")
            value = str(entity.get("entity_value") or "")
            if key and value and key not in out:
                out[key] = value
        return out

    def _reload_gateway_capabilities(self) -> dict[str, Any]:
        try:
            return self.gateway.reload_capabilities()
        except Exception as exc:
            return {"ok": False, "error": str(exc)}


def build_progress(status: str, logs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    steps = [
        {"key": "classify_extract", "title": "识别问题领域并抽取实体"},
        {"key": "gateway_tool_discovery", "title": "拉取 Gateway 只读工具"},
        {"key": "knowledge_retrieval", "title": "查询平台沉淀经验"},
        {"key": "orchestrator_plan", "title": "Python Supervisor 制定排查计划"},
        {"key": "tool_invocation", "title": "调用只读工具收集证据"},
        {"key": "summarize_findings", "title": "总结证据并输出结论"},
        {"key": "verifier_final_answer", "title": "校验证据引用和最终置信度"},
    ]
    by_key: dict[str, list[dict[str, Any]]] = {}
    for item in logs:
        by_key.setdefault(str(item.get("decision_type") or ""), []).append(item)
    first_pending = -1
    for idx, step in enumerate(steps):
        items = by_key.get(step["key"], [])
        if not items:
            step["status"] = "pending"
            if first_pending < 0:
                first_pending = idx
            continue
        latest = items[-1]
        step["reason"] = latest.get("reason") or ""
        step["created_at"] = latest.get("created_at")
        step["status"] = "done"
        if latest.get("status") in {"failed", "timeout", "stopped", "fallback"}:
            step["status"] = latest["status"]
    if status in ACTIVE_STATUSES and first_pending >= 0:
        steps[first_pending]["status"] = "running"
    if status in {"NEED_HUMAN_CONFIRMATION", "DONE", "FAILED"}:
        for step in steps:
            if step["status"] == "pending":
                step["status"] = "skipped"
    return steps


def _entities_to_rows(entities: dict[str, str]) -> list[dict[str, Any]]:
    return [{"entity_type": key, "entity_value": value, "source": "python-orchestrator", "confidence": 0.78} for key, value in entities.items() if value]


def _message_content(user_text: str, ocr_text: str) -> str:
    parts = [user_text.strip()] if user_text.strip() else []
    if ocr_text.strip():
        parts.append("图片识别：\n" + ocr_text.strip())
    return "\n\n".join(parts)


def _append_block(current: str, label: str, value: str) -> str:
    value = value.strip()
    if not value:
        return current
    block = f"{label}：{value}"
    return block if not current.strip() else current + "\n" + block


def _join_non_empty(*values: str) -> str:
    return "\n".join(value.strip() for value in values if value and value.strip())


def _trim_title(value: str) -> str:
    value = " ".join(value.strip().split())
    return value[:80] or "新问题"


def _missing_reply(case_no: str, missing_fields: list[str]) -> str:
    friendly = "、".join(_friendly_field(item) for item in missing_fields)
    return f"[{case_no}] 我还需要补充：{friendly}。如果不确定时间，可以直接说“今天/刚刚/大约几点”；默认按 UTC+8 处理。"


def _friendly_field(field: str) -> str:
    mapping = {
        "user_id_or_uid": "业务 uid",
        "user_id_or_account_id": "用户 ID 或账户 ID",
        "issue_type": "具体异常现象",
        "abnormal_time": "异常发生的大概时间",
    }
    return mapping.get(field, field)


def _requires_realtime(case: dict[str, Any]) -> bool:
    text = f"{case.get('original_text') or ''}\n{case.get('ocr_text') or ''}"
    return any(word in text for word in ["今日", "今天", "刚刚", "现在", "生产", "不准", "没有"])


def _safe_case(case: dict[str, Any]) -> dict[str, Any]:
    return {key: case.get(key) for key in ["case_no", "title", "uid", "source", "original_text", "ocr_text", "issue_domain", "issue_type", "status"]}


def _deterministic_summary(case: dict[str, Any], observations: list[dict[str, Any]]) -> str:
    if not observations:
        return "没有可用的只读证据，建议补充 Gateway 能力或转人工确认。"
    success = [item for item in observations if item.get("status") == "success"]
    failed = [item for item in observations if item.get("status") != "success"]
    lines = [f"已通过 Gateway 查询 {len(observations)} 个只读工具，其中成功 {len(success)} 个、失败 {len(failed)} 个。"]
    for item in observations[:5]:
        if item.get("summary"):
            lines.append(f"{item.get('tool_name')}：{item.get('summary')}")
    lines.append("请业务 Owner 基于上述证据确认最终根因；确认后平台会沉淀经验。")
    return " ".join(lines)


def _elapsed_ms(start: float) -> int:
    return int((time.monotonic() - start) * 1000)


def _mask(value: Any) -> Any:
    if isinstance(value, dict):
        out = {}
        for key, item in value.items():
            if re.search(r"password|secret|token|api[_-]?key|authorization", str(key), re.IGNORECASE):
                out[key] = "<redacted>"
            else:
                out[key] = _mask(item)
        return out
    if isinstance(value, list):
        return [_mask(item) for item in value]
    if isinstance(value, str):
        value = re.sub(r"(?i)(token|secret|password|api_key)=([A-Za-z0-9._\-]+)", r"\1=<redacted>", value)
        value = re.sub(r"1[3-9]\d{9}", "<redacted_phone>", value)
        value = re.sub(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}", "<redacted_email>", value)
    return value
