from __future__ import annotations

import itertools
import base64
import hashlib
import json
import os
import tempfile
import unittest
from unittest import mock
from dataclasses import replace
from datetime import datetime
from decimal import Decimal
from pathlib import Path
from typing import Any

from fastapi.testclient import TestClient
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes

from decision_engine import DecisionRequest, DecisionResponse, VerificationReport

from agent_platform.config import ChatPlatformConfig, Config, LLMConfig, VisionConfig, load_config
from agent_platform.gateway import GatewayHTTPClient
from agent_platform.llm import LLMClient, LLMResult
from agent_platform.repository import _json_or_none
from agent_platform.server import create_app
from agent_platform.service import AgentPlatform


class MemoryRepository:
    def __init__(self) -> None:
        self.case_ids = itertools.count(1)
        self.message_ids = itertools.count(1)
        self.entity_ids = itertools.count(1)
        self.investigation_ids = itertools.count(1)
        self.decision_ids = itertools.count(1)
        self.ledger_ids = itertools.count(1)
        self.runtime_ids = itertools.count(1)
        self.agent_run_ids = itertools.count(1)
        self.agent_run_event_ids = itertools.count(1)
        self.knowledge_ids = itertools.count(1)
        self.capability_ids = itertools.count(1)
        self.cases: dict[int, dict[str, Any]] = {}
        self.messages: dict[int, list[dict[str, Any]]] = {}
        self.entities: dict[int, list[dict[str, Any]]] = {}
        self.investigations: dict[int, dict[str, Any]] = {}
        self.decisions: dict[int, list[dict[str, Any]]] = {}
        self.context_ledger: dict[int, list[dict[str, Any]]] = {}
        self.runtimes: dict[str, dict[str, Any]] = {}
        self.agent_runs: dict[int, dict[str, Any]] = {}
        self.agent_run_events: dict[int, list[dict[str, Any]]] = {}
        self.knowledge: dict[int, dict[str, Any]] = {}
        self.capabilities: dict[int, dict[str, Any]] = {}
        self.services: dict[str, dict[str, Any]] = {}

    def close(self) -> None:
        return

    def create_case(self, data: dict[str, Any]) -> dict[str, Any]:
        case_id = next(self.case_ids)
        now = datetime.now()
        item = {
            "id": case_id,
            "case_no": f"case_20260524_{case_id:06d}",
            "title": data.get("title") or "",
            "uid": data.get("uid") or "",
            "source": data.get("source") or "web",
            "chat_id": data.get("chat_id") or "",
            "thread_id": data.get("thread_id") or "",
            "message_id": data.get("message_id") or "",
            "reporter_user_id": data.get("reporter_user_id") or "",
            "original_text": data.get("original_text") or "",
            "ocr_text": data.get("ocr_text") or "",
            "issue_domain": "",
            "issue_type": "",
            "status": "NEW",
            "priority": "normal",
            "timezone": data.get("timezone") or "Asia/Shanghai",
            "created_at": now,
            "updated_at": now,
            "version": 0,
        }
        self.cases[case_id] = item
        return dict(item)

    def get_case_by_no(self, case_no: str) -> dict[str, Any] | None:
        for item in self.cases.values():
            if item["case_no"] == case_no:
                return dict(item)
        return None

    def get_case_by_id(self, case_id: int) -> dict[str, Any] | None:
        item = self.cases.get(case_id)
        return dict(item) if item else None

    def find_case_by_message_id(self, source: str, message_id: str) -> dict[str, Any] | None:
        for item in self.cases.values():
            if item.get("source") == source and item.get("message_id") == message_id:
                return dict(item)
        return None

    def list_recent_cases(self, limit: int = 30) -> list[dict[str, Any]]:
        return [dict(item) for item in list(self.cases.values())[-limit:]][::-1]

    def update_case_fields(self, case_id: int, fields: dict[str, Any]) -> dict[str, Any]:
        item = self.cases[case_id]
        alias = {"case_title": "title", "case_status": "status"}
        for key, value in fields.items():
            item[alias.get(key, key)] = value
        item["updated_at"] = datetime.now()
        item["version"] += 1
        return dict(item)

    def delete_case(self, case_id: int) -> None:
        self.cases.pop(case_id, None)

    def add_message(self, case_id: int, role: str, content: str, content_type: str = "text") -> dict[str, Any]:
        item = {"id": next(self.message_ids), "case_id": case_id, "role": role, "content": content, "content_type": content_type, "created_at": datetime.now()}
        self.messages.setdefault(case_id, []).append(item)
        return dict(item)

    def list_messages(self, case_id: int) -> list[dict[str, Any]]:
        return [dict(item) for item in self.messages.get(case_id, [])]

    def add_entities(self, case_id: int, entities: list[dict[str, Any]]) -> None:
        bucket = self.entities.setdefault(case_id, [])
        for entity in entities:
            key = entity.get("entity_type")
            value = entity.get("entity_value")
            if not any(item.get("entity_type") == key and item.get("entity_value") == value for item in bucket):
                bucket.append({"id": next(self.entity_ids), "case_id": case_id, **entity})

    def list_entities(self, case_id: int) -> list[dict[str, Any]]:
        return [dict(item) for item in self.entities.get(case_id, [])]

    def create_investigation(self, item: dict[str, Any]) -> dict[str, Any]:
        inv_id = next(self.investigation_ids)
        saved = {"id": inv_id, "investigation_no": f"inv_20260524_{inv_id:06d}", "created_at": datetime.now(), **item}
        self.investigations[inv_id] = saved
        return dict(saved)

    def finish_investigation(self, investigation_id: int, status: str, summary: str, confidence: float | None) -> dict[str, Any]:
        item = self.investigations[investigation_id]
        item.update({"investigation_status": status, "final_summary": summary, "confidence": confidence, "finished_at": datetime.now()})
        return dict(item)

    def add_decision_log(self, item: dict[str, Any]) -> dict[str, Any]:
        saved = {"id": next(self.decision_ids), "created_at": datetime.now(), **item}
        self.decisions.setdefault(item["case_id"], []).append(saved)
        return dict(saved)

    def list_decision_logs(self, case_id: int, limit: int = 100) -> list[dict[str, Any]]:
        return [dict(item) for item in self.decisions.get(case_id, [])[-limit:]]

    def add_context_ledger(self, item: dict[str, Any]) -> dict[str, Any]:
        saved = {"id": next(self.ledger_ids), "created_at": datetime.now(), "updated_at": datetime.now(), **item}
        self.context_ledger.setdefault(item["case_id"], []).append(saved)
        return dict(saved)

    def list_context_ledger(self, case_id: int, limit: int = 100, ledger_type: str = "") -> list[dict[str, Any]]:
        items = self.context_ledger.get(case_id, [])
        if ledger_type:
            items = [item for item in items if item.get("ledger_type") == ledger_type]
        return [dict(item) for item in items[-limit:]]

    def register_agent_runtime(self, item: dict[str, Any]) -> dict[str, Any]:
        runtime_id = item["runtime_id"]
        saved = {
            "id": self.runtimes.get(runtime_id, {}).get("id") or next(self.runtime_ids),
            "runtime_id": runtime_id,
            "runtime_name": item.get("runtime_name") or runtime_id,
            "runtime_type": item.get("runtime_type") or "local",
            "host_name": item.get("host_name") or "",
            "provider_list": list(item.get("provider_list") or []),
            "workspace_root": item.get("workspace_root") or "",
            "status": item.get("runtime_status") or "online",
            "last_heartbeat_at": datetime.now(),
            "registered_at": self.runtimes.get(runtime_id, {}).get("registered_at") or datetime.now(),
            "created_at": self.runtimes.get(runtime_id, {}).get("created_at") or datetime.now(),
            "updated_at": datetime.now(),
        }
        self.runtimes[runtime_id] = saved
        return dict(saved)

    def heartbeat_agent_runtime(self, runtime_id: str, status: str = "online") -> dict[str, Any]:
        item = self.runtimes[runtime_id]
        item["status"] = status
        item["last_heartbeat_at"] = datetime.now()
        item["updated_at"] = datetime.now()
        return dict(item)

    def list_agent_runtimes(self, limit: int = 50, status: str = "") -> list[dict[str, Any]]:
        items = list(self.runtimes.values())
        if status:
            items = [item for item in items if item.get("status") == status]
        return [dict(item) for item in items[:limit]]

    def create_agent_run(self, item: dict[str, Any]) -> dict[str, Any]:
        run_id = next(self.agent_run_ids)
        saved = {
            "id": run_id,
            "run_no": f"run_20260524_{run_id:06d}",
            "case_id": item["case_id"],
            "investigation_id": item.get("investigation_id"),
            "parent_run_id": item.get("parent_run_id"),
            "runtime_id": item.get("runtime_id") or "",
            "agent_name": item["agent_name"],
            "agent_role": item.get("agent_role") or "specialist",
            "trigger_type": item.get("trigger_type") or "case_process",
            "status": item.get("run_status") or "queued",
            "input_summary": item.get("input_summary") or "",
            "output_summary": item.get("output_summary") or "",
            "model_provider": item.get("model_provider") or "",
            "model_name": item.get("model_name") or "",
            "started_at": item.get("started_at"),
            "finished_at": item.get("finished_at"),
            "latency_ms": item.get("latency_ms") or 0,
            "error_message": item.get("error_message") or "",
            "payload": item.get("payload") or {},
            "created_at": datetime.now(),
            "updated_at": datetime.now(),
        }
        self.agent_runs[run_id] = saved
        return dict(saved)

    def update_agent_run(self, run_id: int, fields: dict[str, Any]) -> dict[str, Any]:
        item = self.agent_runs[run_id]
        alias = {"run_status": "status"}
        for key, value in fields.items():
            item[alias.get(key, key)] = value
        item["updated_at"] = datetime.now()
        return dict(item)

    def add_agent_run_event(self, item: dict[str, Any]) -> dict[str, Any]:
        run_id = int(item["run_id"])
        bucket = self.agent_run_events.setdefault(run_id, [])
        event = {
            "id": next(self.agent_run_event_ids),
            "run_id": run_id,
            "event_seq": len(bucket) + 1,
            "event_type": item["event_type"],
            "status": item.get("event_status") or "info",
            "title": item["title"],
            "summary": item.get("summary") or "",
            "payload": item.get("payload") or {},
            "created_at": datetime.now(),
            "updated_at": datetime.now(),
        }
        bucket.append(event)
        return dict(event)

    def list_agent_runs(self, case_id: int, limit: int = 100) -> list[dict[str, Any]]:
        items = [item for item in self.agent_runs.values() if item["case_id"] == case_id]
        return [dict(item) for item in items[:limit]]

    def list_agent_run_events(self, run_id: int, limit: int = 200) -> list[dict[str, Any]]:
        return [dict(item) for item in self.agent_run_events.get(run_id, [])[:limit]]

    def list_knowledge(self, limit: int = 30, issue_domain: str = "", issue_type: str = "", status: str = "") -> list[dict[str, Any]]:
        items = list(self.knowledge.values())
        if issue_domain:
            items = [item for item in items if item.get("issue_domain") == issue_domain]
        if issue_type:
            items = [item for item in items if item.get("issue_type") == issue_type]
        if status:
            items = [item for item in items if item.get("status") == status]
        return [dict(item) for item in items[:limit]]

    def get_knowledge(self, knowledge_id: int) -> dict[str, Any] | None:
        item = self.knowledge.get(knowledge_id)
        return dict(item) if item else None

    def upsert_knowledge(self, item: dict[str, Any]) -> dict[str, Any]:
        knowledge_id = int(item.get("id") or next(self.knowledge_ids))
        saved = {"id": knowledge_id, "status": item.get("knowledge_status") or "active", "observed_case_count": item.get("observed_case_count") or 1, "confidence": item.get("confidence") or 0.7, **item}
        self.knowledge[knowledge_id] = saved
        return dict(saved)

    def delete_knowledge(self, knowledge_id: int) -> None:
        self.knowledge.pop(knowledge_id, None)

    def list_capabilities(self, limit: int = 200, status: str = "", source_type: str = "") -> list[dict[str, Any]]:
        return [dict(item) for item in self.capabilities.values()][:limit]

    def upsert_business_service(self, item: dict[str, Any]) -> dict[str, Any]:
        saved = {"id": len(self.services) + 1, **item}
        self.services[item["service_name"]] = saved
        return dict(saved)

    def upsert_tool_capability(self, item: dict[str, Any]) -> dict[str, Any]:
        cap_id = next(self.capability_ids)
        saved = {"id": cap_id, **item}
        self.capabilities[cap_id] = saved
        return dict(saved)

    def update_tool_capability_status(self, capability_id: int, status: str, published_by: str) -> dict[str, Any]:
        self.capabilities[capability_id]["tool_status"] = status
        self.capabilities[capability_id]["published_by"] = published_by
        return dict(self.capabilities[capability_id])


class FakeGateway(GatewayHTTPClient):
    def __init__(self) -> None:
        self.invocations: list[tuple[str, dict[str, Any]]] = []

    def list_tools(self) -> list[dict[str, Any]]:
        return [
            {"name": "get_health_food_user_profile", "required_scope": "health_food:user:read"},
            {"name": "get_health_food_ai_quota", "required_scope": "health_food:ai_quota:read"},
            {"name": "get_health_food_meal_records", "required_scope": "health_food:meal:read"},
            {"name": "get_health_food_recommendation_status", "required_scope": "health_food:recommendation:read"},
            {"name": "search_logs_by_service", "required_scope": "logs:read_summary"},
        ]

    def invoke_tool(self, tool_name: str, **kwargs: Any) -> dict[str, Any]:
        self.invocations.append((tool_name, kwargs["arguments"]))
        return {
            "tool_call_id": "tc_" + tool_name,
            "query_id": "query_" + tool_name,
            "status": "success",
            "summary": f"{tool_name} success",
            "data": {
                "tool_name": tool_name,
                "uid": kwargs["arguments"].get("uid"),
                "records": [{"private_raw_row": "should_not_enter_llm_context"}],
            },
        }


class MockEvidenceGateway(FakeGateway):
    def invoke_tool(self, tool_name: str, **kwargs: Any) -> dict[str, Any]:
        self.invocations.append((tool_name, kwargs["arguments"]))
        return {
            "tool_call_id": "tc_" + tool_name,
            "query_id": "query_" + tool_name,
            "status": "success",
            "summary": f"{tool_name} mock evidence success",
            "data": {"source": "mock_adapter", "uid": kwargs["arguments"].get("uid")},
        }


class CapturingLLM:
    is_real = False

    def __init__(self) -> None:
        self.summary_observations: list[dict[str, Any]] = []

    def classify_and_extract(self, text: str) -> LLMResult:
        return LLMResult({}, "local_rules", "rules-v1")

    def summarize(self, case: dict[str, Any], observations: list[dict[str, Any]]) -> LLMResult:
        self.summary_observations = observations
        return LLMResult({"summary": "LLM saw compact evidence only", "confidence": 0.81}, "capture", "test")


class RecordingDecisionEngine:
    def __init__(self) -> None:
        self.requests: list[DecisionRequest] = []

    def plan(self, request: DecisionRequest) -> DecisionResponse:
        self.requests.append(request)
        return DecisionResponse(
            action="ask_user",
            reason="unit test decision engine asked for uid",
            missing_fields=["user_id_or_uid"],
            verification=VerificationReport(accepted=True, reason="unit test"),
        )


class AgentPlatformFastAPITest(unittest.TestCase):
    def setUp(self) -> None:
        web_file = Path(tempfile.gettempdir()) / "agent-platform-test.html"
        web_file.write_text("<html>ok</html>", encoding="utf-8")
        self.config = Config(
            host="127.0.0.1",
            port=19091,
            db_driver="memory",
            mysql=None,
            gateway_endpoint="http://gateway.test",
            gateway_bearer_token="",
            gateway_admin_bearer_token="",
            gateway_agent_id="business-troubleshooter-v1",
            max_tool_calls_per_case=10,
            max_tool_failures_per_case=3,
            max_investigation_seconds=120,
            web_asset_path=web_file,
            llm=LLMConfig("local_rules", "", "", "rules-v1", 30, False),
            vision=VisionConfig("local_rules", "", "", "local-vision-placeholder", 30, 3, 10 * 1024 * 1024),
            chat_platform=ChatPlatformConfig("lark", "https://open.larksuite.com", "", "", "", "", (), 3, 10 * 1024 * 1024),
        )
        self.repo = MemoryRepository()
        self.gateway = FakeGateway()
        self.platform = AgentPlatform(self.config, self.repo, gateway=self.gateway)
        self.client_ctx = TestClient(create_app(platform=self.platform))
        self.client = self.client_ctx.__enter__()

    def tearDown(self) -> None:
        self.client_ctx.__exit__(None, None, None)

    def test_health_food_web_chat_runs_python_orchestrator_and_gateway_tools(self) -> None:
        response = self.client.post(
            "/web/api/chat",
            data={"message": "health-food uid hf-user-001 今日没有每日推荐", "async": "0"},
        )

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        self.assertEqual(body["case"]["issue_domain"], "health_food")
        self.assertEqual(body["case"]["status"], "NEED_HUMAN_CONFIRMATION")
        self.assertIn("get_health_food_recommendation_status", [name for name, _ in self.gateway.invocations])
        decision_types = [item["decision_type"] for item in body["ai_decision_logs"]]
        self.assertIn("orchestrator_plan", decision_types)
        self.assertIn("tool_invocation", decision_types)

    def test_greeting_asks_for_problem_without_knowledge_or_gateway(self) -> None:
        self.repo.upsert_knowledge(
            {
                "title": "不应命中的平台经验",
                "issue_domain": "health_food",
                "issue_type": "每日推荐缺失",
                "typical_description": "如果问候语命中这个经验就是错误。",
                "recommended_steps": [],
                "common_causes": [],
                "useful_tools": [],
                "confidence": 0.99,
                "observed_case_count": 9,
            }
        )

        response = self.client.post("/web/api/chat", data={"message": "你好", "async": "0"})

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        self.assertEqual(body["case"]["status"], "WAITING_USER_REPLY")
        self.assertIn("请描述具体生产问题", body["reply"])
        self.assertEqual(self.gateway.invocations, [])
        self.assertIn("intake_agent", [item["agent_name"] for item in body["agent_runs"]])
        self.assertNotIn("平台经验命中", body["reply"])

    def test_mock_gateway_and_local_rules_are_disclosed_in_reply(self) -> None:
        platform = AgentPlatform(self.config, MemoryRepository(), gateway=MockEvidenceGateway())
        with TestClient(create_app(platform=platform)) as client:
            response = client.post(
                "/web/api/chat",
                data={"message": "health-food uid hf-mock-boundary 今日 token 消耗数量不对", "async": "0"},
            )

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        self.assertIn("mock adapter", body["reply"])
        self.assertIn("未启用本地决策 Agent", body["reply"])
        self.assertIn("规则编排", body["reply"])

    def test_case_payload_contains_agent_runs_and_events(self) -> None:
        response = self.client.post(
            "/web/api/chat",
            data={"message": "health-food uid hf-user-runtime 今日没有每日推荐", "async": "0"},
        )

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        runs = body["agent_runs"]
        self.assertTrue(runs)
        agent_names = [item["agent_name"] for item in runs]
        self.assertIn("supervisor", agent_names)
        self.assertIn("knowledge_agent", agent_names)
        self.assertIn("health_food_agent", agent_names)
        self.assertIn("verifier", agent_names)
        supervisor = next(item for item in runs if item["agent_name"] == "supervisor")
        self.assertEqual(supervisor["status"], "completed")
        self.assertIn("orchestrator_plan", [item["event_type"] for item in supervisor["events"]])
        self.assertNotIn("private_raw_row", json.dumps(runs, ensure_ascii=False))

    def test_agent_runtime_register_and_heartbeat(self) -> None:
        register = self.client.post(
            "/web/api/agent-runtimes/register",
            json={
                "runtime_id": "local-mac-codex",
                "runtime_name": "Local Mac Codex",
                "runtime_type": "local",
                "provider_list": ["codex", "claude"],
                "workspace_root": "/tmp/ai-troubleshooter-runtime",
            },
        )

        self.assertEqual(register.status_code, 201, register.text)
        self.assertEqual(register.json()["runtime_id"], "local-mac-codex")
        self.assertEqual(register.json()["provider_list"], ["codex", "claude"])

        heartbeat = self.client.post("/web/api/agent-runtimes/local-mac-codex/heartbeat", json={"status": "online"})
        self.assertEqual(heartbeat.status_code, 200, heartbeat.text)
        self.assertEqual(heartbeat.json()["status"], "online")

        listed = self.client.get("/web/api/agent-runtimes")
        self.assertEqual(listed.status_code, 200, listed.text)
        self.assertEqual(listed.json()["items"][0]["runtime_id"], "local-mac-codex")

    def test_local_agent_discovery_registers_runtime(self) -> None:
        discovered = [
            {
                "provider_id": "claude_code",
                "display_name": "Claude Code",
                "kind": "coding_agent",
                "installed": True,
                "llm_capable": True,
                "version": "1.0.0",
                "enabled": False,
            },
            {
                "provider_id": "cursor",
                "display_name": "Cursor",
                "kind": "editor",
                "installed": True,
                "llm_capable": False,
                "enabled": False,
            },
        ]

        with mock.patch("agent_platform.service.discover_local_agents", return_value=discovered):
            response = self.client.get("/web/api/local-agents/discover")

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        self.assertEqual([item["provider_id"] for item in body["providers"]], ["claude_code", "cursor"])
        self.assertEqual(body["runtime"]["runtime_type"], "local")
        runtime = next(iter(self.repo.runtimes.values()))
        self.assertEqual(runtime["provider_list"][0]["provider_id"], "claude_code")
        self.assertFalse(runtime["provider_list"][0]["enabled"])

    def test_local_agent_enable_requires_non_interactive_llm_provider(self) -> None:
        discovered = [
            {
                "provider_id": "claude_code",
                "display_name": "Claude Code",
                "kind": "coding_agent",
                "installed": True,
                "llm_capable": True,
                "enabled": False,
            },
            {
                "provider_id": "cursor",
                "display_name": "Cursor",
                "kind": "editor",
                "installed": True,
                "llm_capable": False,
                "enabled": False,
            },
        ]

        with mock.patch("agent_platform.service.discover_local_agents", return_value=discovered):
            enabled = self.client.post("/web/api/local-agents/enable", json={"provider_id": "claude_code"})
            rejected = self.client.post("/web/api/local-agents/enable", json={"provider_id": "cursor"})

        self.assertEqual(enabled.status_code, 200, enabled.text)
        self.assertTrue(enabled.json()["provider"]["enabled"])
        self.assertEqual(rejected.status_code, 400, rejected.text)
        self.assertIn("not non-interactive LLM capable", rejected.json()["error"])

    def test_local_agent_enable_keeps_one_decision_provider_active(self) -> None:
        discovered = [
            {
                "provider_id": "claude_code",
                "display_name": "Claude Code",
                "kind": "coding_agent",
                "installed": True,
                "llm_capable": True,
                "enabled": False,
            },
            {
                "provider_id": "codex",
                "display_name": "Codex CLI",
                "kind": "coding_agent",
                "installed": True,
                "llm_capable": True,
                "enabled": False,
            },
        ]

        with mock.patch("agent_platform.service.discover_local_agents", return_value=discovered):
            first = self.client.post("/web/api/local-agents/enable", json={"provider_id": "claude_code"})
            second = self.client.post("/web/api/local-agents/enable", json={"provider_id": "codex"})

        self.assertEqual(first.status_code, 200, first.text)
        self.assertEqual(second.status_code, 200, second.text)
        providers = next(iter(self.repo.runtimes.values()))["provider_list"]
        states = {item["provider_id"]: item["enabled"] for item in providers}
        self.assertEqual(states, {"claude_code": False, "codex": True})

    def test_local_agent_llm_provider_uses_noninteractive_command(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            script = Path(tmp) / "fake-local-agent"
            script.write_text(
                "#!/usr/bin/env bash\n"
                "cat >/dev/null\n"
                "printf '%s' '{\"issue_domain\":\"health_food\",\"entities\":{\"uid\":\"hf-local\"}}'\n",
                encoding="utf-8",
            )
            script.chmod(0o700)
            config = LLMConfig("local_agent", "", "", "custom", 5, False)

            with mock.patch.dict(os.environ, {"LOCAL_AGENT_COMMAND": str(script)}, clear=False):
                result = LLMClient(config).classify_and_extract("health-food uid hf-local 今日推荐缺失")

        self.assertEqual(result.provider, "local_agent")
        self.assertEqual(result.model, "custom")
        self.assertEqual(result.payload["issue_domain"], "health_food")
        self.assertEqual(result.payload["entities"]["uid"], "hf-local")

    def test_web_enabled_codex_agent_drives_decision_advisor_without_restart(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            script = Path(tmp) / "fake-codex-advisor"
            script.write_text(
                "#!/usr/bin/env bash\n"
                "cat >/dev/null\n"
                "printf '%s' '{\"action\":\"invoke_tools\",\"reason\":\"Codex local advisor selected quota evidence first\",\"confidence\":0.82,\"selected_tools\":[\"get_health_food_ai_quota\"]}'\n",
                encoding="utf-8",
            )
            script.chmod(0o700)
            discovered = [
                {
                    "provider_id": "codex",
                    "display_name": "Codex CLI",
                    "kind": "coding_agent",
                    "installed": True,
                    "llm_capable": True,
                    "version": "codex-cli-test",
                    "enabled": False,
                }
            ]

            with (
                mock.patch("agent_platform.service.discover_local_agents", return_value=discovered),
                mock.patch.dict(os.environ, {"LOCAL_AGENT_COMMAND": str(script)}, clear=False),
            ):
                enabled = self.client.post("/web/api/local-agents/enable", json={"provider_id": "codex"})
                response = self.client.post(
                    "/web/api/chat",
                    data={"message": "health-food uid hf-web-codex 今日 token 消耗数量不对", "async": "0"},
                )

        self.assertEqual(enabled.status_code, 200, enabled.text)
        self.assertTrue(enabled.json()["provider"]["enabled"])
        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        self.assertIn("llm_decision_agent", [item["agent_name"] for item in body["agent_runs"]])
        self.assertEqual([name for name, _ in self.gateway.invocations], ["get_health_food_ai_quota"])
        advisor_run = next(item for item in body["agent_runs"] if item["agent_name"] == "llm_decision_agent")
        self.assertEqual(advisor_run["model_provider"], "local_agent")
        self.assertEqual(advisor_run["model_name"], "codex")
        self.assertIn("enabled_local_agent", json.dumps(advisor_run, ensure_ascii=False))
        self.assertIn("codex", json.dumps(advisor_run, ensure_ascii=False))

    def test_web_chat_must_call_decision_engine_plan(self) -> None:
        decision_engine = RecordingDecisionEngine()
        platform = AgentPlatform(self.config, MemoryRepository(), gateway=FakeGateway(), decision_engine=decision_engine)

        result = platform.submit_chat(message="health-food 今日没有每日推荐", async_process=False)

        self.assertEqual(len(decision_engine.requests), 1)
        request = decision_engine.requests[0]
        self.assertEqual(request.case.issue_domain, "health_food")
        self.assertEqual([tool.name for tool in request.available_tools], [item["name"] for item in FakeGateway().list_tools()])
        self.assertIn("业务 uid", result["reply"])

    def test_context_ledger_records_agent_reports_tool_evidence_and_summary(self) -> None:
        response = self.client.post(
            "/web/api/chat",
            data={"message": "health-food uid hf-user-ledger 今日没有每日推荐", "async": "0"},
        )

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        ledger_types = [item["ledger_type"] for item in body["context_ledger"]]
        self.assertIn("case_state", ledger_types)
        self.assertIn("gateway_tools", ledger_types)
        self.assertIn("knowledge_retrieval", ledger_types)
        self.assertIn("agent_report", ledger_types)
        self.assertIn("tool_evidence", ledger_types)
        self.assertIn("final_summary", ledger_types)
        tool_evidence = [item for item in body["context_ledger"] if item["ledger_type"] == "tool_evidence"]
        self.assertTrue(tool_evidence)
        self.assertIn("gateway_tool_call", {ref["ref_type"] for item in tool_evidence for ref in item["evidence_refs"]})
        self.assertNotIn("private_raw_row", json.dumps(tool_evidence, ensure_ascii=False))
        self.assertNotIn('"data"', json.dumps(tool_evidence, ensure_ascii=False))

    def test_llm_summary_receives_compact_evidence_not_raw_tool_data(self) -> None:
        llm = CapturingLLM()
        platform = AgentPlatform(self.config, MemoryRepository(), gateway=FakeGateway(), llm_client=llm)

        result = platform.submit_chat(message="health-food uid hf-user-compact 今日 token 消耗数量不对", async_process=False)

        self.assertIn("LLM saw compact evidence only", result["reply"])
        self.assertTrue(llm.summary_observations)
        for observation in llm.summary_observations:
            self.assertNotIn("data", observation)
            self.assertIn("evidence_refs", observation)
        self.assertNotIn("private_raw_row", json.dumps(llm.summary_observations, ensure_ascii=False))

    def test_repository_json_serializes_database_scalar_types(self) -> None:
        encoded = _json_or_none({"confidence": Decimal("0.8000"), "created_at": datetime(2026, 5, 24, 10, 0, 0)})

        self.assertIn('"0.8000"', encoded or "")
        self.assertIn('"2026-05-24 10:00:00"', encoded or "")

    def test_qwen_profile_reads_spring_ai_config_and_defaults_qwen_vision(self) -> None:
        model_file = Path(tempfile.gettempdir()) / "agent-platform-qwen-model.yml"
        model_file.write_text(
            """
spring:
  ai:
    qwen:
      api-key: unit-test-dashscope-key
      base-url-http: https://dashscope.example/compatible-mode/v1
      chat:
        options:
          model: qwen3.6-flash
""",
            encoding="utf-8",
        )

        with mock.patch.dict(
            "os.environ",
            {"AI_MODEL_PROFILE": "qwen", "AI_MODEL_CONFIG_FILE": str(model_file), "DB_DRIVER": "memory"},
            clear=True,
        ):
            cfg = load_config()

        self.assertEqual(cfg.llm.provider, "openai_compatible")
        self.assertEqual(cfg.llm.base_url, "https://dashscope.example/compatible-mode/v1")
        self.assertEqual(cfg.llm.api_key, "unit-test-dashscope-key")
        self.assertEqual(cfg.llm.model, "qwen3.6-flash")
        self.assertEqual(cfg.vision.provider, "qwen_openai_compatible")
        self.assertEqual(cfg.vision.base_url, "https://dashscope.example/compatible-mode/v1")
        self.assertEqual(cfg.vision.api_key, "unit-test-dashscope-key")
        self.assertEqual(cfg.vision.model, "qwen-vl-plus")

    def test_gpt_profile_uses_openai_key_and_gpt_vision_model(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "AI_MODEL_PROFILE": "gpt",
                "OPENAI_API_KEY": "unit-test-openai-key",
                "OPENAI_MODEL": "gpt-4.1-mini",
                "DB_DRIVER": "memory",
            },
            clear=True,
        ):
            cfg = load_config()

        self.assertEqual(cfg.llm.provider, "openai")
        self.assertEqual(cfg.llm.base_url, "https://api.openai.com/v1")
        self.assertEqual(cfg.llm.api_key, "unit-test-openai-key")
        self.assertEqual(cfg.llm.model, "gpt-4.1-mini")
        self.assertEqual(cfg.vision.provider, "openai")
        self.assertEqual(cfg.vision.base_url, "https://api.openai.com/v1")
        self.assertEqual(cfg.vision.api_key, "unit-test-openai-key")
        self.assertEqual(cfg.vision.model, "gpt-4.1-mini")

    def test_local_mysql_rejects_noncanonical_schema_from_dsn(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "DB_DRIVER": "mysql",
                "DB_DSN": "root:unit-test@tcp(127.0.0.1:3306)/ai_troubleshooter_itest?parseTime=true",
            },
            clear=True,
        ):
            with self.assertRaisesRegex(RuntimeError, "local MySQL platform database"):
                load_config()

    def test_local_mysql_rejects_noncanonical_localhost_schema_from_dsn(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "DB_DRIVER": "mysql",
                "DB_DSN": "root:unit-test@tcp(localhost:3306)/ai_troubleshooter_itest?parseTime=true",
            },
            clear=True,
        ):
            with self.assertRaisesRegex(RuntimeError, "local MySQL platform database"):
                load_config()

    def test_local_mysql_rejects_noncanonical_ipv6_schema_from_dsn(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "DB_DRIVER": "mysql",
                "DB_DSN": "root:unit-test@tcp([::1]:3306)/ai_troubleshooter_itest?parseTime=true",
            },
            clear=True,
        ):
            with self.assertRaisesRegex(RuntimeError, "local MySQL platform database"):
                load_config()

    def test_local_mysql_allows_explicit_noncanonical_schema(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "DB_DRIVER": "mysql",
                "DB_DSN": "root:unit-test@tcp(127.0.0.1:3306)/ai_troubleshooter_itest?parseTime=true",
                "ALLOW_NON_CANONICAL_LOCAL_DB": "true",
            },
            clear=True,
        ):
            cfg = load_config()

        self.assertIsNotNone(cfg.mysql)
        self.assertEqual(cfg.mysql.database if cfg.mysql else "", "ai_troubleshooter_itest")

    def test_explicit_qwen_vision_provider_reuses_dashscope_env(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "AI_MODEL_PROFILE": "local_rules",
                "VISION_PROVIDER": "qwen_openai_compatible",
                "DASHSCOPE_API_KEY": "unit-test-dashscope-key",
                "DB_DRIVER": "memory",
            },
            clear=True,
        ):
            cfg = load_config()

        self.assertEqual(cfg.vision.provider, "qwen_openai_compatible")
        self.assertEqual(cfg.vision.base_url, "https://dashscope.aliyuncs.com/compatible-mode/v1")
        self.assertEqual(cfg.vision.api_key, "unit-test-dashscope-key")
        self.assertEqual(cfg.vision.model, "qwen-vl-plus")

    def test_explicit_openai_vision_provider_reuses_openai_env(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {
                "AI_MODEL_PROFILE": "local_rules",
                "VISION_PROVIDER": "openai",
                "OPENAI_API_KEY": "unit-test-openai-key",
                "DB_DRIVER": "memory",
            },
            clear=True,
        ):
            cfg = load_config()

        self.assertEqual(cfg.vision.provider, "openai")
        self.assertEqual(cfg.vision.base_url, "https://api.openai.com/v1")
        self.assertEqual(cfg.vision.api_key, "unit-test-openai-key")
        self.assertEqual(cfg.vision.model, "gpt-4.1-mini")

    def test_missing_health_food_uid_asks_user_without_gateway_call(self) -> None:
        response = self.client.post("/web/api/chat", data={"message": "health-food 今日没有每日推荐", "async": "0"})

        self.assertEqual(response.status_code, 200, response.text)
        body = response.json()
        self.assertEqual(body["case"]["status"], "WAITING_USER_REPLY")
        self.assertIn("业务 uid", body["reply"])
        self.assertEqual(self.gateway.invocations, [])

    def test_lark_event_is_python_agent_platform_entry(self) -> None:
        response = self.client.post(
            "/lark/events",
            json={
                "chat_id": "oc_dev",
                "thread_id": "thread_dev",
                "message_id": "msg_dev",
                "user_id": "ou_dev",
                "text": "health-food uid hf-user-002 今日 token 消耗数量不对",
            },
        )

        self.assertEqual(response.status_code, 202, response.text)
        body = response.json()
        self.assertEqual(body["case"]["source"], "lark")
        self.assertTrue(body["processing"])

    def test_lark_event_is_idempotent_by_message_id(self) -> None:
        payload = {
            "chat_id": "oc_dev",
            "thread_id": "thread_dev",
            "message_id": "msg_same",
            "user_id": "ou_dev",
            "text": "health-food uid hf-user-002 今日 token 消耗数量不对",
        }

        first = self.client.post("/lark/events", json=payload)
        second = self.client.post("/lark/events", json=payload)

        self.assertEqual(first.status_code, 202, first.text)
        self.assertEqual(second.status_code, 202, second.text)
        self.assertTrue(second.json()["duplicate"])
        self.assertEqual(len(self.repo.cases), 1)

    def test_lark_encrypted_challenge_is_verified_in_python(self) -> None:
        secure = replace(
            self.config,
            chat_platform=ChatPlatformConfig("lark", "https://open.larksuite.com", "", "", "token_1", "encrypt_key_1", ("oc_dev",), 3, 10 * 1024 * 1024),
        )
        platform = AgentPlatform(secure, self.repo, gateway=self.gateway)
        with TestClient(create_app(platform=platform)) as client:
            response = client.post("/lark/events", content=_encrypted_envelope("encrypt_key_1", {"token": "token_1", "challenge": "challenge_1"}))

        self.assertEqual(response.status_code, 200, response.text)
        self.assertEqual(response.json()["challenge"], "challenge_1")

    def test_lark_plain_payload_is_rejected_when_encrypt_key_is_configured(self) -> None:
        secure = replace(
            self.config,
            chat_platform=ChatPlatformConfig("lark", "https://open.larksuite.com", "", "", "token_1", "encrypt_key_1", ("oc_dev",), 3, 10 * 1024 * 1024),
        )
        platform = AgentPlatform(secure, self.repo, gateway=self.gateway)
        with TestClient(create_app(platform=platform)) as client:
            response = client.post("/lark/events", json={"token": "token_1", "challenge": "challenge_1"})

        self.assertEqual(response.status_code, 400, response.text)
        self.assertIn("not encrypted", response.text)

    def test_lark_chat_allowlist_rejects_unknown_chat(self) -> None:
        secure = replace(
            self.config,
            chat_platform=ChatPlatformConfig("lark", "https://open.larksuite.com", "", "", "token_1", "", ("oc_allowed",), 3, 10 * 1024 * 1024),
        )
        platform = AgentPlatform(secure, self.repo, gateway=self.gateway)
        with TestClient(create_app(platform=platform)) as client:
            response = client.post(
                "/lark/events",
                json={"token": "token_1", "chat_id": "oc_other", "message_id": "msg_dev", "user_id": "ou_dev", "text": "health-food uid hf-user-002 今日 token 消耗数量不对"},
            )

        self.assertEqual(response.status_code, 403, response.text)

    def test_capability_import_rejects_dangerous_write_tool(self) -> None:
        manifest = """
service:
  service_name: bad-service
  base_url: http://127.0.0.1:18080
capabilities:
  - tool_name: delete_user
    description: delete user
    method: POST
    path: /v1/readonly/bad/delete-user
    required_params: [uid]
"""
        response = self.client.post(
            "/web/api/capabilities/import",
            json={"raw_config": manifest, "service_name": "bad-service", "base_url": "http://127.0.0.1:18080"},
        )

        self.assertEqual(response.status_code, 201, response.text)
        item = response.json()["capabilities"][0]
        self.assertEqual(item["safety_status"], "rejected")
        self.assertEqual(item["tool_status"], "rejected")

def _encrypted_envelope(encrypt_key: str, payload: dict[str, Any]) -> bytes:
    plaintext = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    key = hashlib.sha256(encrypt_key.encode("utf-8")).digest()
    padding = 16 - len(plaintext) % 16
    padded = plaintext + bytes([padding]) * padding
    encryptor = Cipher(algorithms.AES(key), modes.CBC(key[:16])).encryptor()
    ciphertext = encryptor.update(padded) + encryptor.finalize()
    return json.dumps({"encrypt": base64.b64encode(ciphertext).decode("ascii")}).encode("utf-8")


if __name__ == "__main__":
    unittest.main()
