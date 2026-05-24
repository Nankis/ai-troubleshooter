from __future__ import annotations

import unittest

from agent_platform.classifier import classify_and_extract_rules, merge_llm_result


class ClassifierMergeTest(unittest.TestCase):
    def test_qwen_health_food_domain_alias_keeps_specialist_route(self) -> None:
        merged = merge_llm_result(
            {"issue_domain": "health_food", "issue_type": "AI配额异常", "entities": {"uid": "2054603630081875968"}},
            {
                "issue_domain": "health-food",
                "issue_type": "token次数异常",
                "entities": {"service_name": "health-food"},
                "confidence": 0.9,
            },
        )

        self.assertEqual(merged["issue_domain"], "health_food")
        self.assertEqual(merged["issue_type"], "AI配额异常")
        self.assertEqual(merged["entities"]["uid"], "2054603630081875968")
        self.assertEqual(merged["entities"]["service_name"], "health-food")

    def test_unsupported_llm_domain_does_not_override_rule_domain(self) -> None:
        merged = merge_llm_result(
            {"issue_domain": "health_food", "issue_type": "AI配额异常", "entities": {"uid": "2054603630081875968"}},
            {
                "issue_domain": "token_usage",
                "issue_type": "quota_count_abnormal",
                "entities": {"service_name": "health-food"},
                "confidence": 0.9,
            },
        )

        self.assertEqual(merged["issue_domain"], "health_food")
        self.assertEqual(merged["issue_type"], "AI配额异常")

    def test_generic_llm_issue_type_does_not_override_rule_taxonomy(self) -> None:
        merged = merge_llm_result(
            {"issue_domain": "health_food", "issue_type": "餐食数据异常", "entities": {"service_name": "health-food"}},
            {
                "issue_domain": "health-food",
                "issue_type": "数据异常",
                "entities": {"focus": "餐食记录重复导致热量翻倍"},
                "confidence": 0.95,
            },
        )

        self.assertEqual(merged["issue_domain"], "health_food")
        self.assertEqual(merged["issue_type"], "餐食数据异常")

    def test_debug_local_code_entities_are_explicitly_extracted(self) -> None:
        result = classify_and_extract_rules(
            "health-food uid 2054603630081875968 debug_local_code=true "
            "gateway_evidence_status=insufficient service_name=health-food "
            "suspect_area=RecommendFoodJob"
        )

        entities = result["entities"]
        self.assertEqual(entities["debug_local_code"], "true")
        self.assertEqual(entities["gateway_evidence_status"], "insufficient")
        self.assertEqual(entities["service_name"], "health-food")
        self.assertEqual(entities["suspect_area"], "RecommendFoodJob")

    def test_health_food_explicit_date_maps_to_recommendation_date(self) -> None:
        result = classify_and_extract_rules("health-food uid 2054603630081875968 2026-05-23 推荐数据不准")

        entities = result["entities"]
        self.assertEqual(entities["recommendation_date"], "2026-05-23")

    def test_qwen_date_alias_maps_to_recommendation_date(self) -> None:
        merged = merge_llm_result(
            {"issue_domain": "health_food", "issue_type": "每日推荐不准", "entities": {"uid": "2054603630081875968"}},
            {"issue_domain": "health-food", "entities": {"date": "2026-05-23"}},
        )

        self.assertEqual(merged["entities"]["recommendation_date"], "2026-05-23")


if __name__ == "__main__":
    unittest.main()
