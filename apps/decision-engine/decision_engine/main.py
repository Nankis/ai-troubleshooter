from __future__ import annotations

import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from .engine import DecisionEngine
from .models import DecisionRequest


class DecisionHandler(BaseHTTPRequestHandler):
    engine = DecisionEngine()

    def do_GET(self) -> None:
        if self.path == "/healthz":
            self._write_json(200, {"ok": True, "service": "decision-engine"})
            return
        self._write_json(404, {"error": "not found"})

    def do_POST(self) -> None:
        if self.path != "/v1/decisions/plan":
            self._write_json(404, {"error": "not found"})
            return
        try:
            payload = self._read_json()
            request = DecisionRequest.from_dict(payload)
            response = self.engine.plan(request)
        except Exception as exc:
            self._write_json(400, {"error": str(exc)})
            return
        self._write_json(200, response.to_dict())

    def log_message(self, format: str, *args: Any) -> None:
        return

    def _read_json(self) -> dict[str, Any]:
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length).decode("utf-8") if length else "{}"
        value = json.loads(raw)
        if not isinstance(value, dict):
            raise ValueError("request body must be a JSON object")
        return value

    def _write_json(self, status: int, payload: dict[str, Any]) -> None:
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def main() -> None:
    parser = argparse.ArgumentParser(description="Run the ai-troubleshooter decision engine.")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=19092)
    args = parser.parse_args()
    server = ThreadingHTTPServer((args.host, args.port), DecisionHandler)
    print(f"decision-engine listening on http://{args.host}:{args.port}")
    server.serve_forever()


if __name__ == "__main__":
    main()

