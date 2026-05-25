import unittest

from agent_platform.case_scheduler import CaseScheduler


class CaseSchedulerTest(unittest.TestCase):
    def test_claim_allows_ready_case(self) -> None:
        decision = CaseScheduler().claim("READY_TO_INVESTIGATE")

        self.assertTrue(decision.accepted)
        self.assertEqual(decision.event_type, "scheduler_claimed")
        self.assertEqual(decision.next_status, "READY_TO_INVESTIGATE")

    def test_claim_rejects_active_case(self) -> None:
        decision = CaseScheduler().claim("INVESTIGATING")

        self.assertFalse(decision.accepted)
        self.assertEqual(decision.event_type, "scheduler_rejected")
        self.assertEqual(decision.next_status, "INVESTIGATING")

    def test_finish_records_failure_and_timeout(self) -> None:
        scheduler = CaseScheduler()

        failed = scheduler.finish("INVESTIGATING", failed=True)
        timed_out = scheduler.finish("INVESTIGATING", timed_out=True)

        self.assertEqual(failed.event_type, "scheduler_failed")
        self.assertEqual(failed.next_status, "FAILED")
        self.assertEqual(timed_out.event_type, "scheduler_timed_out")
        self.assertEqual(timed_out.next_status, "FAILED")


if __name__ == "__main__":
    unittest.main()
