from __future__ import annotations

import unittest

from agent_platform.service import _mask


class ServiceHelperTest(unittest.TestCase):
    def test_mask_redacts_token_balance_in_summary_text(self) -> None:
        self.assertEqual(
            _mask("health-food ai quota normal: tokens=123456.000000 daily_chat=996/1000"),
            "health-food ai quota normal: tokens=<redacted> daily_chat=996/1000",
        )


if __name__ == "__main__":
    unittest.main()
