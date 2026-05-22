from __future__ import annotations

import json
import urllib.request
from dataclasses import dataclass
from typing import Any

from .models import ToolSpec


@dataclass(slots=True)
class GatewayClient:
    endpoint: str
    bearer_token: str = ""
    timeout_seconds: int = 5

    def list_tools(self) -> list[ToolSpec]:
        data = self._request("GET", "/tools")
        return [ToolSpec.from_dict(item) for item in data.get("tools", []) if isinstance(item, dict)]

    def invoke_tool(self, tool_name: str, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", f"/tools/{tool_name}/invoke", payload)

    def _request(self, method: str, path: str, payload: dict[str, Any] | None = None) -> dict[str, Any]:
        body = None
        headers = {"Content-Type": "application/json"}
        if self.bearer_token:
            headers["Authorization"] = f"Bearer {self.bearer_token}"
        if payload is not None:
            body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        req = urllib.request.Request(
            self.endpoint.rstrip("/") + path,
            data=body,
            method=method,
            headers=headers,
        )
        with urllib.request.urlopen(req, timeout=self.timeout_seconds) as resp:
            raw = resp.read().decode("utf-8")
        return json.loads(raw or "{}")

