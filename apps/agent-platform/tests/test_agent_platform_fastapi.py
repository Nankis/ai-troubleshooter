from __future__ import annotations

import itertools
import base64
import hashlib
import json
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

from agent_platform.config import ChatPlatformConfig, Config, LLMConfig, VisionConfig, load_config
from agent_platform.gateway import GatewayHTTPClient
from agent_platform.llm import LLMResult
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
        self.knowledge_ids = itertools.count(1)
        self.capability_ids = itertools.count(1)
        self.cases: dict[int, dict[str, Any]] = {}
        self.messages: dict[int, list[dict[str, Any]]] = {}
        self.entities: dict[int, list[dict[str, Any]]] = {}
        self.investigations: dict[int, dict[str, Any]] = {}
        self.decisions: dict[int, list[dict[str, Any]]] = {}
        self.context_ledger: dict[int, list[dict[str, Any]]] = {}
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


class CapturingLLM:
    is_real = False

    def __init__(self) -> None:
        self.summary_observations: list[dict[str, Any]] = []

    def classify_and_extract(self, text: str) -> LLMResult:
        return LLMResult({}, "local_rules", "rules-v1")

    def summarize(self, case: dict[str, Any], observations: list[dict[str, Any]]) -> LLMResult:
        self.summary_observations = observations
        return LLMResult({"summary": "LLM saw compact evidence only", "confidence": 0.81}, "capture", "test")


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
