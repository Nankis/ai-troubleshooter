from __future__ import annotations

import unittest

from agent_platform.service import _mask, _requires_realtime


class ServiceHelperTest(unittest.TestCase):
    def test_explicit_date_requires_realtime_check(self) -> None:
        self.assertTrue(_requires_realtime({"original_text": "health-food uid 1 2026-05-23 推荐来源错配"}))

    def test_requesting_real_data_requires_realtime_check(self) -> None:
        self.assertTrue(_requires_realtime({"original_text": "请查真实数据，不要只用平台经验"}))

    def test_mask_redacts_token_balance_in_summary_text(self) -> None:
        self.assertEqual(
            _mask("health-food ai quota normal: tokens=123456.000000 daily_chat=996/1000"),
            "health-food ai quota normal: tokens=<redacted> daily_chat=996/1000",
        )


if __name__ == "__main__":
    unittest.main()
