from __future__ import annotations

from dataclasses import dataclass


CLAIMABLE_STATUSES = {"NEW", "NEED_MORE_INFO", "WAITING_USER_REPLY", "READY_TO_INVESTIGATE"}
ACTIVE_STATUSES = {"INVESTIGATING", "WAITING_TOOL_RESULT"}
TERMINAL_STATUSES = {"NEED_HUMAN_CONFIRMATION", "DONE", "FAILED"}


@dataclass(frozen=True, slots=True)
class ScheduleDecision:
    accepted: bool
    event_type: str
    next_status: str
    reason: str


class CaseScheduler:
    """Minimal synchronous scheduler state machine for case processing."""

    def claim(self, current_status: str) -> ScheduleDecision:
        status = str(current_status or "").strip() or "NEW"
        if status in CLAIMABLE_STATUSES:
            return ScheduleDecision(
                accepted=True,
                event_type="scheduler_claimed",
                next_status="READY_TO_INVESTIGATE",
                reason=f"case status {status} is claimable",
            )
        return ScheduleDecision(
            accepted=False,
            event_type="scheduler_rejected",
            next_status=status,
            reason=f"case status {status} is not claimable",
        )

    def finish(self, final_status: str, *, failed: bool = False, timed_out: bool = False) -> ScheduleDecision:
        status = str(final_status or "").strip() or "UNKNOWN"
        if timed_out:
            return ScheduleDecision(False, "scheduler_timed_out", "FAILED", "case processing exceeded timeout")
        if failed:
            return ScheduleDecision(False, "scheduler_failed", "FAILED", "case processing failed")
        if status in TERMINAL_STATUSES or status in {"NEED_MORE_INFO", "WAITING_USER_REPLY"}:
            return ScheduleDecision(True, "scheduler_finished", status, f"case finished with status {status}")
        if status in ACTIVE_STATUSES:
            return ScheduleDecision(False, "scheduler_still_active", status, f"case remains active with status {status}")
        return ScheduleDecision(True, "scheduler_finished", status, f"case finished with status {status}")
