from __future__ import annotations

import json
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any


@dataclass(slots=True)
class GatewayHTTPClient:
    endpoint: str
    bearer_token: str = ""
    admin_bearer_token: str = ""
    timeout_seconds: int = 5

    def list_tools(self) -> list[dict[str, Any]]:
        payload = self._request("GET", "/tools")
        return [item for item in payload.get("tools", []) if isinstance(item, dict)]

    def invoke_tool(
        self,
        tool_name: str,
        *,
        case_no: str,
        agent_id: str,
        caller_user_id: str,
        chat_id: str,
        arguments: dict[str, Any],
    ) -> dict[str, Any]:
        return self._request(
            "POST",
            f"/tools/{tool_name}/invoke",
            {
                "case_id": case_no,
                "agent_id": agent_id,
                "caller_user_id": caller_user_id,
                "chat_id": chat_id,
                "arguments": arguments,
            },
        )

    def reload_capabilities(self) -> dict[str, Any]:
        return self._request("POST", "/admin/capabilities/reload", bearer_token=self.admin_bearer_token or self.bearer_token)

    def _request(self, method: str, path: str, payload: dict[str, Any] | None = None, bearer_token: str | None = None) -> dict[str, Any]:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8") if payload is not None else None
        headers = {"Content-Type": "application/json"}
        token = self.bearer_token if bearer_token is None else bearer_token
        if token:
            headers["Authorization"] = f"Bearer {token}"
        request = urllib.request.Request(self.endpoint.rstrip("/") + path, data=body, method=method, headers=headers)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:
                raw = response.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            raw = exc.read().decode("utf-8")
            try:
                value = json.loads(raw or "{}")
            except json.JSONDecodeError:
                value = {"error": raw or exc.reason}
            raise RuntimeError(value.get("error") or value.get("summary") or str(exc)) from exc
        return json.loads(raw or "{}")
