from __future__ import annotations

import unittest

from agent_platform.service import _decision_log_snapshot, _local_code_findings, _mask
from decision_engine import AgentReport, DecisionResponse, VerificationReport


class ServiceHelperTest(unittest.TestCase):
    def test_mask_redacts_token_balance_in_summary_text(self) -> None:
        self.assertEqual(
            _mask("health-food ai quota normal: tokens=123456.000000 daily_chat=996/1000"),
            "health-food ai quota normal: tokens=<redacted> daily_chat=996/1000",
        )

    def test_local_code_findings_are_actionable(self) -> None:
        findings = _local_code_findings(
            [
                {
                    "file_path": "src/main/java/com/example/FoodServiceImpl.java",
                    "line_range": {"start": 10, "end": 15},
                    "line_numbers": [12],
                    "primary_symbol": {"name": "FoodServiceImpl.generate", "kind": "method", "line_number": 10},
                    "suspect_reasons": ["命中问题关键词：recommendation"],
                    "follow_up_checks": ["核对 uid 过滤。"],
                    "code_excerpt": [
                        {"line_number": 12, "text": "return recommend(uid);"},
                    ],
                }
            ],
            1,
        )

        self.assertIn("FoodServiceImpl.java:10-15", findings[0])
        self.assertIn("方法/符号", findings[0])
        self.assertIn("可疑点", findings[0])
        self.assertIn("建议核对", findings[0])
        self.assertIn("L12: return recommend(uid);", findings[0])

    def test_decision_log_snapshot_compacts_local_code_excerpt(self) -> None:
        snapshot = _decision_log_snapshot(
            DecisionResponse(
                action="local_code_inspection",
                reason="matched",
                agent_reports=[
                    AgentReport(
                        agent_name="local_code_agent",
                        action="local_code_inspection",
                        reason="matched",
                        evidence=[
                            {
                                "file_path": "src/main/java/FoodServiceImpl.java",
                                "primary_symbol": {"name": "FoodServiceImpl.generate", "line_number": 10},
                                "line_range": {"start": 10, "end": 15},
                                "line_numbers": [12],
                                "code_excerpt": [{"line_number": 12, "text": "x" * 500}],
                                "suspect_reasons": ["reason"],
                                "follow_up_checks": ["check"],
                            }
                        ],
                    )
                ],
                verification=VerificationReport(accepted=True, reason="ok"),
            )
        )

        evidence = snapshot["agent_reports"][0]["evidence"][0]
        self.assertNotIn("code_excerpt", evidence)
        self.assertEqual(evidence["code_excerpt_line_count"], 1)
        self.assertIn("primary_symbol", evidence)


if __name__ == "__main__":
    unittest.main()
