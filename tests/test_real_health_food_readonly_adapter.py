from __future__ import annotations

import importlib.util
import json
import os
import sys
import threading
import unittest
import urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest import mock


def load_real_adapter(env: dict[str, str]):
    path = Path(__file__).resolve().parents[1] / "scripts" / "real-health-food-readonly-adapter.py"
    module_name = f"real_health_food_readonly_adapter_{id(env)}"
    spec = importlib.util.spec_from_file_location(module_name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError("failed to load real-health-food-readonly-adapter.py")
    with mock.patch.dict(os.environ, env, clear=False):
        module = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = module
        spec.loader.exec_module(module)
    return module


class FakeHealthFoodAdminHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        parsed = urllib.parse.urlparse(self.path)
        query = urllib.parse.parse_qs(parsed.query)
        self.server.seen_queries.append(query)  # type: ignore[attr-defined]
        if query.get("password", [""])[0] != "unit-test-secret":
            self.write_json({"code": -1, "msg": "密码错误"})
            return
        self.write_json(
            {
                "code": 0,
                "msg": "success",
                "data": {
                    "lines": [
                        {
                            "time": "2026-05-23 10:05:01,123",
                            "summary": "ERROR FoodServiceImpl - recommend failed token=abc123456789",
                            "text": (
                                "2026-05-23 10:05:01,123 [worker] ERROR FoodServiceImpl.java:526 - "
                                "trace_prod_1 - generateDailyFoodRecommend error password=unit-test-secret "
                                "email=user@example.com phone=13800138000"
                            ),
                        }
                    ],
                    "totalMatches": 1,
                    "page": 1,
                    "pageSize": 10,
                    "hasMore": False,
                },
            }
        )

    def write_json(self, value: dict) -> None:
        body = json.dumps(value).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt: str, *args: object) -> None:
        return


class RealHealthFoodReadonlyAdapterTest(unittest.TestCase):
    def test_local_mysql_queries_use_bound_parameters(self) -> None:
        adapter = load_real_adapter({"HEALTH_FOOD_ALLOWED_SERVICE_NAMES": "health-food"})
        calls: list[tuple[str, tuple[object, ...] | None]] = []

        def fake_mysql_query(sql: str, params: tuple[object, ...] | None = None) -> list[dict]:
            calls.append((sql, params))
            return []

        with mock.patch.object(adapter, "mysql_query", side_effect=fake_mysql_query):
            adapter.query_meals(
                "2054603630081875968",
                {
                    "start_time": "2026-05-23T00:00:00+08:00",
                    "end_time": "2026-05-24T00:00:00+08:00",
                },
            )

        self.assertEqual(len(calls), 1)
        sql, params = calls[0]
        self.assertIn("uid=%s", sql)
        self.assertIn("BETWEEN %s AND %s", sql)
        self.assertNotIn("2054603630081875968", sql)
        self.assertEqual(params[0], "2054603630081875968")

    def test_admin_log_upstream_is_normalized_masked_and_limited(self) -> None:
        server = ThreadingHTTPServer(("127.0.0.1", 0), FakeHealthFoodAdminHandler)
        server.seen_queries = []  # type: ignore[attr-defined]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

        def cleanup_server() -> None:
            server.shutdown()
            thread.join(1)
            server.server_close()

        self.addCleanup(cleanup_server)

        adapter = load_real_adapter(
            {
                "HEALTH_FOOD_ADMIN_BASE_URL": f"http://127.0.0.1:{server.server_port}",
                "HEALTH_FOOD_ADMIN_SECRET": "unit-test-secret",
                "HEALTH_FOOD_LOG_MAX_LIMIT": "5",
                "HEALTH_FOOD_ALLOWED_SERVICE_NAMES": "health-food",
            }
        )

        status, body = adapter.handle_ops(
            "/v1/readonly/ops/logs/search",
            {
                "request_id": "req_prod_log_1",
                "params": {
                    "ServiceName": "health-food",
                    "StartTime": "2026-05-23T10:00:00+08:00",
                    "EndTime": "2026-05-23T10:10:00+08:00",
                    "Keyword": "recommend",
                    "Limit": 3,
                },
            },
        )

        self.assertEqual(status, 200)
        self.assertEqual(body["data"]["total"], 1)
        sample = body["data"]["samples"][0]
        self.assertEqual(sample["level"], "error")
        self.assertEqual(sample["trace_id"], "trace_prod_1")
        encoded_body = json.dumps(body, ensure_ascii=False)
        self.assertNotIn("unit-test-secret", encoded_body)
        self.assertNotIn("user@example.com", encoded_body)
        self.assertNotIn("13800138000", encoded_body)
        self.assertIn("<redacted>", sample["excerpt"])

        seen_queries = server.seen_queries  # type: ignore[attr-defined]
        self.assertEqual(seen_queries[0]["password"][0], "unit-test-secret")
        self.assertEqual(seen_queries[0]["content"][0], "recommend")
        self.assertEqual(seen_queries[0]["date"][0], "2026-05-23")


if __name__ == "__main__":
    unittest.main()
