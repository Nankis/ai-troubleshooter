from __future__ import annotations

import json
import uuid
from contextlib import contextmanager
from datetime import datetime
from typing import Any, Iterator, Protocol

import pymysql
from pymysql.cursors import DictCursor

from .config import MySQLConfig


class Repository(Protocol):
    def close(self) -> None: ...
    def create_case(self, data: dict[str, Any]) -> dict[str, Any]: ...
    def find_case_by_message_id(self, source: str, message_id: str) -> dict[str, Any] | None: ...
    def get_case_by_no(self, case_no: str) -> dict[str, Any] | None: ...
    def get_case_by_id(self, case_id: int) -> dict[str, Any] | None: ...
    def list_recent_cases(self, limit: int = 30) -> list[dict[str, Any]]: ...
    def update_case_fields(self, case_id: int, fields: dict[str, Any]) -> dict[str, Any]: ...
    def delete_case(self, case_id: int) -> None: ...
    def add_message(self, case_id: int, role: str, content: str, content_type: str = "text") -> dict[str, Any]: ...
    def list_messages(self, case_id: int) -> list[dict[str, Any]]: ...
    def add_entities(self, case_id: int, entities: list[dict[str, Any]]) -> None: ...
    def list_entities(self, case_id: int) -> list[dict[str, Any]]: ...
    def create_investigation(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def finish_investigation(self, investigation_id: int, status: str, summary: str, confidence: float | None) -> dict[str, Any]: ...
    def add_decision_log(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def list_decision_logs(self, case_id: int, limit: int = 100) -> list[dict[str, Any]]: ...
    def add_context_ledger(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def list_context_ledger(self, case_id: int, limit: int = 100, ledger_type: str = "") -> list[dict[str, Any]]: ...
    def register_agent_runtime(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def heartbeat_agent_runtime(self, runtime_id: str, status: str = "online") -> dict[str, Any]: ...
    def list_agent_runtimes(self, limit: int = 50, status: str = "") -> list[dict[str, Any]]: ...
    def create_agent_run(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def update_agent_run(self, run_id: int, fields: dict[str, Any]) -> dict[str, Any]: ...
    def add_agent_run_event(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def list_agent_runs(self, case_id: int, limit: int = 100) -> list[dict[str, Any]]: ...
    def list_agent_run_events(self, run_id: int, limit: int = 200) -> list[dict[str, Any]]: ...
    def list_knowledge(self, limit: int = 30, issue_domain: str = "", issue_type: str = "", status: str = "") -> list[dict[str, Any]]: ...
    def get_knowledge(self, knowledge_id: int) -> dict[str, Any] | None: ...
    def upsert_knowledge(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def delete_knowledge(self, knowledge_id: int) -> None: ...
    def list_capabilities(self, limit: int = 200, status: str = "", source_type: str = "") -> list[dict[str, Any]]: ...
    def upsert_business_service(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def upsert_tool_capability(self, item: dict[str, Any]) -> dict[str, Any]: ...
    def update_tool_capability_status(self, capability_id: int, status: str, published_by: str) -> dict[str, Any]: ...


class MySQLRepository:
    def __init__(self, config: MySQLConfig) -> None:
        self._config = config
        self._pool_kwargs = {
            "host": config.host,
            "port": config.port,
            "user": config.user,
            "password": config.password,
            "database": config.database,
            "charset": config.charset,
            "autocommit": True,
            "cursorclass": DictCursor,
        }
        with self._connect() as conn:
            conn.ping(reconnect=True)

    def close(self) -> None:
        return

    @contextmanager
    def _connect(self) -> Iterator[pymysql.connections.Connection]:
        conn = pymysql.connect(**self._pool_kwargs)
        try:
            yield conn
        finally:
            conn.close()

    def create_case(self, data: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        tmp_no = f"case_pending_{now.timestamp_ns() if hasattr(now, 'timestamp_ns') else uuid.uuid4().hex}"
        source = data.get("source") or "web"
        with self._connect() as conn, conn.cursor() as cur:
            try:
                cur.execute(
                    """
                    INSERT INTO tb_troubleshoot_case
                    (case_no, case_title, uid, source, chat_id, thread_id, message_id, reporter_user_id,
                     original_text, ocr_text, case_status, priority, timezone, create_time, update_time, version)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 0)
                    """,
                    (
                        tmp_no,
                        _none_if_empty(data.get("title")),
                        data.get("uid") or data.get("reporter_user_id") or "",
                        source,
                        _none_if_empty(data.get("chat_id")),
                        _none_if_empty(data.get("thread_id")),
                        _none_if_empty(data.get("message_id")),
                        _none_if_empty(data.get("reporter_user_id")),
                        _none_if_empty(data.get("original_text")),
                        _none_if_empty(data.get("ocr_text")),
                        "NEW",
                        data.get("priority") or "normal",
                        data.get("timezone") or "Asia/Shanghai",
                        now,
                        now,
                    ),
                )
            except pymysql.err.IntegrityError:
                existing = self.find_case_by_message_id(source, str(data.get("message_id") or ""))
                if existing is not None:
                    return existing
                raise
            case_id = int(cur.lastrowid)
            case_no = f"case_{now.strftime('%Y%m%d')}_{case_id:06d}"
            cur.execute("UPDATE tb_troubleshoot_case SET case_no = %s WHERE id = %s", (case_no, case_id))
        found = self.get_case_by_id(case_id)
        if found is None:
            raise RuntimeError("created case not found")
        return found

    def get_case_by_no(self, case_no: str) -> dict[str, Any] | None:
        return self._fetch_one(_case_select() + " WHERE case_no = %s AND status = 1", (case_no,))

    def get_case_by_id(self, case_id: int) -> dict[str, Any] | None:
        return self._fetch_one(_case_select() + " WHERE id = %s AND status = 1", (case_id,))

    def find_case_by_message_id(self, source: str, message_id: str) -> dict[str, Any] | None:
        if not source or not message_id:
            return None
        return self._fetch_one(_case_select() + " WHERE source = %s AND message_id = %s AND status = 1", (source, message_id))

    def list_recent_cases(self, limit: int = 30) -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 100, 30)
        return self._fetch_all(_case_select() + " WHERE status = 1 ORDER BY update_time DESC LIMIT %s", (limit,))

    def update_case_fields(self, case_id: int, fields: dict[str, Any]) -> dict[str, Any]:
        allowed = {
            "case_title",
            "uid",
            "original_text",
            "ocr_text",
            "issue_domain",
            "issue_type",
            "case_status",
            "priority",
            "timezone",
            "closed_at",
        }
        updates = {key: value for key, value in fields.items() if key in allowed}
        if not updates:
            found = self.get_case_by_id(case_id)
            if found is None:
                raise KeyError("case not found")
            return found
        assignments = [f"{key} = %s" for key in updates]
        args = [_none_if_empty(value) for value in updates.values()]
        args.extend([datetime.now(), case_id])
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                f"UPDATE tb_troubleshoot_case SET {', '.join(assignments)}, update_time = %s, version = version + 1 WHERE id = %s AND status = 1",
                tuple(args),
            )
        found = self.get_case_by_id(case_id)
        if found is None:
            raise KeyError("case not found")
        return found

    def delete_case(self, case_id: int) -> None:
        self._execute("UPDATE tb_troubleshoot_case SET status = 0, update_time = %s WHERE id = %s AND status = 1", (datetime.now(), case_id))

    def add_message(self, case_id: int, role: str, content: str, content_type: str = "text") -> dict[str, Any]:
        now = datetime.now()
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO tb_troubleshoot_case_message
                (case_id, role, content, content_type, create_time)
                VALUES (%s, %s, %s, %s, %s)
                """,
                (case_id, role, content, content_type or "text", now),
            )
            msg_id = int(cur.lastrowid)
        return {
            "id": msg_id,
            "case_id": case_id,
            "role": role,
            "content": content,
            "content_type": content_type or "text",
            "created_at": now,
        }

    def list_messages(self, case_id: int) -> list[dict[str, Any]]:
        return self._fetch_all(
            """
            SELECT id, case_id, role, platform_message_id, content, content_type, create_time AS created_at
            FROM tb_troubleshoot_case_message
            WHERE case_id = ? ORDER BY id
            """.replace("?", "%s"),
            (case_id,),
        )

    def add_entities(self, case_id: int, entities: list[dict[str, Any]]) -> None:
        now = datetime.now()
        with self._connect() as conn, conn.cursor() as cur:
            for entity in entities:
                entity_type = str(entity.get("entity_type") or entity.get("type") or "").strip()
                entity_value = str(entity.get("entity_value") or entity.get("value") or "").strip()
                if not entity_type or not entity_value:
                    continue
                cur.execute(
                    """
                    SELECT COUNT(1) AS count
                    FROM tb_troubleshoot_case_entity
                    WHERE case_id = %s AND entity_type = %s AND entity_value = %s AND status = 1
                    """,
                    (case_id, entity_type, entity_value),
                )
                if int(cur.fetchone()["count"]) > 0:
                    continue
                cur.execute(
                    """
                    INSERT INTO tb_troubleshoot_case_entity
                    (case_id, entity_type, entity_value, source, confidence, create_time)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    """,
                    (case_id, entity_type, entity_value, entity.get("source") or "agent-platform", entity.get("confidence"), now),
                )

    def list_entities(self, case_id: int) -> list[dict[str, Any]]:
        return self._fetch_all(
            """
            SELECT id, case_id, entity_type, entity_value, source, confidence, create_time AS created_at
            FROM tb_troubleshoot_case_entity
            WHERE case_id = %s AND status = 1 ORDER BY id
            """,
            (case_id,),
        )

    def create_investigation(self, item: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        tmp_no = f"inv_pending_{uuid.uuid4().hex}"
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO tb_troubleshoot_investigation
                (investigation_no, case_id, agent_id, agent_version, model_provider, model_name,
                 investigation_status, initial_hypothesis, started_at, create_time, update_time)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    tmp_no,
                    item["case_id"],
                    item["agent_id"],
                    _none_if_empty(item.get("agent_version")),
                    _none_if_empty(item.get("model_provider")),
                    _none_if_empty(item.get("model_name")),
                    item.get("investigation_status") or "running",
                    _none_if_empty(item.get("initial_hypothesis")),
                    now,
                    now,
                    now,
                ),
            )
            investigation_id = int(cur.lastrowid)
            investigation_no = f"inv_{now.strftime('%Y%m%d')}_{investigation_id:06d}"
            cur.execute("UPDATE tb_troubleshoot_investigation SET investigation_no = %s WHERE id = %s", (investigation_no, investigation_id))
        return self._fetch_one("SELECT * FROM tb_troubleshoot_investigation WHERE id = %s", (investigation_id,)) or {}

    def finish_investigation(self, investigation_id: int, status: str, summary: str, confidence: float | None) -> dict[str, Any]:
        now = datetime.now()
        self._execute(
            """
            UPDATE tb_troubleshoot_investigation
            SET investigation_status = %s, final_summary = %s, confidence = %s, finished_at = %s, update_time = %s
            WHERE id = %s
            """,
            (status, summary, confidence, now, now, investigation_id),
        )
        return self._fetch_one("SELECT * FROM tb_troubleshoot_investigation WHERE id = %s", (investigation_id,)) or {}

    def add_decision_log(self, item: dict[str, Any]) -> dict[str, Any]:
        now = item.get("created_at") or datetime.now()
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO tb_troubleshoot_ai_decision_log
                (case_id, investigation_id, agent_id, decision_type, reason, input_snapshot_json, output_snapshot_json,
                 selected_tools_json, decision_status, latency_ms, error_message, create_time)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    item["case_id"],
                    item.get("investigation_id") or None,
                    item["agent_id"],
                    item["decision_type"],
                    _none_if_empty(item.get("reason")),
                    _json_or_none(item.get("input")),
                    _json_or_none(item.get("output")),
                    _json_or_none(item.get("selected_tools")),
                    item.get("status") or "success",
                    item.get("latency_ms") or 0,
                    _none_if_empty(item.get("error_message")),
                    now,
                ),
            )
            log_id = int(cur.lastrowid)
        return {"id": log_id, **item, "created_at": now}

    def list_decision_logs(self, case_id: int, limit: int = 100) -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 500, 100)
        rows = self._fetch_all(
            """
            SELECT id, case_id, investigation_id, agent_id, decision_type, reason, input_snapshot_json,
                   output_snapshot_json, selected_tools_json, decision_status AS status,
                   latency_ms, error_message, create_time AS created_at
            FROM tb_troubleshoot_ai_decision_log
            WHERE case_id = %s AND status = 1
            ORDER BY id DESC LIMIT %s
            """,
            (case_id, limit),
        )
        return list(reversed(rows))

    def add_context_ledger(self, item: dict[str, Any]) -> dict[str, Any]:
        now = item.get("created_at") or datetime.now()
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO tb_troubleshoot_context_ledger
                (case_id, ledger_type, ledger_key, summary, evidence_refs_json, payload_json, source_agent, create_time, update_time)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    item["case_id"],
                    item["ledger_type"],
                    item.get("ledger_key") or "",
                    _none_if_empty(item.get("summary")),
                    _json_or_none(item.get("evidence_refs")),
                    _json_or_none(item.get("payload")),
                    _none_if_empty(item.get("source_agent")),
                    now,
                    now,
                ),
            )
            ledger_id = int(cur.lastrowid)
        found = self._fetch_one(_context_ledger_select() + " WHERE id = %s AND status = 1", (ledger_id,))
        if found is None:
            raise RuntimeError("created context ledger not found")
        return _decode_context_ledger_row(found)

    def list_context_ledger(self, case_id: int, limit: int = 100, ledger_type: str = "") -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 500, 100)
        query = _context_ledger_select() + " WHERE case_id = %s AND status = 1"
        args: list[Any] = [case_id]
        if ledger_type:
            query += " AND ledger_type = %s"
            args.append(ledger_type)
        query += " ORDER BY id DESC LIMIT %s"
        args.append(limit)
        rows = self._fetch_all(query, tuple(args))
        return [_decode_context_ledger_row(row) for row in reversed(rows)]

    def register_agent_runtime(self, item: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        runtime_id = str(item.get("runtime_id") or "").strip()
        if not runtime_id:
            raise ValueError("runtime_id is required")
        self._execute(
            """
            INSERT INTO tb_troubleshoot_agent_runtime
            (runtime_id, runtime_name, runtime_type, host_name, provider_list_json, workspace_root,
             runtime_status, last_heartbeat_at, registered_at, create_time, update_time)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON DUPLICATE KEY UPDATE runtime_name=VALUES(runtime_name), runtime_type=VALUES(runtime_type),
            host_name=VALUES(host_name), provider_list_json=VALUES(provider_list_json), workspace_root=VALUES(workspace_root),
            runtime_status=VALUES(runtime_status), last_heartbeat_at=VALUES(last_heartbeat_at), update_time=VALUES(update_time)
            """,
            (
                runtime_id,
                item.get("runtime_name") or runtime_id,
                item.get("runtime_type") or "local",
                _none_if_empty(item.get("host_name")),
                _json_or_none(item.get("provider_list")),
                _none_if_empty(item.get("workspace_root")),
                item.get("runtime_status") or "registered",
                item.get("last_heartbeat_at") or now,
                item.get("registered_at") or now,
                now,
                now,
            ),
        )
        found = self._fetch_one(_agent_runtime_select() + " WHERE runtime_id = %s AND status = 1", (runtime_id,))
        if found is None:
            raise RuntimeError("registered runtime not found")
        return _decode_agent_runtime_row(found)

    def heartbeat_agent_runtime(self, runtime_id: str, status: str = "online") -> dict[str, Any]:
        now = datetime.now()
        self._execute(
            """
            UPDATE tb_troubleshoot_agent_runtime
            SET runtime_status = %s, last_heartbeat_at = %s, update_time = %s
            WHERE runtime_id = %s AND status = 1
            """,
            (status or "online", now, now, runtime_id),
        )
        found = self._fetch_one(_agent_runtime_select() + " WHERE runtime_id = %s AND status = 1", (runtime_id,))
        if found is None:
            raise KeyError("agent runtime not found")
        return _decode_agent_runtime_row(found)

    def list_agent_runtimes(self, limit: int = 50, status: str = "") -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 200, 50)
        query = _agent_runtime_select() + " WHERE status = 1"
        args: list[Any] = []
        if status:
            query += " AND runtime_status = %s"
            args.append(status)
        query += " ORDER BY update_time DESC LIMIT %s"
        args.append(limit)
        return [_decode_agent_runtime_row(row) for row in self._fetch_all(query, tuple(args))]

    def create_agent_run(self, item: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        tmp_no = f"run_pending_{uuid.uuid4().hex}"
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO tb_troubleshoot_agent_run
                (run_no, case_id, investigation_id, parent_run_id, runtime_id, agent_name, agent_role,
                 trigger_type, run_status, input_summary, output_summary, model_provider, model_name,
                 started_at, finished_at, latency_ms, error_message, payload_json, create_time, update_time)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    tmp_no,
                    item["case_id"],
                    item.get("investigation_id") or None,
                    item.get("parent_run_id") or None,
                    _none_if_empty(item.get("runtime_id")),
                    item["agent_name"],
                    item.get("agent_role") or "specialist",
                    item.get("trigger_type") or "case_process",
                    item.get("run_status") or "queued",
                    _none_if_empty(item.get("input_summary")),
                    _none_if_empty(item.get("output_summary")),
                    _none_if_empty(item.get("model_provider")),
                    _none_if_empty(item.get("model_name")),
                    item.get("started_at"),
                    item.get("finished_at"),
                    item.get("latency_ms") or None,
                    _none_if_empty(item.get("error_message")),
                    _json_or_none(item.get("payload")),
                    now,
                    now,
                ),
            )
            run_id = int(cur.lastrowid)
            run_no = f"run_{now.strftime('%Y%m%d')}_{run_id:06d}"
            cur.execute("UPDATE tb_troubleshoot_agent_run SET run_no = %s WHERE id = %s", (run_no, run_id))
        found = self._fetch_one(_agent_run_select() + " WHERE id = %s AND status = 1", (run_id,))
        if found is None:
            raise RuntimeError("created agent run not found")
        return _decode_agent_run_row(found)

    def update_agent_run(self, run_id: int, fields: dict[str, Any]) -> dict[str, Any]:
        allowed = {
            "investigation_id",
            "runtime_id",
            "run_status",
            "input_summary",
            "output_summary",
            "model_provider",
            "model_name",
            "started_at",
            "finished_at",
            "latency_ms",
            "error_message",
            "payload",
        }
        updates = {key: value for key, value in fields.items() if key in allowed}
        if not updates:
            found = self._fetch_one(_agent_run_select() + " WHERE id = %s AND status = 1", (run_id,))
            if found is None:
                raise KeyError("agent run not found")
            return _decode_agent_run_row(found)
        column_map = {"payload": "payload_json"}
        assignments = [f"{column_map.get(key, key)} = %s" for key in updates]
        args: list[Any] = []
        for key, value in updates.items():
            if key == "payload":
                args.append(_json_or_none(value))
            else:
                args.append(_none_if_empty(value))
        args.extend([datetime.now(), run_id])
        self._execute(
            f"UPDATE tb_troubleshoot_agent_run SET {', '.join(assignments)}, update_time = %s WHERE id = %s AND status = 1",
            tuple(args),
        )
        found = self._fetch_one(_agent_run_select() + " WHERE id = %s AND status = 1", (run_id,))
        if found is None:
            raise KeyError("agent run not found")
        return _decode_agent_run_row(found)

    def add_agent_run_event(self, item: dict[str, Any]) -> dict[str, Any]:
        now = item.get("created_at") or datetime.now()
        run_id = int(item["run_id"])
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                "SELECT COALESCE(MAX(event_seq), 0) + 1 AS next_seq FROM tb_troubleshoot_agent_run_event WHERE run_id = %s AND status = 1",
                (run_id,),
            )
            event_seq = int(cur.fetchone()["next_seq"])
            cur.execute(
                """
                INSERT INTO tb_troubleshoot_agent_run_event
                (run_id, event_seq, event_type, event_status, title, summary, payload_json, create_time, update_time)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    run_id,
                    event_seq,
                    item["event_type"],
                    item.get("event_status") or "info",
                    item["title"],
                    _none_if_empty(item.get("summary")),
                    _json_or_none(item.get("payload")),
                    now,
                    now,
                ),
            )
            event_id = int(cur.lastrowid)
        found = self._fetch_one(_agent_run_event_select() + " WHERE id = %s AND status = 1", (event_id,))
        if found is None:
            raise RuntimeError("created agent run event not found")
        return _decode_agent_run_event_row(found)

    def list_agent_runs(self, case_id: int, limit: int = 100) -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 500, 100)
        rows = self._fetch_all(
            _agent_run_select() + " WHERE case_id = %s AND status = 1 ORDER BY id DESC LIMIT %s",
            (case_id, limit),
        )
        return [_decode_agent_run_row(row) for row in reversed(rows)]

    def list_agent_run_events(self, run_id: int, limit: int = 200) -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 1000, 200)
        rows = self._fetch_all(
            _agent_run_event_select() + " WHERE run_id = %s AND status = 1 ORDER BY event_seq DESC LIMIT %s",
            (run_id, limit),
        )
        return [_decode_agent_run_event_row(row) for row in reversed(rows)]

    def list_knowledge(self, limit: int = 30, issue_domain: str = "", issue_type: str = "", status: str = "") -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 100, 30)
        query = _knowledge_select() + " WHERE status = 1"
        args: list[Any] = []
        if issue_domain:
            query += " AND issue_domain = %s"
            args.append(issue_domain)
        if issue_type:
            query += " AND issue_type = %s"
            args.append(issue_type)
        if status:
            query += " AND knowledge_status = %s"
            args.append(status)
        else:
            query += " AND knowledge_status <> 'deleted'"
        query += " ORDER BY update_time DESC LIMIT %s"
        args.append(limit)
        return self._fetch_all(query, tuple(args))

    def get_knowledge(self, knowledge_id: int) -> dict[str, Any] | None:
        return self._fetch_one(_knowledge_select() + " WHERE id = %s AND status = 1", (knowledge_id,))

    def upsert_knowledge(self, item: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        knowledge_id = int(item.get("id") or 0)
        args = (
            item["title"],
            item["issue_domain"],
            _none_if_empty(item.get("issue_type")),
            _none_if_empty(item.get("typical_description")),
            _none_if_empty(item.get("typical_ocr_features")),
            _json_or_none(item.get("required_fields")),
            _json_or_none(item.get("recommended_steps")),
            _json_or_none(item.get("common_causes")),
            _json_or_none(item.get("useful_tools")),
            _json_or_none(item.get("success_case_ids")),
            _json_or_none(item.get("failure_case_ids")),
            item.get("confidence", 0.7),
            item.get("knowledge_status") or "active",
            item.get("observed_case_count") or 1,
            _none_if_empty(item.get("last_root_cause_category")),
            _none_if_empty(item.get("last_confirmed_reason")),
            item.get("last_evolved_at") or now,
        )
        with self._connect() as conn, conn.cursor() as cur:
            if knowledge_id:
                cur.execute(
                    """
                    UPDATE tb_troubleshoot_knowledge_item SET
                    title=%s, issue_domain=%s, issue_type=%s, typical_description=%s, typical_ocr_features=%s,
                    required_fields_json=%s, recommended_steps_json=%s, common_causes_json=%s, useful_tools_json=%s,
                    success_case_ids_json=%s, failure_case_ids_json=%s, confidence=%s, knowledge_status=%s,
                    observed_case_count=%s, last_root_cause_category=%s, last_confirmed_reason=%s, last_evolved_at=%s,
                    update_time=%s
                    WHERE id=%s AND status=1
                    """,
                    (*args, now, knowledge_id),
                )
            else:
                cur.execute(
                    """
                    INSERT INTO tb_troubleshoot_knowledge_item
                    (title, issue_domain, issue_type, typical_description, typical_ocr_features, required_fields_json,
                     recommended_steps_json, common_causes_json, useful_tools_json, success_case_ids_json, failure_case_ids_json,
                     confidence, knowledge_status, observed_case_count, last_root_cause_category, last_confirmed_reason,
                     last_evolved_at, create_time, update_time)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (*args, now, now),
                )
                knowledge_id = int(cur.lastrowid)
        found = self.get_knowledge(knowledge_id)
        if found is None:
            raise KeyError("knowledge item not found")
        return found

    def delete_knowledge(self, knowledge_id: int) -> None:
        self._execute(
            "UPDATE tb_troubleshoot_knowledge_item SET knowledge_status='deleted', update_time=%s WHERE id=%s AND status=1",
            (datetime.now(), knowledge_id),
        )

    def list_capabilities(self, limit: int = 200, status: str = "", source_type: str = "") -> list[dict[str, Any]]:
        limit = _bounded(limit, 1, 500, 200)
        query = _capability_select() + " WHERE status = 1"
        args: list[Any] = []
        if status:
            query += " AND tool_status = %s"
            args.append(status)
        if source_type:
            query += " AND source_type = %s"
            args.append(source_type)
        query += " ORDER BY service_name, tool_name LIMIT %s"
        args.append(limit)
        return self._fetch_all(query, tuple(args))

    def upsert_business_service(self, item: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        self._execute(
            """
            INSERT INTO tb_troubleshoot_business_service
            (service_name, owner_team, environment, base_url, health_check_path, auth_type, secret_ref, service_status, create_time, update_time)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON DUPLICATE KEY UPDATE owner_team=VALUES(owner_team), environment=VALUES(environment), base_url=VALUES(base_url),
            health_check_path=VALUES(health_check_path), auth_type=VALUES(auth_type), secret_ref=VALUES(secret_ref),
            service_status=VALUES(service_status), update_time=VALUES(update_time)
            """,
            (
                item["service_name"],
                _none_if_empty(item.get("owner_team")),
                item.get("environment") or "local",
                _none_if_empty(item.get("base_url")),
                _none_if_empty(item.get("health_check_path")),
                item.get("auth_type") or "bearer",
                _none_if_empty(item.get("secret_ref")),
                item.get("service_status") or "enabled",
                now,
                now,
            ),
        )
        return self._fetch_one(
            "SELECT id, service_name, owner_team, environment, base_url, health_check_path, auth_type, secret_ref, service_status, create_time, update_time FROM tb_troubleshoot_business_service WHERE service_name=%s AND status=1",
            (item["service_name"],),
        ) or {}

    def upsert_tool_capability(self, item: dict[str, Any]) -> dict[str, Any]:
        now = datetime.now()
        normalized = _normalize_capability(item)
        self._execute(
            """
            INSERT INTO tb_troubleshoot_tool_registry
            (tool_name, description, service_name, source_type, input_schema_json, output_schema_json, required_scope, backend_handler,
             readonly_base_url, readonly_path, http_method, secret_ref, required_params_json, optional_params_json,
             max_time_range_minutes, max_limit, timeout_ms, sensitivity_level, safety_status, safety_reasons_json,
             approval_status, validation_status, tool_status, created_by, create_time, update_time)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON DUPLICATE KEY UPDATE description=VALUES(description), service_name=VALUES(service_name), source_type=VALUES(source_type),
            input_schema_json=VALUES(input_schema_json), output_schema_json=VALUES(output_schema_json), required_scope=VALUES(required_scope),
            backend_handler=VALUES(backend_handler), readonly_base_url=VALUES(readonly_base_url), readonly_path=VALUES(readonly_path),
            http_method=VALUES(http_method), secret_ref=VALUES(secret_ref), required_params_json=VALUES(required_params_json),
            optional_params_json=VALUES(optional_params_json), max_time_range_minutes=VALUES(max_time_range_minutes), max_limit=VALUES(max_limit),
            timeout_ms=VALUES(timeout_ms), sensitivity_level=VALUES(sensitivity_level), safety_status=VALUES(safety_status),
            safety_reasons_json=VALUES(safety_reasons_json), approval_status=VALUES(approval_status), validation_status=VALUES(validation_status),
            tool_status=VALUES(tool_status), created_by=VALUES(created_by), update_time=VALUES(update_time)
            """,
            (
                normalized["tool_name"],
                normalized["description"],
                _none_if_empty(normalized.get("service_name")),
                normalized["source_type"],
                normalized["input_schema_json"],
                normalized["output_schema_json"],
                normalized["required_scope"],
                normalized["backend_handler"],
                _none_if_empty(normalized.get("readonly_base_url")),
                _none_if_empty(normalized.get("readonly_path")),
                normalized["http_method"],
                _none_if_empty(normalized.get("secret_ref")),
                _json_or_none(normalized.get("required_params")),
                _json_or_none(normalized.get("optional_params")),
                normalized.get("max_time_range_minutes") or None,
                normalized.get("max_limit") or None,
                normalized.get("timeout_ms") or None,
                normalized["sensitivity_level"],
                normalized["safety_status"],
                _json_or_none(normalized.get("safety_reasons")),
                normalized["approval_status"],
                normalized["validation_status"],
                normalized["tool_status"],
                _none_if_empty(normalized.get("created_by")),
                now,
                now,
            ),
        )
        return self._fetch_one(_capability_select() + " WHERE tool_name=%s AND status=1", (normalized["tool_name"],)) or {}

    def update_tool_capability_status(self, capability_id: int, status: str, published_by: str) -> dict[str, Any]:
        now = datetime.now()
        if status == "enabled":
            self._execute(
                """
                UPDATE tb_troubleshoot_tool_registry
                SET tool_status=%s, approval_status='approved', published_by=%s, published_at=%s, update_time=%s
                WHERE id=%s AND status=1
                """,
                (status, published_by, now, now, capability_id),
            )
        else:
            self._execute(
                "UPDATE tb_troubleshoot_tool_registry SET tool_status=%s, update_time=%s WHERE id=%s AND status=1",
                (status, now, capability_id),
            )
        found = self._fetch_one(_capability_select() + " WHERE id=%s AND status=1", (capability_id,))
        if found is None:
            raise KeyError("capability not found")
        return found

    def _execute(self, query: str, args: tuple[Any, ...]) -> None:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(query, args)

    def _fetch_one(self, query: str, args: tuple[Any, ...]) -> dict[str, Any] | None:
        rows = self._fetch_all(query, args)
        return rows[0] if rows else None

    def _fetch_all(self, query: str, args: tuple[Any, ...]) -> list[dict[str, Any]]:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(query, args)
            return [dict(row) for row in cur.fetchall()]


def _case_select() -> str:
    return """
    SELECT id, case_no, case_title AS title, uid, source, chat_id, thread_id, message_id, reporter_user_id,
           original_text, ocr_text, issue_domain, issue_type, case_status AS status, priority, timezone,
           occurred_at, create_time AS created_at, update_time AS updated_at, closed_at, version
    FROM tb_troubleshoot_case
    """


def _knowledge_select() -> str:
    return """
    SELECT id, title, issue_domain, issue_type, typical_description, typical_ocr_features,
           required_fields_json, recommended_steps_json, common_causes_json, useful_tools_json,
           success_case_ids_json, failure_case_ids_json, confidence, knowledge_status AS status,
           observed_case_count, last_root_cause_category, last_confirmed_reason, last_evolved_at,
           create_time AS created_at, update_time AS updated_at
    FROM tb_troubleshoot_knowledge_item
    """


def _capability_select() -> str:
    return """
    SELECT id, tool_name, description, service_name, source_type, input_schema_json, output_schema_json,
           required_scope, backend_handler, readonly_base_url, readonly_path, http_method, secret_ref,
           mcp_server_id, mcp_tool_name, param_map_json, fixed_params_json, required_params_json,
           optional_params_json, max_time_range_minutes, max_limit, timeout_ms, sensitivity_level,
           safety_status, safety_reasons_json, approval_status, validation_status, tool_status,
           created_by, published_by, published_at, create_time AS created_at, update_time AS updated_at
    FROM tb_troubleshoot_tool_registry
    """


def _context_ledger_select() -> str:
    return """
    SELECT id, case_id, ledger_type, ledger_key, summary, evidence_refs_json, payload_json,
           source_agent, create_time AS created_at, update_time AS updated_at
    FROM tb_troubleshoot_context_ledger
    """


def _agent_runtime_select() -> str:
    return """
    SELECT id, runtime_id, runtime_name, runtime_type, host_name, provider_list_json, workspace_root,
           runtime_status AS status, last_heartbeat_at, registered_at, create_time AS created_at, update_time AS updated_at
    FROM tb_troubleshoot_agent_runtime
    """


def _agent_run_select() -> str:
    return """
    SELECT id, run_no, case_id, investigation_id, parent_run_id, runtime_id, agent_name, agent_role,
           trigger_type, run_status AS status, input_summary, output_summary, model_provider, model_name,
           started_at, finished_at, latency_ms, error_message, payload_json, create_time AS created_at, update_time AS updated_at
    FROM tb_troubleshoot_agent_run
    """


def _agent_run_event_select() -> str:
    return """
    SELECT id, run_id, event_seq, event_type, event_status AS status, title, summary, payload_json,
           create_time AS created_at, update_time AS updated_at
    FROM tb_troubleshoot_agent_run_event
    """


def _decode_context_ledger_row(row: dict[str, Any]) -> dict[str, Any]:
    out = dict(row)
    out["evidence_refs"] = _loads_json(out.pop("evidence_refs_json", None), [])
    out["payload"] = _loads_json(out.pop("payload_json", None), {})
    return out


def _decode_agent_runtime_row(row: dict[str, Any]) -> dict[str, Any]:
    out = dict(row)
    out["provider_list"] = _loads_json(out.pop("provider_list_json", None), [])
    return out


def _decode_agent_run_row(row: dict[str, Any]) -> dict[str, Any]:
    out = dict(row)
    out["payload"] = _loads_json(out.pop("payload_json", None), {})
    return out


def _decode_agent_run_event_row(row: dict[str, Any]) -> dict[str, Any]:
    out = dict(row)
    out["payload"] = _loads_json(out.pop("payload_json", None), {})
    return out


def _normalize_capability(item: dict[str, Any]) -> dict[str, Any]:
    required = [str(v) for v in item.get("required_params", []) if str(v).strip()]
    optional = [str(v) for v in item.get("optional_params", []) if str(v).strip()]
    input_schema = item.get("input_schema_json") or {"type": "object", "required": required, "properties": {name: {"type": "string"} for name in [*required, *optional]}}
    out = dict(item)
    out.setdefault("description", out.get("tool_name", ""))
    out.setdefault("source_type", "http_adapter")
    out.setdefault("input_schema_json", json.dumps(input_schema, ensure_ascii=False))
    out.setdefault("output_schema_json", json.dumps({"type": "object"}, ensure_ascii=False))
    out.setdefault("backend_handler", "dynamic_http." + out["tool_name"])
    out.setdefault("http_method", "POST")
    out.setdefault("sensitivity_level", "normal")
    out.setdefault("safety_status", "readonly_candidate")
    out.setdefault("safety_reasons", ["readonly path and safe method accepted"])
    out.setdefault("approval_status", "pending")
    out.setdefault("validation_status", "not_run")
    out.setdefault("tool_status", "draft")
    return out


def _none_if_empty(value: Any) -> Any:
    if value is None:
        return None
    if isinstance(value, str) and value.strip() == "":
        return None
    return value


def _json_or_none(value: Any) -> str | None:
    if value is None or value == "":
        return None
    if isinstance(value, str):
        return value
    return json.dumps(value, ensure_ascii=False, default=str)


def _loads_json(value: Any, default: Any) -> Any:
    if not value:
        return default
    if not isinstance(value, str):
        return value
    try:
        return json.loads(value)
    except json.JSONDecodeError:
        return default


def _bounded(value: int, minimum: int, maximum: int, default: int) -> int:
    if value < minimum or value > maximum:
        return default
    return value
