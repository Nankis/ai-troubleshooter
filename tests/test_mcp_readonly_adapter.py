from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path


def load_adapter_module():
    path = Path(__file__).resolve().parents[1] / "scripts" / "mcp-readonly-adapter.py"
    spec = importlib.util.spec_from_file_location("mcp_readonly_adapter", path)
    if spec is None or spec.loader is None:
        raise RuntimeError("failed to load mcp-readonly-adapter.py")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class MCPReadonlyAdapterTest(unittest.TestCase):
    def test_normalize_params_adds_snake_case_aliases(self) -> None:
        adapter = load_adapter_module()

        params = adapter.normalize_params(
            {
                "ServiceName": "health-food",
                "StartTime": "2026-05-23T09:50:00+08:00",
                "trace-id": "trace_1",
                "uid": "hf_user_001",
            }
        )

        self.assertEqual(params["service_name"], "health-food")
        self.assertEqual(params["start_time"], "2026-05-23T09:50:00+08:00")
        self.assertEqual(params["trace_id"], "trace_1")
        self.assertEqual(params["uid"], "hf_user_001")


if __name__ == "__main__":
    unittest.main()
