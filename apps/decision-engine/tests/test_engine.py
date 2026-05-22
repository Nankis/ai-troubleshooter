import unittest

from decision_engine import CaseSnapshot, DecisionEngine, DecisionRequest
from decision_engine.agent_team import Verifier
from decision_engine.models import AgentReport, DecisionResponse, KnowledgeCandidate, ToolPlan, ToolSpec


class DecisionEngineTest(unittest.TestCase):
    def test_kline_missing_fields_asks_user(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={"symbol": "BTCUSDT"},
            )
        )

        self.assertEqual(response.action, "ask_user")
        self.assertIn("interval", response.missing_fields)
        self.assertIn("abnormal_time", response.missing_fields)
        self.assertIsNotNone(response.verification)
        self.assertEqual(response.verification.tool_count, 0)

    def test_kline_complete_fields_plans_tools(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={
                    "symbol": "BTCUSDT",
                    "interval": "1m",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "price_mismatch",
                },
                max_tool_calls=2,
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_internal_kline", "get_external_kline_compare"])
        self.assertEqual([item.agent_name for item in response.agent_reports], ["supervisor", "knowledge_agent", "kline_agent"])
        self.assertTrue(response.verification.accepted)
        self.assertEqual(response.verification.tool_budget, 2)

    def test_available_tools_filter_plan(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="asset"),
                entities={
                    "user_id": "u_1",
                    "asset_symbol": "USDT",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "balance_mismatch",
                },
                available_tools=[ToolSpec(name="get_asset_events")],
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_asset_events"])
        self.assertEqual(response.agent_reports[-1].agent_name, "asset_agent")

    def test_asset_requires_user_or_account(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="asset"),
                entities={
                    "asset_symbol": "USDT",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "balance_mismatch",
                },
            )
        )

        self.assertEqual(response.action, "ask_user")
        self.assertEqual(response.missing_fields[0], "user_id_or_account_id")
        self.assertEqual(response.agent_reports[-1].agent_name, "asset_agent")

    def test_high_confidence_knowledge_can_answer_directly(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={
                    "symbol": "BTCUSDT",
                    "interval": "1m",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "known_sop",
                },
                knowledge_candidates=[
                    KnowledgeCandidate(
                        title="历史 SOP",
                        confidence=0.93,
                        observed_case_count=3,
                        requires_realtime_check=False,
                        source="knowledge:1",
                    )
                ],
            )
        )

        self.assertEqual(response.action, "answer_from_knowledge")
        self.assertEqual(response.knowledge_source, "knowledge:1")
        self.assertEqual([item.agent_name for item in response.agent_reports], ["supervisor", "knowledge_agent"])
        self.assertTrue(response.verification.accepted)
        self.assertEqual(response.verification.tool_count, 0)

    def test_realtime_knowledge_still_invokes_tools(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={
                    "symbol": "BTCUSDT",
                    "interval": "1m",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "price_mismatch",
                },
                knowledge_candidates=[
                    KnowledgeCandidate(
                        title="历史 SOP",
                        confidence=0.94,
                        observed_case_count=4,
                        requires_realtime_check=True,
                    )
                ],
            )
        )

        self.assertEqual(response.action, "invoke_tools")

    def test_unknown_domain_uses_fallback_agent(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="ops"),
                entities={"service_name": "market-api", "trace_id": "trace_1"},
                max_tool_calls=2,
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["search_logs_by_service", "get_recent_deployments"])
        self.assertEqual(response.agent_reports[-1].agent_name, "fallback_agent")

    def test_verifier_filters_unavailable_and_truncates_tools(self) -> None:
        verifier = Verifier()
        request = DecisionRequest(
            case=CaseSnapshot(case_no="case_1", issue_domain="asset"),
            available_tools=[ToolSpec(name="get_asset_events")],
            max_tool_calls=1,
        )
        proposal = DecisionResponse(
            action="invoke_tools",
            reason="specialist proposal",
            tool_plan=[
                ToolPlan(tool_name="get_asset_snapshot", reason="unavailable"),
                ToolPlan(tool_name="get_asset_events", reason="allowed"),
                ToolPlan(tool_name="get_similar_cases", reason="over budget"),
            ],
        )

        response = verifier.verify(
            request,
            proposal,
            [AgentReport(agent_name="asset_agent", action="invoke_tools", reason="test")],
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_asset_events"])
        self.assertIn("unavailable_tool=get_asset_snapshot", response.verification.violations)
        self.assertIn("tool_plan_truncated_by_budget", response.verification.violations)


if __name__ == "__main__":
    unittest.main()
