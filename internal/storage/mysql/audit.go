package mysql

import (
	"context"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/audit"
)

func (s *Store) Record(ctx context.Context, record audit.Record) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tool_call_audits
(tool_call_id, case_ref, investigation_ref, agent_id, lark_user_id, tool_name, required_scope, arguments_summary,
 policy_decision, deny_reason, query_id, result_count, latency_ms, error_message, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
 case_ref = VALUES(case_ref),
 investigation_ref = VALUES(investigation_ref),
 agent_id = VALUES(agent_id),
 lark_user_id = VALUES(lark_user_id),
 tool_name = VALUES(tool_name),
 required_scope = VALUES(required_scope),
 arguments_summary = VALUES(arguments_summary),
 policy_decision = VALUES(policy_decision),
 deny_reason = VALUES(deny_reason),
 query_id = VALUES(query_id),
 result_count = VALUES(result_count),
 latency_ms = VALUES(latency_ms),
 error_message = VALUES(error_message)`,
		record.ToolCallID,
		record.CaseID,
		nullableString(record.InvestigationID),
		record.AgentID,
		nullableString(record.LarkUserID),
		record.ToolName,
		nullableString(record.RequiredScope),
		nullableString(record.ArgumentsSummary),
		record.PolicyDecision,
		nullableString(record.DenyReason),
		nullableString(record.QueryID),
		nullableInt(record.ResultCount),
		record.LatencyMS,
		nullableString(record.ErrorMessage),
		record.CreatedAt,
	)
	return err
}
