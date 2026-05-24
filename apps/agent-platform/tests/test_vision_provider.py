from __future__ import annotations

import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Any

from agent_platform.config import VisionConfig
from agent_platform.vision import ImageInput, OpenAICompatibleVisionClient


class VisionProviderTest(unittest.TestCase):
    def test_openai_compatible_vision_posts_image_data_url_and_parses_json(self) -> None:
        captured: dict[str, Any] = {}

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self) -> None:
                length = int(self.headers.get("Content-Length") or "0")
                captured["path"] = self.path
                captured["authorization"] = self.headers.get("Authorization")
                captured["body"] = json.loads(self.rfile.read(length).decode("utf-8"))
                response = {
                    "choices": [
                        {
                            "message": {
                                "content": json.dumps(
                                    {
                                        "ocr_text": "uid hf-user-vision 今日 token 为 0",
                                        "summary": "截图显示 token 额度为 0",
                                        "key_fields": {"uid": "hf-user-vision"},
                                        "uncertainties": [],
                                    },
                                    ensure_ascii=False,
                                )
                            }
                        }
                    ]
                }
                raw = json.dumps(response, ensure_ascii=False).encode("utf-8")
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

            def log_message(self, _: str, *args: Any) -> None:
                return

        server = HTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            base_url = f"http://127.0.0.1:{server.server_port}/compatible-mode/v1"
            client = OpenAICompatibleVisionClient(
                VisionConfig("qwen_openai_compatible", base_url, "unit-test-vision-key", "qwen-vl-plus", 5, 3, 10 * 1024 * 1024)
            )

            result = client.analyze(
                "用户说截图里 token 不对",
                [ImageInput("quota.png", "image/png", b"\x89PNG\r\n\x1a\nunit-test")],
            )
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)

        self.assertTrue(result.is_real)
        self.assertEqual(result.provider, "qwen_openai_compatible")
        self.assertEqual(result.model, "qwen-vl-plus")
        self.assertIn("hf-user-vision", result.ocr_text)
        self.assertEqual(result.summary, "截图显示 token 额度为 0")
        self.assertEqual(captured["path"], "/compatible-mode/v1/chat/completions")
        self.assertEqual(captured["authorization"], "Bearer unit-test-vision-key")
        body = captured["body"]
        self.assertEqual(body["model"], "qwen-vl-plus")
        content = body["messages"][1]["content"]
        self.assertEqual(content[0]["type"], "text")
        self.assertEqual(content[1]["type"], "image_url")
        self.assertTrue(content[1]["image_url"]["url"].startswith("data:image/png;base64,"))

    def test_openai_compatible_vision_requires_real_provider_config(self) -> None:
        client = OpenAICompatibleVisionClient(VisionConfig("openai", "https://api.openai.com/v1", "", "gpt-4.1-mini", 1, 3, 1024))

        with self.assertRaisesRegex(RuntimeError, "VISION_API_KEY"):
            client.analyze("请识别截图", [ImageInput("x.png", "image/png", b"png")])


if __name__ == "__main__":
    unittest.main()
