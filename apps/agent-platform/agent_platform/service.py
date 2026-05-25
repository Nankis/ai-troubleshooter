from __future__ import annotations

import json
import os
import re
import socket
import time
import uuid
from dataclasses import asdict
from datetime import datetime
from pathlib import Path
from typing import Any

from decision_engine import CaseSnapshot, DecisionEngine, DecisionRequest
from decision_engine.agent_team import SupervisorAgentTeam
from decision_engine.models import InvestigationBrief, KnowledgeCandidate, ToolPlan, ToolSpec

from .capabilities import import_capabilities
from .case_scheduler import CaseScheduler
from .classifier import classify_and_extract_rules, merge_llm_result
from .config import Config, LLMConfig
from .context_ledger import (
    compact_decision_payload,
    compact_failed_observation,
    compact_gateway_tools_payload,
    compact_knowledge_payload,
    compact_tool_observation,
    evidence_refs_from_observations,
    final_answer_verification,
)
from .decision_advisor import RuntimeLLMDecisionAdvisor
from .gateway import GatewayHTTPClient
from .llm import LLMClient
from .local_agents import discover_local_agents, probe_local_agent, runtime_id, runtime_name
from .repository import Repository
from .vision import ImageInput, LocalVisionClient, build_vision_client


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
        self.llm = llm_client or LLMClient(config.llm)
        self.decision_engine = decision_engine or self._build_decision_engine()
        self.vision = vision_client or build_vision_client(config.vision, config.llm)
        self.scheduler = CaseScheduler()

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
            "vision_provider": self.config.vision.provider,
            "vision_model": self.config.vision.model,
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
        self._record_vision_decision(case, images, vision_result)
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
        self._record_vision_decision(case, images, vision_result)
        if async_process:
            case = self._transition(case, "READY_TO_INVESTIGATE")
            return self.case_payload(case, {"case_id": case["id"], "case_no": case["case_no"], "status": case["status"], "reply": f"[{case['case_no']}] 已开始排查。"}, processing=True)
        result = self.process_case(int(case["id"]))
        return self.case_payload(self._require_case(result["case_no"]), result, processing=False)

    def process_case(self, case_id: int) -> dict[str, Any]:
        case = self._require_case_by_id(case_id)
        claim = self.scheduler.claim(str(case.get("status") or ""))
        if not claim.accepted:
            self._record_decision(case, None, "process_skipped", "case is already being processed or in terminal status", {"status": case["status"]}, {}, "skipped")
            return {"case_id": case["id"], "case_no": case["case_no"], "status": case["status"], "reply": f"[{case['case_no']}] 当前状态为 {case['status']}，跳过重复排障。"}

        start_time = time.monotonic()
        investigation: dict[str, Any] | None = None
        process_run: dict[str, Any] | None = None
        try:
            process_run = self._start_agent_run(
                case,
                agent_name="supervisor",
                agent_role="orchestrator",
                trigger_type="case_process",
                input_summary=f"case={case['case_no']} source={case.get('source') or 'unknown'}",
                payload={"case": _safe_case(case)},
            )
            self._record_agent_run_event(process_run, "case_received", "running", "Supervisor 接收排障 case", f"case_status={case['status']}")
            self._record_agent_run_event(process_run, claim.event_type, "success", "Scheduler claim case", claim.reason, asdict(claim))
            self._transition(case, claim.next_status)
            case = self._require_case_by_id(case_id)
            latest_user_text = self._latest_user_text(case)
            if _should_direct_answer_latest_message(latest_user_text, case):
                result = self._answer_direct_chat_with_decision_agent(case, process_run, latest_user_text)
                self._finish_agent_run(process_run, "completed", result.get("reply", ""), {"direct_answer": True})
                return result

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
            self._record_agent_run_event(
                process_run,
                "case_classified",
                "success",
                "完成问题分类和实体抽取",
                f"domain={issue_domain or 'unknown'} issue_type={issue_type or 'unknown'} entity_keys={','.join(sorted(entities))}",
                {"classification": classification, "entities": entities},
            )

            decision_agent = self._decision_agent_source()
            if not decision_agent:
                missing_fields = _preflight_missing_fields(case, entities)
                if missing_fields:
                    self._record_decision(
                        case,
                        None,
                        "intake_preflight",
                        "insufficient problem details; ask user before any Gateway or knowledge lookup",
                        {"case": _safe_case(case), "entities": entities},
                        {"missing_fields": missing_fields, "decision_agent_required_before_investigation": True},
                        "success",
                    )
                    self._record_agent_run_event(process_run, "intake_preflight", "success", "信息不足，进入补充询问", ",".join(missing_fields))
                    result = self._ask_user(case, missing_fields)
                    self._finish_agent_run(process_run, "completed", result.get("reply", ""))
                    return result
                result = self._require_decision_agent(case)
                self._record_agent_run_event(process_run, "decision_agent_required", "blocked", "未启用真实决策 Agent，停止排障", result.get("reply", ""))
                self._finish_agent_run(process_run, "completed", result.get("reply", ""), {"blocked": "decision_agent_required"})
                return result
            self._record_decision(
                case,
                None,
                "decision_agent_ready",
                "real decision agent is enabled before Gateway and knowledge lookup",
                {"case": _safe_case(case)},
                decision_agent,
                "success",
            )
            self._record_agent_run_event(
                process_run,
                "decision_agent_ready",
                "success",
                "真实决策 Agent 已启用",
                f"{decision_agent.get('source')}/{decision_agent.get('provider')}",
                decision_agent,
            )

            tools = self._list_gateway_tools(case)
            self._record_agent_run_event(process_run, "gateway_tools_loaded", "success", "完成 Gateway readonly tools 发现", f"tool_count={len(tools)}")
            knowledge = self._knowledge_candidates(case)
            self._record_agent_run_event(process_run, "knowledge_loaded", "success", "完成平台经验候选检索", f"knowledge_count={len(knowledge)}")
            brief = self._build_investigation_brief(case, entities, tools, knowledge)
            self._record_agent_run_event(
                process_run,
                "investigation_brief_built",
                "success",
                "生成 Brief 驱动排障目标",
                brief.goal,
                asdict(brief),
            )
            request = self._decision_request(case, entities, tools, knowledge, brief)
            step_start = time.monotonic()
            decision = self.decision_engine.plan(request)
            self._record_decision(
                case,
                None,
                "orchestrator_plan",
                "Python Supervisor selected next action and verifier checked tool budget/safety",
                {"case": _safe_case(case), "entities": entities, "available_tool_count": len(tools)},
                _decision_log_snapshot(decision),
                "success",
                latency_ms=_elapsed_ms(step_start),
                selected_tools=[item.tool_name for item in decision.tool_plan],
            )
            self._record_agent_run_event(
                process_run,
                "orchestrator_plan",
                "success",
                "Supervisor 产出排查计划",
                decision.reason,
                compact_decision_payload(decision),
            )
            self._record_agent_report_runs(case, process_run, decision)
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
                result = self._ask_user(case, decision.missing_fields)
                self._finish_agent_run(process_run, "completed", result.get("reply", ""))
                return result
            if decision.action == "answer_from_knowledge":
                result = self._answer_from_knowledge(case, decision.knowledge_source, decision.reason, decision.confidence)
                self._finish_agent_run(process_run, "completed", result.get("reply", ""))
                return result
            if decision.action == "local_code_inspection":
                result = self._answer_from_local_code(case, decision)
                self._finish_agent_run(process_run, "completed", result.get("reply", ""))
                return result
            if decision.action != "invoke_tools":
                result = self._need_human(case, decision.reason, decision.confidence)
                self._finish_agent_run(process_run, "completed", result.get("reply", ""))
                return result

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
            if process_run is not None:
                process_run = self.repository.update_agent_run(int(process_run["id"]), {"investigation_id": int(investigation["id"])})
                self._record_agent_run_event(process_run, "investigation_started", "running", "创建 investigation 记录", str(investigation.get("investigation_no") or ""))
            tool_call_ids, observations = self._invoke_tools(case, investigation, decision.tool_plan)
            self._record_agent_run_event(
                process_run,
                "tool_execution_finished",
                "success",
                "完成只读工具调用",
                f"tool_calls={len(tool_call_ids)} observations={len(observations)}",
                {"tool_call_ids": tool_call_ids, "observation_count": len(observations)},
            )
            summary, confidence = self._summarize(case, observations)
            self.repository.finish_investigation(int(investigation["id"]), "finished", summary, confidence)
            self.repository.add_message(case_id, "agent", f"[{case['case_no']}] {summary}")
            self._transition(case, "NEED_HUMAN_CONFIRMATION")
            latest = self._require_case_by_id(case_id)
            result = {"case_id": case_id, "case_no": case["case_no"], "status": latest["status"], "reply": f"[{case['case_no']}] {summary}", "tool_call_ids": tool_call_ids}
            self._finish_agent_run(process_run, "completed", result["reply"], {"confidence": confidence, "tool_call_ids": tool_call_ids})
            return result
        except Exception as exc:
            latest = self.repository.get_case_by_id(case_id) or case
            self._record_decision(latest, investigation, "process_failure", "Python Agent Platform stopped and finalized the case", {}, {"error": str(exc)}, "failed", error_message=str(exc))
            self._finish_agent_run(process_run, "failed", f"排查失败：{exc}", {"error": str(exc)}, error_message=str(exc))
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

    def register_agent_runtime(self, payload: dict[str, Any]) -> dict[str, Any]:
        providers = payload.get("provider_list") or payload.get("providers") or []
        if isinstance(providers, str):
            providers = [item.strip() for item in providers.split(",") if item.strip()]
        item = self.repository.register_agent_runtime(
            {
                "runtime_id": str(payload.get("runtime_id") or "").strip(),
                "runtime_name": str(payload.get("runtime_name") or payload.get("name") or "").strip(),
                "runtime_type": str(payload.get("runtime_type") or "local").strip(),
                "host_name": str(payload.get("host_name") or "").strip(),
                "provider_list": providers,
                "workspace_root": str(payload.get("workspace_root") or "").strip(),
                "runtime_status": str(payload.get("runtime_status") or "online").strip(),
            }
        )
        return item

    def heartbeat_agent_runtime(self, runtime_id: str, payload: dict[str, Any]) -> dict[str, Any]:
        return self.repository.heartbeat_agent_runtime(runtime_id, str(payload.get("runtime_status") or payload.get("status") or "online"))

    def list_agent_runtimes(self) -> dict[str, Any]:
        return {"items": self.repository.list_agent_runtimes(50)}

    def discover_local_agent_runtime(self, payload: dict[str, Any] | None = None) -> dict[str, Any]:
        payload = payload or {}
        workspace_root = self._local_agent_workspace_root(payload)
        providers = self._discovered_local_agent_providers(workspace_root)
        runtime = self.repository.register_agent_runtime(
            {
                "runtime_id": runtime_id(),
                "runtime_name": runtime_name(),
                "runtime_type": "local",
                "host_name": socket.gethostname(),
                "provider_list": providers,
                "workspace_root": workspace_root,
                "runtime_status": "online",
            }
        )
        return {"runtime": runtime, "providers": providers}

    def enable_local_agent_provider(self, payload: dict[str, Any]) -> dict[str, Any]:
        provider_id = _normalize_provider_id(payload.get("provider_id") or payload.get("provider") or "")
        if not provider_id:
            raise ValueError("provider_id is required")
        workspace_root = self._local_agent_workspace_root(payload)
        providers = self._discovered_local_agent_providers(workspace_root)
        target = next((item for item in providers if _normalize_provider_id(item.get("provider_id")) == provider_id), None)
        if target is None:
            raise KeyError("local agent provider not found")
        enabled = _truthy(payload.get("enabled", True))
        allow_non_llm = _truthy(payload.get("allow_non_llm", False))
        if enabled and not target.get("installed"):
            raise ValueError(f"local agent provider {provider_id} is not installed")
        if enabled and not target.get("llm_capable") and not allow_non_llm:
            raise ValueError(f"local agent provider {provider_id} is not non-interactive LLM capable")
        for provider in providers:
            current_id = _normalize_provider_id(provider.get("provider_id"))
            if enabled and provider.get("llm_capable"):
                provider["enabled"] = current_id == provider_id
            elif current_id == provider_id:
                provider["enabled"] = enabled
        runtime = self.repository.register_agent_runtime(
            {
                "runtime_id": runtime_id(),
                "runtime_name": runtime_name(),
                "runtime_type": "local",
                "host_name": socket.gethostname(),
                "provider_list": providers,
                "workspace_root": workspace_root,
                "runtime_status": "online",
            }
        )
        target = next(item for item in providers if _normalize_provider_id(item.get("provider_id")) == provider_id)
        return {"runtime": runtime, "provider": target}

    def probe_local_agent_provider(self, payload: dict[str, Any]) -> dict[str, Any]:
        provider_id = str(payload.get("provider_id") or payload.get("provider") or "").strip()
        if not provider_id:
            raise ValueError("provider_id is required")
        result = probe_local_agent(
            provider_id,
            execute=_truthy(payload.get("execute", False)),
            workspace_root=self._local_agent_workspace_root(payload),
            timeout_seconds=_bounded_int(payload.get("timeout_seconds"), 1, 120, 15),
        )
        result["enabled"] = _normalize_provider_id(result.get("provider_id")) in self._enabled_local_agent_provider_ids()
        return result

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
            "agent_runs": self._case_agent_runs(int(case["id"])),
            "ai_decision_logs": logs,
            "context_ledger": self.repository.list_context_ledger(int(case["id"]), 100),
            "investigation_brief": self._latest_investigation_brief(int(case["id"])),
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
                    "case_status": _status_after_user_message(str(case.get("status") or "")),
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

    def _build_decision_engine(self) -> DecisionEngine:
        advisor = None
        if not _decision_llm_disabled():
            advisor = RuntimeLLMDecisionAdvisor(
                self.llm,
                self._enabled_decision_local_agent_provider_id,
                timeout_seconds=self.config.llm.timeout_seconds,
                fallback_enabled=_decision_llm_enabled(self.config),
            )
        return DecisionEngine(SupervisorAgentTeam(decision_advisor=advisor))

    def _local_agent_workspace_root(self, payload: dict[str, Any]) -> str:
        configured = str(payload.get("workspace_root") or os.getenv("LOCAL_AGENT_WORKSPACE_ROOT") or "").strip()
        if configured:
            return configured
        return str(Path(__file__).resolve().parents[3])

    def _discovered_local_agent_providers(self, workspace_root: str) -> list[dict[str, Any]]:
        enabled = self._enabled_local_agent_provider_ids()
        providers = discover_local_agents(workspace_root)
        for provider in providers:
            provider_id = _normalize_provider_id(provider.get("provider_id"))
            provider["enabled"] = provider_id in enabled
        return providers

    def _enabled_local_agent_provider_ids(self) -> set[str]:
        enabled: set[str] = set()
        for runtime in self.repository.list_agent_runtimes(50):
            if str(runtime.get("runtime_id") or "") != runtime_id():
                continue
            for provider in runtime.get("provider_list") or []:
                if isinstance(provider, dict) and provider.get("enabled"):
                    provider_id = _normalize_provider_id(provider.get("provider_id"))
                    if provider_id:
                        enabled.add(provider_id)
        return enabled

    def _enabled_decision_local_agent_provider_id(self) -> str:
        providers: list[dict[str, Any]] = []
        for runtime in self.repository.list_agent_runtimes(50):
            if str(runtime.get("runtime_id") or "") != runtime_id():
                continue
            for provider in runtime.get("provider_list") or []:
                if not isinstance(provider, dict):
                    continue
                if not provider.get("enabled") or not provider.get("llm_capable") or not provider.get("installed"):
                    continue
                provider_id = _normalize_provider_id(provider.get("provider_id"))
                if provider_id:
                    providers.append({"provider_id": provider_id, "enabled_at": runtime.get("updated_at")})
        preferred = _normalize_provider_id(os.getenv("LOCAL_AGENT_PROVIDER", ""))
        if preferred:
            for provider in providers:
                if provider["provider_id"] == preferred:
                    return preferred
        return str(providers[0]["provider_id"]) if providers else ""

    def _decision_agent_source(self) -> dict[str, Any]:
        local_provider = self._enabled_decision_local_agent_provider_id()
        if local_provider:
            return {
                "source": "local_agent",
                "provider": local_provider,
                "model": local_provider,
                "runtime_id": runtime_id(),
            }
        if _decision_llm_enabled(self.config) and self.llm.is_real:
            return {
                "source": "platform_llm",
                "provider": self.config.llm.provider,
                "model": self.config.llm.model,
                "decision_llm_enabled": True,
            }
        return {}

    def _decision_agent_llm(self, decision_agent: dict[str, Any]) -> LLMClient | None:
        if decision_agent.get("source") == "local_agent":
            provider = str(decision_agent.get("provider") or "").strip()
            if not provider:
                return None
            return LLMClient(LLMConfig("local_agent", "", "", provider, self.config.llm.timeout_seconds, False))
        if decision_agent.get("source") == "platform_llm" and self.llm.is_real:
            return self.llm
        return None

    def _latest_user_text(self, case: dict[str, Any]) -> str:
        messages = self.repository.list_messages(int(case["id"]))
        for message in reversed(messages):
            if str(message.get("role") or "").lower() == "user":
                return str(message.get("content") or "").strip()
        return str(case.get("original_text") or "").strip()

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

    def _build_investigation_brief(
        self,
        case: dict[str, Any],
        entities: dict[str, str],
        tools: list[dict[str, Any]],
        knowledge: list[KnowledgeCandidate],
    ) -> InvestigationBrief:
        brief = InvestigationBrief(
            problem=_brief_problem(case),
            goal=_brief_goal(case, entities),
            success_criteria=[
                "结论必须引用平台经验、Gateway 只读工具、真实日志/DB 或 debug-only 本地代码线索之一",
                "证据不足时必须追问或转人工确认，不能编造下游状态",
                "每个工具调用必须说明对应假设和预期证据",
            ],
            constraints={
                "gateway_only": True,
                "readonly_only": True,
                "max_tool_calls": self.config.max_tool_calls_per_case,
                "max_tool_failures": self.config.max_tool_failures_per_case,
                "timeout_seconds": self.config.max_investigation_seconds,
                "local_code_debug_only": True,
            },
            hypotheses=_brief_hypotheses(case, entities, tools, knowledge),
            available_evidence=_brief_available_evidence(case, entities, tools, knowledge),
            stop_conditions=[
                "missing_required_user_identifier",
                "no_verified_tool_plan",
                "tool_budget_exhausted",
                "tool_failure_budget_exhausted",
                "gateway_timeout_or_unavailable",
                "evidence_inconclusive_requires_human",
            ],
        )
        payload = asdict(brief)
        self._record_context_ledger(
            case,
            "investigation_brief",
            "current",
            brief.goal,
            [],
            payload,
            "supervisor",
        )
        self._record_decision(
            case,
            None,
            "investigation_brief",
            "build high-level troubleshooting brief before tool planning",
            {
                "case": _safe_case(case),
                "entity_keys": sorted(entities.keys()),
                "available_tool_count": len(tools),
                "knowledge_count": len(knowledge),
            },
            payload,
            "success",
        )
        return brief

    def _latest_investigation_brief(self, case_id: int) -> dict[str, Any]:
        rows = self.repository.list_context_ledger(case_id, 5, "investigation_brief")
        if not rows:
            return {}
        payload = rows[-1].get("payload")
        return dict(payload) if isinstance(payload, dict) else {}

    def _decision_request(
        self,
        case: dict[str, Any],
        entities: dict[str, str],
        tools: list[dict[str, Any]],
        knowledge: list[KnowledgeCandidate],
        brief: InvestigationBrief | None = None,
    ) -> DecisionRequest:
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
            investigation_brief=brief,
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
        if not str(case.get("issue_domain") or "").strip():
            self._record_decision(
                case,
                None,
                "knowledge_retrieval",
                "skip platform knowledge retrieval because issue_domain is unknown",
                {"issue_domain": case.get("issue_domain"), "issue_type": case.get("issue_type")},
                {"matched_count": 0, "skipped": "unknown_issue_domain"},
                "skipped",
            )
            self._record_context_ledger(
                case,
                "knowledge_retrieval",
                "platform_knowledge_candidates",
                "skipped platform knowledge retrieval because issue_domain is unknown",
                [],
                [],
                "knowledge_agent",
            )
            return []
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
                requires_realtime_check=False,
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
            plan_snapshot = _tool_plan_snapshot(plan)
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
                self._record_decision(case, investigation, "tool_invocation", plan.reason, plan_snapshot, response, status, latency_ms=_elapsed_ms(step_start), selected_tools=[plan.tool_name])
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
                self._record_decision(case, investigation, "tool_invocation", plan.reason, plan_snapshot, {"error": str(exc)}, "failed", latency_ms=_elapsed_ms(step_start), error_message=str(exc), selected_tools=[plan.tool_name])
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
        summary = _join_boundary_notes(_boundary_notes(self, observations), summary)
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

    def _answer_direct_chat_with_decision_agent(self, case: dict[str, Any], process_run: dict[str, Any] | None, latest_text: str) -> dict[str, Any]:
        decision_agent = self._decision_agent_source()
        if not decision_agent:
            reply = (
                f"[{case['case_no']}] 当前未启用真实决策 Agent，不能用 local_rules 回答平台咨询或普通对话。"
                "请先在右侧启用 Codex/Claude Code 等本地决策 Agent，"
                "或配置真实 LLM 并设置 DECISION_LLM_ENABLED=true。"
            )
            self._record_decision(
                case,
                None,
                "decision_agent_direct_answer",
                "blocked direct chat because no real decision agent is enabled",
                {"latest_user_text": latest_text, "case": _safe_case(case)},
                {"blocked": "decision_agent_required"},
                "blocked",
            )
            self.repository.add_message(int(case["id"]), "agent", reply)
            latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "WAITING_USER_REPLY"})
            return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply, "blocked": "decision_agent_required"}

        llm = self._decision_agent_llm(decision_agent)
        if llm is None:
            reply = f"[{case['case_no']}] 决策层 Agent 配置不可用，已停止回答；未查询 Gateway 或平台经验。"
            self.repository.add_message(int(case["id"]), "agent", reply)
            latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "WAITING_USER_REPLY"})
            return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply, "blocked": "decision_agent_unavailable"}

        agent_run = self._start_agent_run(
            case,
            agent_name="llm_decision_agent",
            agent_role="decision_advisor",
            trigger_type="direct_chat",
            input_summary=latest_text,
            payload={"latest_user_text": latest_text, "decision_agent": decision_agent},
            parent_run_id=int(process_run["id"]) if process_run and process_run.get("id") else 0,
            model_provider=str(decision_agent.get("source") or ""),
            model_name=str(decision_agent.get("provider") or decision_agent.get("model") or ""),
        )
        self._record_agent_run_event(agent_run, "direct_chat_started", "running", "决策层 Agent 直接回答非排障输入", latest_text)
        step_start = time.monotonic()
        try:
            payload = {
                "latest_user_message": latest_text,
                "case": _safe_case(case),
                "runtime_status": {
                    "main_llm_provider": self.config.llm.provider,
                    "main_llm_model": self.config.llm.model,
                    "decision_agent": decision_agent,
                    "current_answer_agent": {
                        "provider": str(decision_agent.get("provider") or decision_agent.get("model") or ""),
                        "source": str(decision_agent.get("source") or ""),
                    },
                    "gateway_is_not_allowed": True,
                },
                "rules": [
                    "This is not a production troubleshooting request.",
                    "Do not call or imply Gateway, platform knowledge, downstream DB, logs, or mock evidence.",
                    "Answer concisely and say when the user needs to enable or repair a local Agent.",
                ],
            }
            llm_result = llm.answer_chat(payload)
            reply_body = str(llm_result.payload.get("reply") or "").strip()
            if not reply_body:
                raise RuntimeError("decision agent returned empty reply")
            reply = f"[{case['case_no']}] {reply_body}"
            self._record_decision(
                case,
                None,
                "decision_agent_direct_answer",
                "decision agent answered latest non-troubleshooting message without Gateway",
                {"latest_user_text": latest_text, "decision_agent": decision_agent},
                {"provider": llm_result.provider, "model": llm_result.model, "reply": reply_body, "confidence": llm_result.payload.get("confidence")},
                "success",
                latency_ms=_elapsed_ms(step_start),
            )
            self.repository.add_message(int(case["id"]), "agent", reply)
            self._record_context_ledger(
                case,
                "agent_report",
                "direct_chat_answer",
                reply,
                [],
                {"reply": reply, "decision_agent": decision_agent, "gateway_called": False},
                "llm_decision_agent",
            )
            latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "WAITING_USER_REPLY"})
            self._finish_agent_run(agent_run, "completed", reply, {"provider": llm_result.provider, "model": llm_result.model})
            return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply}
        except Exception as exc:
            reply = f"[{case['case_no']}] 决策层 Agent 直接回答失败：{exc}。我没有查询 Gateway、平台经验或下游服务。"
            self._record_decision(
                case,
                None,
                "decision_agent_direct_answer",
                "decision agent failed while answering non-troubleshooting message",
                {"latest_user_text": latest_text, "decision_agent": decision_agent},
                {"error": str(exc)},
                "failed",
                latency_ms=_elapsed_ms(step_start),
                error_message=str(exc),
            )
            self.repository.add_message(int(case["id"]), "agent", reply)
            latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "WAITING_USER_REPLY"})
            self._finish_agent_run(agent_run, "failed", reply, {"error": str(exc)}, error_message=str(exc))
            return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply}

    def _require_decision_agent(self, case: dict[str, Any]) -> dict[str, Any]:
        reply = (
            f"[{case['case_no']}] 当前未启用真实决策 Agent，已停止排障。"
            "我不会查询 Gateway、平台经验或用 local_rules 给排障结论。"
            "请先在右侧启用 Codex/Claude Code 等本地决策 Agent，"
            "或配置 Qwen/GPT/Claude/公司模型网关并设置 DECISION_LLM_ENABLED=true 后重新提交。"
        )
        self._record_decision(
            case,
            None,
            "decision_agent_ready",
            "blocked before Gateway and knowledge lookup because no real decision agent is enabled",
            {"case": _safe_case(case)},
            {
                "blocked": "decision_agent_required",
                "llm_provider": self.config.llm.provider,
                "llm_model": self.config.llm.model,
                "decision_llm_enabled": _decision_llm_enabled(self.config),
                "enabled_local_agent": "",
            },
            "blocked",
        )
        self.repository.add_message(int(case["id"]), "agent", reply)
        latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "WAITING_USER_REPLY"})
        return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply, "blocked": "decision_agent_required"}

    def _answer_from_knowledge(self, case: dict[str, Any], source: str, reason: str, confidence: float) -> dict[str, Any]:
        self._transition(case, "INVESTIGATING")
        inv = self.repository.create_investigation({"case_id": case["id"], "agent_id": self.config.gateway_agent_id, "agent_version": "python-agent-platform-v1", "model_provider": self.config.llm.provider, "model_name": self.config.llm.model, "initial_hypothesis": "high-confidence platform knowledge matched"})
        reply = f"[{case['case_no']}] 平台经验命中：{source}。{reason} 请业务 Owner 确认根因后沉淀为正式经验。"
        reply = _join_boundary_notes(_boundary_notes(self, []), reply)
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

    def _answer_from_local_code(self, case: dict[str, Any], decision: DecisionResponse) -> dict[str, Any]:
        report = next((item for item in decision.agent_reports if item.agent_name == "local_code_agent"), None)
        evidence = report.evidence if report else []
        top_hits = _local_code_top_hits(evidence, 4)
        findings = _local_code_findings(evidence, 4)
        lines = [
            f"[{case['case_no']}] 本地代码辅助排查已完成：{decision.reason}",
            "注意：这不是生产证据，只能作为最后手段的本地代码定位线索；最终仍要用 Gateway/DB/日志确认。",
        ]
        if findings:
            lines.append("")
            lines.append("优先定位：")
            lines.extend(findings)
        elif top_hits:
            lines.append("")
            lines.append("优先查看：" + "；".join(top_hits))
        if report and report.observations:
            modes = [item for item in report.observations if item.startswith("analysis_modes=") or item.startswith("analysis_backends=")]
            if modes:
                lines.append("")
                lines.append("分析范围：" + "，".join(modes))
        reply = "\n".join(lines)
        self.repository.add_message(int(case["id"]), "agent", reply)
        self._record_context_ledger(
            case,
            "final_summary",
            "local_code_inspection_summary",
            reply,
            [],
            {
                "summary": reply,
                "confidence": decision.confidence or 0.5,
                "top_hits": top_hits,
                "risk": "local_code_evidence_is_debug_only",
            },
            "local_code_agent",
        )
        latest = self.repository.update_case_fields(int(case["id"]), {"case_status": "NEED_HUMAN_CONFIRMATION"})
        return {"case_id": case["id"], "case_no": case["case_no"], "status": latest["status"], "reply": reply, "confidence": decision.confidence}

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

    def _record_vision_decision(self, case: dict[str, Any], images: list[ImageInput], vision_result: Any) -> None:
        if not images:
            return
        self._record_decision(
            case,
            None,
            "vision_analyze",
            "analyze uploaded images with configured Python Vision provider",
            {
                "image_count": len(images),
                "media_types": [item.media_type for item in images],
                "filenames": [item.filename for item in images],
            },
            {
                "provider": getattr(vision_result, "provider", ""),
                "model": getattr(vision_result, "model", ""),
                "is_real": getattr(vision_result, "is_real", False),
                "summary": getattr(vision_result, "summary", ""),
                "ocr_text": getattr(vision_result, "ocr_text", ""),
            },
            "success" if getattr(vision_result, "is_real", False) else "fallback",
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
                "summary": str(_mask(summary)),
                "evidence_refs": _mask(evidence_refs),
                "payload": _mask(payload),
                "source_agent": source_agent,
            }
        )

    def _start_agent_run(
        self,
        case: dict[str, Any],
        *,
        agent_name: str,
        agent_role: str,
        trigger_type: str,
        input_summary: str,
        payload: Any | None = None,
        parent_run_id: int = 0,
        model_provider: str = "",
        model_name: str = "",
    ) -> dict[str, Any]:
        return self.repository.create_agent_run(
            {
                "case_id": int(case["id"]),
                "parent_run_id": parent_run_id or None,
                "agent_name": agent_name,
                "agent_role": agent_role,
                "trigger_type": trigger_type,
                "run_status": "running",
                "input_summary": str(_mask(input_summary)),
                "model_provider": model_provider or self.config.llm.provider,
                "model_name": model_name or self.config.llm.model,
                "started_at": datetime.now(),
                "payload": _mask(payload or {}),
            }
        )

    def _finish_agent_run(
        self,
        run: dict[str, Any] | None,
        status: str,
        output_summary: str,
        payload: Any | None = None,
        *,
        error_message: str = "",
    ) -> None:
        if run is None:
            return
        if str(run.get("agent_role") or "") == "orchestrator" and str(run.get("trigger_type") or "") == "case_process":
            scheduler_result = self.scheduler.finish(status, failed=status == "failed")
            self._record_agent_run_event(run, scheduler_result.event_type, "failed" if status == "failed" else "success", "Scheduler finish case", scheduler_result.reason, asdict(scheduler_result))
        started_at = run.get("started_at")
        latency_ms = 0
        if isinstance(started_at, datetime):
            latency_ms = int((datetime.now() - started_at).total_seconds() * 1000)
        updated = self.repository.update_agent_run(
            int(run["id"]),
            {
                "run_status": status,
                "output_summary": str(_mask(output_summary))[:4000],
                "finished_at": datetime.now(),
                "latency_ms": latency_ms,
                "error_message": error_message,
                "payload": _mask(payload or run.get("payload") or {}),
            },
        )
        self._record_agent_run_event(updated, "run_finished", status, "Agent run finished", output_summary, payload or {})

    def _record_agent_run_event(
        self,
        run: dict[str, Any] | None,
        event_type: str,
        status: str,
        title: str,
        summary: str = "",
        payload: Any | None = None,
    ) -> None:
        if run is None:
            return
        self.repository.add_agent_run_event(
            {
                "run_id": int(run["id"]),
                "event_type": event_type,
                "event_status": status,
                "title": title,
                "summary": str(_mask(summary))[:4000],
                "payload": _mask(payload or {}),
            }
        )

    def _record_agent_report_runs(self, case: dict[str, Any], parent_run: dict[str, Any] | None, decision: DecisionResponse) -> None:
        parent_id = int(parent_run["id"]) if parent_run and parent_run.get("id") else 0
        for report in decision.agent_reports:
            if report.agent_name == "supervisor":
                self._record_agent_run_event(parent_run, "agent_report", "success", "Supervisor report", report.reason, _agent_report_payload(report))
                continue
            model_provider, model_name = _agent_report_model_fields(report, self.config.llm.provider, self.config.llm.model)
            run = self.repository.create_agent_run(
                {
                    "case_id": int(case["id"]),
                    "parent_run_id": parent_id or None,
                    "agent_name": report.agent_name,
                    "agent_role": _agent_role(report.agent_name),
                    "trigger_type": "supervisor_dispatch",
                    "run_status": "completed",
                    "input_summary": f"case={case['case_no']} action={report.action}",
                    "output_summary": report.reason,
                    "model_provider": model_provider,
                    "model_name": model_name,
                    "started_at": datetime.now(),
                    "finished_at": datetime.now(),
                    "latency_ms": 0,
                    "payload": _agent_report_payload(report),
                }
            )
            self._record_agent_run_event(run, "agent_report", "success", f"{report.agent_name} report", report.reason, _agent_report_payload(report))
        if decision.verification is not None:
            verifier_payload = asdict(decision.verification)
            run = self.repository.create_agent_run(
                {
                    "case_id": int(case["id"]),
                    "parent_run_id": parent_id or None,
                    "agent_name": "verifier",
                    "agent_role": "verifier",
                    "trigger_type": "verify_plan",
                    "run_status": "completed",
                    "input_summary": f"action={decision.action} tool_count={len(decision.tool_plan)}",
                    "output_summary": decision.verification.reason,
                    "model_provider": self.config.llm.provider,
                    "model_name": self.config.llm.model,
                    "started_at": datetime.now(),
                    "finished_at": datetime.now(),
                    "latency_ms": 0,
                    "payload": verifier_payload,
                }
            )
            self._record_agent_run_event(run, "verification", "success" if decision.verification.accepted else "failed", "Verifier report", decision.verification.reason, verifier_payload)

    def _case_agent_runs(self, case_id: int) -> list[dict[str, Any]]:
        runs = self.repository.list_agent_runs(case_id, 100)
        for run in runs:
            run["events"] = self.repository.list_agent_run_events(int(run["id"]), 100)
        return runs

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
        {"key": "decision_agent_ready", "title": "确认真实决策 Agent 可用"},
        {"key": "gateway_tool_discovery", "title": "拉取 Gateway 只读工具"},
        {"key": "knowledge_retrieval", "title": "查询平台沉淀经验"},
        {"key": "investigation_brief", "title": "生成 Brief 排障目标"},
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
        if latest.get("status") in {"failed", "timeout", "stopped", "fallback", "blocked"}:
            step["status"] = latest["status"]
    if status in ACTIVE_STATUSES and first_pending >= 0:
        steps[first_pending]["status"] = "running"
    if status in {"NEED_HUMAN_CONFIRMATION", "DONE", "FAILED"}:
        for step in steps:
            if step["status"] == "pending":
                step["status"] = "skipped"
    return steps


def _brief_problem(case: dict[str, Any]) -> str:
    text = _join_non_empty(str(case.get("original_text") or ""), str(case.get("ocr_text") or ""))
    return " ".join(text.split())[:500] or f"case={case.get('case_no') or ''}"


def _brief_goal(case: dict[str, Any], entities: dict[str, str]) -> str:
    domain = str(case.get("issue_domain") or "unknown")
    issue_type = str(case.get("issue_type") or entities.get("issue_type") or "unknown")
    uid = entities.get("uid") or entities.get("user_id") or str(case.get("uid") or "")
    if uid:
        return f"定位 {domain} 用户 {uid} 的 {issue_type} 是否由真实业务数据、日志或已沉淀经验支持"
    return f"定位 {domain} 的 {issue_type}，先确认必要实体再查询只读证据"


def _brief_hypotheses(
    case: dict[str, Any],
    entities: dict[str, str],
    tools: list[dict[str, Any]],
    knowledge: list[KnowledgeCandidate],
) -> list[dict[str, Any]]:
    text = " ".join(
        [
            str(case.get("issue_domain") or ""),
            str(case.get("issue_type") or ""),
            str(case.get("original_text") or ""),
            str(case.get("ocr_text") or ""),
            " ".join(entities.values()),
        ]
    ).lower()
    tool_names = {str(item.get("name") or item.get("tool_name") or "") for item in tools}
    out: list[dict[str, Any]] = []

    def add(item_id: str, question: str, evidence: str, tools_needed: list[str]) -> None:
        available = [name for name in tools_needed if name in tool_names]
        if tools_needed and not available:
            return
        if any(item.get("id") == item_id for item in out):
            return
        out.append(
            {
                "id": item_id,
                "question": question,
                "expected_evidence": evidence,
                "candidate_tools": available,
            }
        )

    if "health_food" in text or "health-food" in text or "推荐" in text or "餐" in text or "token" in text:
        if "token" in text or "quota" in text or "配额" in text or "消耗" in text:
            add("quota_or_entitlement", "用户 AI token/会员配额是否异常", "quota/account/membership readonly rows", ["get_health_food_ai_quota", "get_health_food_user_profile"])
        if "推荐" in text:
            add("recommendation_generation", "每日推荐是否生成、失败或与餐食指纹不一致", "recommendation status and meal fingerprint", ["get_health_food_recommendation_status"])
            add("input_data_completeness", "推荐输入餐食记录是否缺失或发生变化", "meal record range and fingerprint", ["get_health_food_meal_records"])
        add("user_eligibility", "用户资料、会员等级或设备状态是否影响结果", "user profile and membership readonly rows", ["get_health_food_user_profile"])

    add("service_error", "下游服务是否有相关错误日志", "bounded log samples from readonly log adapter", ["search_logs_by_service"])
    if knowledge:
        add("historical_pattern", "平台历史经验是否能提供优先排查路径", "knowledge candidate id and confidence", [])
    add("similar_case", "是否存在相似历史 case 可辅助定位", "similar case ids and summaries", ["get_similar_cases"])
    return out[:6]


def _brief_available_evidence(
    case: dict[str, Any],
    entities: dict[str, str],
    tools: list[dict[str, Any]],
    knowledge: list[KnowledgeCandidate],
) -> list[dict[str, Any]]:
    tool_names = [str(item.get("name") or item.get("tool_name") or "") for item in tools if item.get("name") or item.get("tool_name")]
    return [
        {
            "source": "case",
            "kind": "case_snapshot",
            "summary": f"case_no={case.get('case_no')} domain={case.get('issue_domain') or 'unknown'} issue_type={case.get('issue_type') or 'unknown'}",
        },
        {
            "source": "platform_mysql",
            "kind": "entities",
            "summary": "entity_keys=" + ",".join(sorted(entities.keys())),
        },
        {
            "source": "gateway",
            "kind": "readonly_tools",
            "count": len(tool_names),
            "names": tool_names[:12],
        },
        {
            "source": "platform_mysql",
            "kind": "knowledge_candidates",
            "count": len(knowledge),
            "sources": [item.source for item in knowledge[:5]],
        },
    ]


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


def _status_after_user_message(current: str) -> str:
    return current if current in ACTIVE_STATUSES else "WAITING_USER_REPLY"


def _missing_reply(case_no: str, missing_fields: list[str]) -> str:
    if "problem_description" in missing_fields:
        return f"[{case_no}] 请描述具体生产问题，例如受影响的服务、用户 uid、异常现象和大概时间。只发问候或泛泛提问时，我不会查询平台经验或下游服务。"
    friendly = "、".join(_friendly_field(item) for item in missing_fields)
    return f"[{case_no}] 我还需要补充：{friendly}。如果不确定时间，可以直接说“今天/刚刚/大约几点”；默认按 UTC+8 处理。"


def _should_direct_answer_latest_message(latest_text: str, case: dict[str, Any]) -> bool:
    text = latest_text.strip()
    if not text:
        return False
    if _looks_like_missing_field_answer(text, case):
        return False
    if _has_business_troubleshooting_signal(text):
        return _has_platform_override_signal(text)
    if _has_platform_meta_signal(text):
        return True
    if _looks_like_general_question(text):
        return True
    return False


def _looks_like_missing_field_answer(text: str, case: dict[str, Any]) -> bool:
    if not str(case.get("issue_domain") or case.get("issue_type") or "").strip():
        return False
    cleaned = text.strip()
    if re.fullmatch(r"(?:uid|user[_ -]?id)?[:：]?\s*[A-Za-z0-9_.@:-]{2,80}", cleaned, flags=re.IGNORECASE):
        return True
    if len(cleaned) <= 40 and any(word in cleaned for word in ("今天", "昨天", "刚刚", "上午", "下午", "晚上", "UTC", "utc", "+8", "Asia/Shanghai")):
        return True
    return False


def _has_platform_meta_signal(text: str) -> bool:
    lowered = text.lower()
    words = (
        "模型",
        "llm",
        "agent",
        "claude",
        "codex",
        "cursor",
        "decision",
        "决策层",
        "本地决策",
        "gateway",
        "网关",
        "mock",
        "local_rules",
        "规则",
        "瞎说",
        "胡言",
        "乱说",
        "骗",
        "忽悠",
        "你刚才",
        "你现在",
        "怎么用",
        "怎么配置",
        "启用",
        "停用",
    )
    return any(word in lowered for word in words)


def _has_platform_override_signal(text: str) -> bool:
    lowered = text.lower()
    words = (
        "模型",
        "llm",
        "决策层",
        "本地决策",
        "decision",
        "claude code",
        "codex cli",
        "cursor agent",
        "gateway",
        "网关",
        "mock",
        "local_rules",
        "规则",
        "瞎说",
        "胡言",
        "乱说",
        "骗",
        "忽悠",
        "你刚才",
        "怎么配置",
        "启用",
        "停用",
    )
    return any(word in lowered for word in words)


def _has_business_troubleshooting_signal(text: str) -> bool:
    lowered = text.lower()
    words = (
        "health-food",
        "health_food",
        "asset-service",
        "market-service",
        "uid",
        "user_id",
        "用户",
        "账户",
        "订单",
        "资产",
        "行情",
        "k线",
        "推荐",
        "餐食",
        "配额",
        "日志",
        "报错",
        "异常",
        "失败",
        "不准",
        "不对",
        "没有",
        "缺失",
        "超时",
        "生产",
        "接口",
    )
    return any(word in lowered for word in words)


def _looks_like_general_question(text: str) -> bool:
    cleaned = text.strip()
    if len(cleaned) > 120:
        return False
    question_words = ("?", "？", "什么", "怎么", "为什么", "能不能", "可以", "是谁", "如何")
    return any(word in cleaned for word in question_words)


def _preflight_missing_fields(case: dict[str, Any], entities: dict[str, str]) -> list[str]:
    text = _case_text(case)
    if _low_signal_user_text(text):
        return ["problem_description"]
    issue_domain = str(case.get("issue_domain") or "").strip()
    if not issue_domain:
        return ["problem_description"]
    if issue_domain in {"health_food", "asset", "market"} and not _has_user_identifier(entities):
        return ["user_id_or_uid"]
    return []


def _case_text(case: dict[str, Any]) -> str:
    return _join_non_empty(str(case.get("original_text") or ""), str(case.get("ocr_text") or ""))


def _low_signal_user_text(text: str) -> bool:
    cleaned = re.sub(r"\s+", "", text.strip().lower())
    if not cleaned:
        return True
    greetings = {"hi", "hello", "你好", "您好", "在吗", "哈喽", "嗨"}
    if cleaned in greetings:
        return True
    diagnostic_words = ("报错", "异常", "不对", "失败", "没有", "缺失", "慢", "超时", "错误", "排查", "token", "推荐", "登录", "支付", "订单", "资产", "行情", "health-food")
    return len(cleaned) < 8 and not any(word in cleaned for word in diagnostic_words)


def _has_user_identifier(entities: dict[str, str]) -> bool:
    for key in ("uid", "user_id", "account_id", "member_id"):
        if str(entities.get(key) or "").strip():
            return True
    return False


def _decision_log_snapshot(decision: DecisionResponse) -> dict[str, Any]:
    payload = decision.to_dict()
    compact_reports: list[dict[str, Any]] = []
    for report in payload.get("agent_reports") or []:
        if not isinstance(report, dict):
            continue
        item = dict(report)
        item["observations"] = [str(value)[:240] for value in (item.get("observations") or [])[:16]]
        item["risks"] = list((item.get("risks") or [])[:8])
        item["evidence"] = _compact_decision_evidence(str(item.get("agent_name") or ""), item.get("evidence") or [])
        compact_reports.append(item)
    payload["agent_reports"] = compact_reports
    return payload


def _tool_plan_snapshot(plan: ToolPlan) -> dict[str, Any]:
    return {
        "tool_name": plan.tool_name,
        "arguments": plan.arguments,
        "hypothesis_id": plan.hypothesis_id,
        "expected_evidence": plan.expected_evidence,
    }


def _agent_report_payload(report: Any) -> dict[str, Any]:
    payload = asdict(report)
    payload["observations"] = [str(value)[:240] for value in (payload.get("observations") or [])[:16]]
    payload["risks"] = list((payload.get("risks") or [])[:8])
    payload["evidence"] = _compact_decision_evidence(str(payload.get("agent_name") or ""), payload.get("evidence") or [])
    return _mask(payload)


def _agent_role(agent_name: str) -> str:
    if agent_name == "knowledge_agent":
        return "knowledge"
    if agent_name == "local_code_agent":
        return "local_code"
    if agent_name == "llm_decision_agent":
        return "decision_advisor"
    if agent_name == "verifier":
        return "verifier"
    if agent_name.endswith("_agent"):
        return "specialist"
    return "agent"


def _agent_report_model_fields(report: Any, default_provider: str, default_model: str) -> tuple[str, str]:
    if str(getattr(report, "agent_name", "")) != "llm_decision_agent":
        return default_provider, default_model
    observations = [str(value) for value in (getattr(report, "observations", None) or [])]
    provider = next((item.split("=", 1)[1] for item in observations if item.startswith("provider=")), "")
    model = next((item.split("=", 1)[1] for item in observations if item.startswith("model=")), "")
    local_provider = next((item.split("=", 1)[1] for item in observations if item.startswith("local_provider=")), "")
    if local_provider:
        return "local_agent", local_provider
    if provider or model:
        return provider or default_provider, model or default_model
    return default_provider, default_model


def _decision_llm_enabled(config: Config) -> bool:
    configured = os.getenv("DECISION_LLM_ENABLED", "").strip().lower()
    if configured in {"1", "true", "yes", "y", "on", "enabled"}:
        return _real_llm_provider(config.llm.provider)
    if configured in {"0", "false", "no", "n", "off", "disabled"}:
        return False
    return config.llm.provider.lower().strip() in {"local_agent", "local_cli", "local-agent", "local-cli"}


def _decision_llm_disabled() -> bool:
    return os.getenv("DECISION_LLM_ENABLED", "").strip().lower() in {"0", "false", "no", "n", "off", "disabled"}


def _real_llm_provider(provider: str) -> bool:
    return provider.lower().strip() not in {"", "local", "local_rules", "rules"}


def _normalize_provider_id(value: Any) -> str:
    return str(value or "").strip().lower().replace("-", "_")


def _truthy(value: Any) -> bool:
    if isinstance(value, bool):
        return value
    return str(value or "").strip().lower() in {"1", "true", "yes", "y", "on", "enabled"}


def _bounded_int(value: Any, minimum: int, maximum: int, default: int) -> int:
    try:
        numeric = int(value)
    except (TypeError, ValueError):
        numeric = default
    return max(minimum, min(maximum, numeric))


def _compact_decision_evidence(agent_name: str, evidence: Any) -> list[dict[str, Any]]:
    if not isinstance(evidence, list):
        return []
    if agent_name != "local_code_agent":
        return evidence[:20]
    compact: list[dict[str, Any]] = []
    for row in evidence[:8]:
        if not isinstance(row, dict):
            continue
        compact.append(
            {
                "file_path": row.get("file_path"),
                "primary_symbol": row.get("primary_symbol"),
                "line_range": row.get("line_range"),
                "line_numbers": row.get("line_numbers"),
                "suspect_reasons": list((row.get("suspect_reasons") or [])[:4]),
                "follow_up_checks": list((row.get("follow_up_checks") or [])[:3]),
                "code_excerpt_line_count": len(row.get("code_excerpt") or []),
                "analysis_modes": row.get("analysis_modes"),
            }
        )
    return compact


def _local_code_top_hits(evidence: list[dict[str, Any]], limit: int) -> list[str]:
    hits: list[str] = []
    for item in evidence[:limit]:
        file_path = str(item.get("file_path") or "").strip()
        if not file_path:
            continue
        primary = item.get("primary_symbol") if isinstance(item.get("primary_symbol"), dict) else {}
        line_range = item.get("line_range") if isinstance(item.get("line_range"), dict) else {}
        raw_lines = item.get("line_numbers") or []
        line_numbers = [str(value) for value in raw_lines[:5]]
        raw_terms = item.get("matched_terms") or []
        terms = [str(value) for value in raw_terms[:4] if str(value).strip()]
        detail = file_path
        if line_range.get("start") and line_range.get("end"):
            detail += f":{line_range.get('start')}-{line_range.get('end')}"
        elif line_numbers:
            detail += f":{','.join(line_numbers)}"
        if primary.get("name"):
            detail += f" `{primary.get('name')}`"
        if terms:
            detail += f" ({'/'.join(terms)})"
        hits.append(detail)
    return hits


def _local_code_findings(evidence: list[dict[str, Any]], limit: int) -> list[str]:
    findings: list[str] = []
    for index, item in enumerate(evidence[:limit], start=1):
        file_path = str(item.get("file_path") or "").strip()
        if not file_path:
            continue
        primary = item.get("primary_symbol") if isinstance(item.get("primary_symbol"), dict) else {}
        line_range = item.get("line_range") if isinstance(item.get("line_range"), dict) else {}
        line_numbers = [str(value) for value in (item.get("line_numbers") or [])[:8]]
        reasons = [str(value) for value in (item.get("suspect_reasons") or [])[:4] if str(value).strip()]
        checks = [str(value) for value in (item.get("follow_up_checks") or [])[:3] if str(value).strip()]
        excerpt = item.get("code_excerpt") if isinstance(item.get("code_excerpt"), list) else []

        block = [
            f"{index}. {file_path}{_format_line_range(line_range)}",
        ]
        if primary.get("name"):
            symbol_line = f"L{primary.get('line_number')}" if primary.get("line_number") else "-"
            block.append(f"   - 方法/符号：`{primary.get('name')}`（{primary.get('kind') or 'symbol'}，{symbol_line}）")
        if line_numbers:
            block.append(f"   - 命中行：{', '.join('L' + value for value in line_numbers)}")
        if reasons:
            block.append("   - 可疑点：" + "；".join(reasons))
        if checks:
            block.append("   - 建议核对：" + "；".join(checks))
        code_lines = _format_code_excerpt(excerpt)
        if code_lines:
            block.append("   - 相关代码：")
            block.extend(code_lines)
        findings.append("\n".join(block))
    return findings


def _format_line_range(value: dict[str, Any]) -> str:
    start = value.get("start")
    end = value.get("end")
    if start and end:
        return f":{start}-{end}"
    return ""


def _format_code_excerpt(excerpt: list[Any], limit: int = 8) -> list[str]:
    lines: list[str] = []
    for item in excerpt[:limit]:
        if not isinstance(item, dict):
            continue
        line_number = item.get("line_number")
        text = str(item.get("text") or "")
        lines.append(f"     L{line_number}: {text}")
    return lines


def _friendly_field(field: str) -> str:
    mapping = {
        "user_id_or_uid": "业务 uid",
        "user_id_or_account_id": "用户 ID 或账户 ID",
        "problem_description": "具体生产问题",
        "issue_type": "具体异常现象",
        "abnormal_time": "异常发生的大概时间",
    }
    return mapping.get(field, field)


def _safe_case(case: dict[str, Any]) -> dict[str, Any]:
    return {key: case.get(key) for key in ["case_no", "title", "uid", "source", "issue_domain", "issue_type", "status"]}


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


def _boundary_notes(_platform: AgentPlatform, observations: list[dict[str, Any]]) -> list[str]:
    notes: list[str] = []
    if _observations_contain_mock(observations):
        notes.append("注意：当前 Gateway 返回的是 mock adapter 证据，只能验证排障链路，不能作为真实业务结论。")
    return notes


def _join_boundary_notes(notes: list[str], summary: str) -> str:
    if not notes:
        return summary
    return " ".join([*notes, summary])


def _observations_contain_mock(observations: list[dict[str, Any]]) -> bool:
    for item in observations:
        try:
            text = json.dumps(item, ensure_ascii=False).lower()
        except TypeError:
            text = str(item).lower()
        if "mock" in text:
            return True
    return False


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
        value = re.sub(r"(?i)\b(available[_-]?tokens?|tokens?|secret|password|api[_-]?key|authorization)\s*[:=]\s*([A-Za-z0-9._\-]+)", r"\1=<redacted>", value)
        value = re.sub(r"1[3-9]\d{9}", "<redacted_phone>", value)
        value = re.sub(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}", "<redacted_email>", value)
    return value
