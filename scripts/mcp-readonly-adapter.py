#!/usr/bin/env python3
"""Bridge an allowlisted MCP server into the readonly adapter HTTP contract.

The decision layer still talks only to Investigation Gateway. Gateway talks to
this adapter through the existing readonly HTTP connector, and this adapter is
the only component that talks to the MCP server.
"""

from __future__ import annotations

import json
import os
import queue
import re
import subprocess
import sys
import threading
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any


DEFAULT_CONFIG = {
    "server": {
        "command": [sys.executable, "scripts/mock-health-food-mcp-server.py"],
        "request_timeout_seconds": 5,
        "protocol_version": "2025-06-18",
    },
    "routes": [
        {
            "path": "/v1/readonly/health-food/user/profile",
            "tool_name": "health_food_user_profile",
            "source": "health-food-mcp/mock",
            "version": "mcp-mock-v1",
            "required_params": ["uid"],
        },
        {
            "path": "/v1/readonly/health-food/ai/quota",
            "tool_name": "health_food_ai_quota",
            "source": "health-food-mcp/mock",
            "version": "mcp-mock-v1",
            "required_params": ["uid"],
        },
        {
            "path": "/v1/readonly/health-food/meals/range",
            "tool_name": "health_food_meal_records",
            "source": "health-food-mcp/mock",
            "version": "mcp-mock-v1",
            "required_params": ["uid"],
        },
        {
            "path": "/v1/readonly/health-food/recommendation/status",
            "tool_name": "health_food_recommendation_status",
            "source": "health-food-mcp/mock",
            "version": "mcp-mock-v1",
            "required_params": ["uid", "recommendation_date"],
        },
        {
            "path": "/v1/readonly/ops/logs/search",
            "tool_name": "health_food_search_logs",
            "source": "health-food-mcp/mock",
            "version": "mcp-mock-v1",
            "required_params": ["service_name"],
        },
        {
            "path": "/v1/readonly/ops/cases/similar",
            "tool_name": "health_food_similar_cases",
            "source": "health-food-mcp/mock",
            "version": "mcp-mock-v1",
            "required_params": [],
        },
    ],
}


@dataclass(frozen=True, slots=True)
class Route:
    path: str
    tool_name: str
    source: str
    version: str
    required_params: tuple[str, ...]
    param_map: dict[str, str]
    fixed_params: dict[str, Any]
    forward_all_params: bool


class MCPProtocolError(RuntimeError):
    pass


class MCPClient:
    def __init__(self, command: list[str], protocol_version: str, request_timeout_seconds: float) -> None:
        if not command:
            raise ValueError("MCP server command is required")
        self.command = command
        self.protocol_version = protocol_version
        self.request_timeout_seconds = request_timeout_seconds
        self._next_id = 1
        self._lock = threading.Lock()
        self._responses: queue.Queue[dict[str, Any]] = queue.Queue()
        self._process = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
        )
        self._stdout_thread = threading.Thread(target=self._read_stdout, daemon=True)
        self._stderr_thread = threading.Thread(target=self._read_stderr, daemon=True)
        self._stdout_thread.start()
        self._stderr_thread.start()
        self.initialize()

    def close(self) -> None:
        if self._process.poll() is None:
            self._process.terminate()
            try:
                self._process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                self._process.kill()

    def initialize(self) -> None:
        result = self.request(
            "initialize",
            {
                "protocolVersion": self.protocol_version,
                "capabilities": {},
                "clientInfo": {"name": "ai-troubleshooter-mcp-readonly-adapter", "version": "0.1.0"},
            },
        )
        if not isinstance(result, dict):
            raise MCPProtocolError("MCP initialize did not return an object")
        self.notify("notifications/initialized", {})

    def list_tools(self) -> list[dict[str, Any]]:
        result = self.request("tools/list", {})
        tools = result.get("tools") if isinstance(result, dict) else None
        if not isinstance(tools, list):
            raise MCPProtocolError("MCP tools/list did not return tools")
        return [tool for tool in tools if isinstance(tool, dict)]

    def call_tool(self, name: str, arguments: dict[str, Any]) -> dict[str, Any]:
        result = self.request("tools/call", {"name": name, "arguments": arguments})
        if not isinstance(result, dict):
            raise MCPProtocolError("MCP tools/call did not return an object")
        if result.get("isError"):
            raise MCPProtocolError(f"MCP tool {name} returned isError=true")
        structured = result.get("structuredContent")
        if isinstance(structured, dict):
            return structured
        content = result.get("content")
        if isinstance(content, list):
            texts = [item.get("text", "") for item in content if isinstance(item, dict) and item.get("type") == "text"]
            joined = "\n".join(item for item in texts if item)
            if joined:
                try:
                    parsed = json.loads(joined)
                except json.JSONDecodeError:
                    return {"text": joined}
                if isinstance(parsed, dict):
                    return parsed
                return {"value": parsed}
        return {"raw_result": result}

    def request(self, method: str, params: dict[str, Any]) -> Any:
        with self._lock:
            request_id = self._next_id
            self._next_id += 1
            self._write({"jsonrpc": "2.0", "id": request_id, "method": method, "params": params})
            deadline = time.monotonic() + self.request_timeout_seconds
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise TimeoutError(f"MCP request {method} timed out")
                try:
                    message = self._responses.get(timeout=remaining)
                except queue.Empty as exc:
                    raise TimeoutError(f"MCP request {method} timed out") from exc
                if message.get("id") != request_id:
                    continue
                if "error" in message:
                    raise MCPProtocolError(f"MCP request {method} failed: {message['error']}")
                return message.get("result")

    def notify(self, method: str, params: dict[str, Any]) -> None:
        with self._lock:
            self._write({"jsonrpc": "2.0", "method": method, "params": params})

    def _write(self, message: dict[str, Any]) -> None:
        if self._process.poll() is not None:
            raise MCPProtocolError("MCP server process exited")
        if self._process.stdin is None:
            raise MCPProtocolError("MCP server stdin is unavailable")
        self._process.stdin.write(json.dumps(message, separators=(",", ":")) + "\n")
        self._process.stdin.flush()

    def _read_stdout(self) -> None:
        assert self._process.stdout is not None
        for line in self._process.stdout:
            line = line.strip()
            if not line:
                continue
            try:
                message = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(message, dict) and "id" in message:
                self._responses.put(message)

    def _read_stderr(self) -> None:
        assert self._process.stderr is not None
        for line in self._process.stderr:
            print(f"[mcp-server] {line.rstrip()}", file=sys.stderr)


def now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def load_config() -> dict[str, Any]:
    raw = os.getenv("MCP_ADAPTER_CONFIG_JSON", "").strip()
    if not raw:
        return DEFAULT_CONFIG
    data = json.loads(raw)
    if not isinstance(data, dict):
        raise ValueError("MCP_ADAPTER_CONFIG_JSON must be a JSON object")
    return data


def parse_routes(config: dict[str, Any]) -> dict[str, Route]:
    routes: dict[str, Route] = {}
    for item in config.get("routes") or []:
        if not isinstance(item, dict):
            continue
        path = str(item.get("path") or "").strip()
        tool_name = str(item.get("tool_name") or "").strip()
        if not path.startswith("/") or not tool_name:
            raise ValueError(f"invalid MCP route config: {item}")
        routes[path] = Route(
            path=path,
            tool_name=tool_name,
            source=str(item.get("source") or "mcp-readonly-adapter"),
            version=str(item.get("version") or "mcp-v1"),
            required_params=tuple(str(param) for param in item.get("required_params") or ()),
            param_map={str(key): str(value) for key, value in (item.get("param_map") or {}).items()},
            fixed_params=dict(item.get("fixed_params") or {}),
            forward_all_params=bool(item.get("forward_all_params", True)),
        )
    if not routes:
        raise ValueError("at least one MCP route is required")
    return routes


class AdapterState:
    def __init__(self) -> None:
        self.config = load_config()
        server = self.config.get("server") or {}
        if not isinstance(server, dict):
            raise ValueError("server config must be an object")
        command = [str(item) for item in server.get("command") or []]
        timeout = float(server.get("request_timeout_seconds") or 5)
        protocol_version = str(server.get("protocol_version") or os.getenv("MCP_PROTOCOL_VERSION", "2025-06-18"))
        self.routes = parse_routes(self.config)
        self.client = MCPClient(command, protocol_version=protocol_version, request_timeout_seconds=timeout)
        self.tools = self.client.list_tools()
        self.tool_names = {str(tool.get("name")) for tool in self.tools}
        missing = sorted({route.tool_name for route in self.routes.values()} - self.tool_names)
        if missing:
            raise ValueError(f"MCP server missing allowlisted tools: {', '.join(missing)}")

    def close(self) -> None:
        self.client.close()


def normalize_params(params: dict[str, Any]) -> dict[str, Any]:
    normalized = dict(params)
    for key, value in list(params.items()):
        if not isinstance(key, str):
            continue
        snake = to_snake_case(key)
        if snake and snake not in normalized:
            normalized[snake] = value
    return normalized


def tool_arguments(route: Route, params: dict[str, Any]) -> dict[str, Any]:
    if route.forward_all_params:
        arguments = dict(params)
    else:
        arguments = {}
    for source_name, target_name in route.param_map.items():
        if source_name in params:
            arguments[target_name] = params[source_name]
    for key, value in route.fixed_params.items():
        arguments[key] = value
    return arguments


def to_snake_case(value: str) -> str:
    value = value.strip()
    if not value:
        return ""
    value = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", value)
    value = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", value)
    value = value.replace("-", "_")
    return value.lower()


STATE: AdapterState | None = None
API_KEY = os.getenv("MCP_ADAPTER_API_KEY", "")
PORT = int(os.getenv("MCP_READONLY_ADAPTER_PORT", "19085"))


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/healthz":
            assert STATE is not None
            self.write_json(
                200,
                {
                    "ok": True,
                    "routes": sorted(STATE.routes),
                    "mcp_tools": sorted(STATE.tool_names),
                    "source": "mcp-readonly-adapter",
                },
            )
            return
        self.write_json(404, {"code": "NOT_FOUND", "error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        assert STATE is not None
        if API_KEY and self.headers.get("Authorization") != f"Bearer {API_KEY}":
            self.write_json(401, {"code": "UNAUTHORIZED", "error": "invalid adapter token"})
            return
        route = STATE.routes.get(self.path)
        if route is None:
            self.write_json(404, {"code": "NOT_FOUND", "error": f"unknown readonly path {self.path}"})
            return
        length = int(self.headers.get("Content-Length") or "0")
        raw = self.rfile.read(length)
        try:
            payload = json.loads(raw or b"{}")
        except json.JSONDecodeError:
            self.write_json(400, {"code": "BAD_JSON", "error": "invalid json"})
            return
        params = payload.get("params") or {}
        if not isinstance(params, dict):
            self.write_json(400, {"code": "BAD_PARAMS", "error": "params must be object"})
            return
        params = normalize_params(params)
        missing = [name for name in route.required_params if not str(params.get(name) or "").strip()]
        if missing:
            self.write_json(400, {"code": "MISSING_PARAM", "error": f"missing required params: {', '.join(missing)}"})
            return
        arguments = tool_arguments(route, params)
        try:
            data = STATE.client.call_tool(route.tool_name, arguments)
        except TimeoutError as exc:
            self.write_json(504, {"code": "MCP_TIMEOUT", "error": str(exc)})
            return
        except Exception as exc:  # noqa: BLE001 - adapter must surface MCP errors as readonly failures.
            self.write_json(502, {"code": "MCP_TOOL_ERROR", "error": str(exc)})
            return
        request_id = str(payload.get("request_id") or f"req_mcp_{int(time.time() * 1000)}")
        self.write_json(
            200,
            {
                "request_id": request_id,
                "source": route.source,
                "queried_at": now_iso(),
                "data_updated_at": data.get("data_updated_at") or data.get("updated_at") or now_iso(),
                "version": route.version,
                "data": data,
                "warnings": data.get("warnings") if isinstance(data.get("warnings"), list) else [],
            },
        )

    def write_json(self, status: int, body: dict[str, Any]) -> None:
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, fmt: str, *args: object) -> None:
        print(f"{self.log_date_time_string()} {fmt % args}")


def main() -> None:
    global STATE
    STATE = AdapterState()
    server = ThreadingHTTPServer(("127.0.0.1", PORT), Handler)
    print(f"mcp readonly adapter listening on http://127.0.0.1:{PORT}")
    try:
        server.serve_forever()
    finally:
        STATE.close()


if __name__ == "__main__":
    main()
