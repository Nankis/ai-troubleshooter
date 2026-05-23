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

    def test_tool_arguments_can_map_gateway_params_to_mcp_schema(self) -> None:
        adapter = load_adapter_module()
        route = adapter.Route(
            path="/v1/readonly/db/tables/list",
            tool_name="listTables",
            source="dms-mcp",
            version="v1",
            required_params=("schema_name",),
            param_map={"schema_name": "schemaName", "page_size": "pageSize"},
            fixed_params={"envType": "product"},
            forward_all_params=False,
        )

        arguments = adapter.tool_arguments(
            route,
            {
                "schema_name": "health_food",
                "page_size": 20,
                "extra": "should-not-forward",
            },
        )

        self.assertEqual(
            arguments,
            {
                "schemaName": "health_food",
                "pageSize": 20,
                "envType": "product",
            },
        )


if __name__ == "__main__":
    unittest.main()
