package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
)

type Store struct {
	db *sql.DB
}

func New(ctx context.Context, dsn string) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("mysql dsn is required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateCase(ctx context.Context, input caseflow.CreateCaseInput) (*caseflow.Case, error) {
	now := time.Now()
	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	tmpNo := fmt.Sprintf("case_pending_%d", now.UnixNano())
	res, err := tx.ExecContext(ctx, `
INSERT INTO cases
(case_no, source, chat_id, thread_id, message_id, reporter_user_id, original_text, ocr_text, status, priority, timezone, created_at, updated_at, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		tmpNo, fallback(input.Source, "lark"), nullableString(input.ChatID), nullableString(input.ThreadID), nullableString(input.MessageID),
		nullableString(input.ReporterUserID), nullableString(input.OriginalText), nullableString(input.OCRText),
		caseflow.StatusNew, "normal", tz, now, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	caseNo := fmt.Sprintf("case_%s_%06d", now.Format("20060102"), id)
	if _, err := tx.ExecContext(ctx, `UPDATE cases SET case_no = ? WHERE id = ?`, caseNo, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetCase(ctx, id)
}

func (s *Store) GetCase(ctx context.Context, id int64) (*caseflow.Case, error) {
	row := s.db.QueryRowContext(ctx, caseSelect()+` WHERE id = ?`, id)
	return scanCase(row)
}

func (s *Store) FindCaseByNo(ctx context.Context, caseNo string) (*caseflow.Case, error) {
	row := s.db.QueryRowContext(ctx, caseSelect()+` WHERE case_no = ?`, caseNo)
	return scanCase(row)
}

func (s *Store) UpdateCase(ctx context.Context, id int64, expectedVersion int64, update func(*caseflow.Case) error) (*caseflow.Case, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	current, err := scanCase(tx.QueryRowContext(ctx, caseSelect()+` WHERE id = ? FOR UPDATE`, id))
	if err != nil {
		return nil, err
	}
	if current.Version != expectedVersion {
		return nil, caseflow.ErrVersionConflict
	}
	next := *current
	if err := update(&next); err != nil {
		return nil, err
	}
	next.Version++
	next.UpdatedAt = time.Now()
	if _, err := tx.ExecContext(ctx, `
UPDATE cases
SET source = ?, chat_id = ?, thread_id = ?, message_id = ?, reporter_user_id = ?, original_text = ?, ocr_text = ?,
    issue_domain = ?, issue_type = ?, status = ?, priority = ?, timezone = ?, occurred_at = ?, updated_at = ?, closed_at = ?, version = ?
WHERE id = ?`,
		next.Source, nullableString(next.ChatID), nullableString(next.ThreadID), nullableString(next.MessageID), nullableString(next.ReporterUserID),
		nullableString(next.OriginalText), nullableString(next.OCRText), nullableString(next.IssueDomain), nullableString(next.IssueType),
		next.Status, next.Priority, next.Timezone, nullableTime(next.OccurredAt), next.UpdatedAt, nullableTime(next.ClosedAt), next.Version, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetCase(ctx, id)
}

func (s *Store) AddEntities(ctx context.Context, caseID int64, entities []caseflow.Entity) error {
	if len(entities) == 0 {
		return nil
	}
	now := time.Now()
	for _, entity := range entities {
		if entity.Type == "" || entity.Value == "" {
			continue
		}
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM case_entities WHERE case_id = ? AND entity_type = ? AND entity_value = ?`, caseID, entity.Type, entity.Value).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		_, err := s.db.ExecContext(ctx, `
INSERT INTO case_entities (case_id, entity_type, entity_value, source, confidence, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
			caseID, entity.Type, entity.Value, fallback(entity.Source, "llm"), nullableFloat(entity.Confidence), now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListEntities(ctx context.Context, caseID int64) ([]caseflow.Entity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, case_id, entity_type, entity_value, source, confidence, created_at FROM case_entities WHERE case_id = ? ORDER BY id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []caseflow.Entity{}
	for rows.Next() {
		var entity caseflow.Entity
		var confidence sql.NullFloat64
		if err := rows.Scan(&entity.ID, &entity.CaseID, &entity.Type, &entity.Value, &entity.Source, &confidence, &entity.CreatedAt); err != nil {
			return nil, err
		}
		if confidence.Valid {
			entity.Confidence = &confidence.Float64
		}
		out = append(out, entity)
	}
	return out, rows.Err()
}

func (s *Store) AddMessage(ctx context.Context, msg caseflow.Message) (caseflow.Message, error) {
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO case_messages (case_id, role, lark_message_id, content, content_type, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		msg.CaseID, msg.Role, nullableString(msg.LarkMessageID), msg.Content, fallback(msg.ContentType, "text"), now)
	if err != nil {
		return caseflow.Message{}, err
	}
	id, _ := res.LastInsertId()
	msg.ID = id
	msg.CreatedAt = now
	if msg.ContentType == "" {
		msg.ContentType = "text"
	}
	return msg, nil
}

func (s *Store) ListMessages(ctx context.Context, caseID int64) ([]caseflow.Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, case_id, role, lark_message_id, content, content_type, created_at FROM case_messages WHERE case_id = ? ORDER BY id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []caseflow.Message{}
	for rows.Next() {
		var msg caseflow.Message
		var larkMessageID sql.NullString
		if err := rows.Scan(&msg.ID, &msg.CaseID, &msg.Role, &larkMessageID, &msg.Content, &msg.ContentType, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msg.LarkMessageID = nullStringValue(larkMessageID)
		out = append(out, msg)
	}
	return out, rows.Err()
}

func (s *Store) CreateInvestigation(ctx context.Context, inv caseflow.Investigation) (caseflow.Investigation, error) {
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return caseflow.Investigation{}, err
	}
	defer rollback(tx)
	tmpNo := fmt.Sprintf("inv_pending_%d", now.UnixNano())
	res, err := tx.ExecContext(ctx, `
INSERT INTO investigations
(investigation_no, case_id, agent_id, agent_version, model_provider, model_name, status, initial_hypothesis, started_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tmpNo, inv.CaseID, inv.AgentID, nullableString(inv.AgentVersion), nullableString(inv.ModelProvider), nullableString(inv.ModelName),
		fallback(inv.Status, "running"), nullableString(inv.InitialHypothesis), now, now, now)
	if err != nil {
		return caseflow.Investigation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return caseflow.Investigation{}, err
	}
	invNo := fmt.Sprintf("inv_%s_%06d", now.Format("20060102"), id)
	if _, err := tx.ExecContext(ctx, `UPDATE investigations SET investigation_no = ? WHERE id = ?`, invNo, id); err != nil {
		return caseflow.Investigation{}, err
	}
	if err := tx.Commit(); err != nil {
		return caseflow.Investigation{}, err
	}
	inv.ID = id
	inv.InvestigationNo = invNo
	inv.StartedAt = now
	inv.CreatedAt = now
	inv.UpdatedAt = now
	if inv.Status == "" {
		inv.Status = "running"
	}
	return inv, nil
}

func (s *Store) FinishInvestigation(ctx context.Context, id int64, status string, summary string, confidence *float64) (caseflow.Investigation, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
UPDATE investigations SET status = ?, final_summary = ?, confidence = ?, finished_at = ?, updated_at = ? WHERE id = ?`,
		status, nullableString(summary), nullableFloat(confidence), now, now, id)
	if err != nil {
		return caseflow.Investigation{}, err
	}
	return s.getInvestigation(ctx, id)
}

func (s *Store) AddAIDecisionLog(ctx context.Context, item caseflow.AIDecisionLog) (caseflow.AIDecisionLog, error) {
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.Status == "" {
		item.Status = "success"
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO ai_decision_logs
(case_id, investigation_id, agent_id, decision_type, reason, input_snapshot_json, output_snapshot_json, selected_tools_json, status, latency_ms, error_message, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.CaseID,
		nullableInt64(item.InvestigationID),
		item.AgentID,
		item.DecisionType,
		nullableString(item.Reason),
		nullableString(item.InputSnapshotJSON),
		nullableString(item.OutputSnapshotJSON),
		nullableString(item.SelectedToolsJSON),
		item.Status,
		item.LatencyMS,
		nullableString(item.ErrorMessage),
		item.CreatedAt,
	)
	if err != nil {
		return caseflow.AIDecisionLog{}, err
	}
	id, _ := res.LastInsertId()
	item.ID = id
	return item, nil
}

func (s *Store) ListAIDecisionLogs(ctx context.Context, caseID int64, limit int) ([]caseflow.AIDecisionLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, case_id, investigation_id, agent_id, decision_type, reason, input_snapshot_json, output_snapshot_json,
       selected_tools_json, status, latency_ms, error_message, created_at
FROM ai_decision_logs
WHERE case_id = ?
ORDER BY id DESC
LIMIT ?`, caseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []caseflow.AIDecisionLog{}
	for rows.Next() {
		item, err := scanAIDecisionLogRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s *Store) UpsertRootCause(ctx context.Context, rootCause caseflow.RootCause) (caseflow.RootCause, error) {
	now := time.Now()
	if rootCause.ConfirmedAt.IsZero() {
		rootCause.ConfirmedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO root_causes
(case_id, ai_predicted_reason, human_confirmed_reason, root_cause_category, owner_service, owner_team,
 is_cache_issue, is_data_sync_issue, is_external_source_issue, is_frontend_display_issue, is_user_misunderstanding,
 fix_action, prevention_action, confirmed_by, confirmed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
 ai_predicted_reason = VALUES(ai_predicted_reason),
 human_confirmed_reason = VALUES(human_confirmed_reason),
 root_cause_category = VALUES(root_cause_category),
 owner_service = VALUES(owner_service),
 owner_team = VALUES(owner_team),
 is_cache_issue = VALUES(is_cache_issue),
 is_data_sync_issue = VALUES(is_data_sync_issue),
 is_external_source_issue = VALUES(is_external_source_issue),
 is_frontend_display_issue = VALUES(is_frontend_display_issue),
 is_user_misunderstanding = VALUES(is_user_misunderstanding),
 fix_action = VALUES(fix_action),
 prevention_action = VALUES(prevention_action),
 confirmed_by = VALUES(confirmed_by),
 confirmed_at = VALUES(confirmed_at),
 updated_at = VALUES(updated_at)`,
		rootCause.CaseID, nullableString(rootCause.AIPredictedReason), rootCause.HumanConfirmedReason, rootCause.RootCauseCategory,
		nullableString(rootCause.OwnerService), nullableString(rootCause.OwnerTeam), rootCause.IsCacheIssue, rootCause.IsDataSyncIssue,
		rootCause.IsExternalSourceIssue, rootCause.IsFrontendDisplayIssue, rootCause.IsUserMisunderstanding,
		nullableString(rootCause.FixAction), nullableString(rootCause.PreventionAction), nullableString(rootCause.ConfirmedBy),
		rootCause.ConfirmedAt, now, now)
	if err != nil {
		return caseflow.RootCause{}, err
	}
	return s.GetRootCause(ctx, rootCause.CaseID)
}

func (s *Store) GetRootCause(ctx context.Context, caseID int64) (caseflow.RootCause, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, case_id, ai_predicted_reason, human_confirmed_reason, root_cause_category, owner_service, owner_team,
       is_cache_issue, is_data_sync_issue, is_external_source_issue, is_frontend_display_issue, is_user_misunderstanding,
       fix_action, prevention_action, confirmed_by, confirmed_at, created_at, updated_at
FROM root_causes WHERE case_id = ?`, caseID)
	return scanRootCause(row)
}

func (s *Store) AddCaseFeedback(ctx context.Context, feedback caseflow.CaseFeedback) (caseflow.CaseFeedback, error) {
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO case_feedbacks
(case_id, rating, ai_useful, wrong_conclusion, missing_key_information, missing_tools_json, comment, created_by, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feedback.CaseID, nullableInt(feedback.Rating), feedback.AIUseful, feedback.WrongConclusion, nullableString(feedback.MissingKeyInformation),
		nullableString(feedback.MissingToolsJSON), nullableString(feedback.Comment), nullableString(feedback.CreatedBy), now)
	if err != nil {
		return caseflow.CaseFeedback{}, err
	}
	id, _ := res.LastInsertId()
	feedback.ID = id
	feedback.CreatedAt = now
	return feedback, nil
}

func (s *Store) ListCaseFeedback(ctx context.Context, caseID int64) ([]caseflow.CaseFeedback, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, case_id, rating, ai_useful, wrong_conclusion, missing_key_information, missing_tools_json, comment, created_by, created_at
FROM case_feedbacks WHERE case_id = ? ORDER BY id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []caseflow.CaseFeedback{}
	for rows.Next() {
		item, err := scanFeedbackRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpsertKnowledgeItem(ctx context.Context, item caseflow.KnowledgeItem) (caseflow.KnowledgeItem, error) {
	if item.ID == 0 {
		existing, err := s.FindKnowledgeItem(ctx, item.IssueDomain, item.IssueType, item.LastRootCauseCategory)
		if err == nil {
			item.ID = existing.ID
			item.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, caseflow.ErrNotFound) {
			return caseflow.KnowledgeItem{}, err
		}
	}
	now := time.Now()
	if item.LastEvolvedAt.IsZero() {
		item.LastEvolvedAt = now
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.ID != 0 {
		_, err := s.db.ExecContext(ctx, knowledgeUpdateSQL(), knowledgeArgs(item, now, item.ID)...)
		if err != nil {
			return caseflow.KnowledgeItem{}, err
		}
		return s.GetKnowledgeItem(ctx, item.ID)
	}
	res, err := s.db.ExecContext(ctx, knowledgeInsertSQL(), knowledgeInsertArgs(item, now)...)
	if err != nil {
		return caseflow.KnowledgeItem{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetKnowledgeItem(ctx, id)
}

func (s *Store) GetKnowledgeItem(ctx context.Context, id int64) (caseflow.KnowledgeItem, error) {
	row := s.db.QueryRowContext(ctx, knowledgeSelect()+` WHERE id = ?`, id)
	return scanKnowledgeItem(row)
}

func (s *Store) FindKnowledgeItem(ctx context.Context, issueDomain string, issueType string, rootCauseCategory string) (caseflow.KnowledgeItem, error) {
	row := s.db.QueryRowContext(ctx, knowledgeSelect()+` WHERE issue_domain = ? AND issue_type <=> ? AND last_root_cause_category <=> ? LIMIT 1`,
		issueDomain, nullableString(issueType), nullableString(rootCauseCategory))
	return scanKnowledgeItem(row)
}

func (s *Store) ListKnowledgeItems(ctx context.Context, filter caseflow.KnowledgeFilter) ([]caseflow.KnowledgeItem, error) {
	query := knowledgeSelect()
	conds := []string{"1=1"}
	args := []any{}
	if filter.IssueDomain != "" {
		conds = append(conds, "issue_domain = ?")
		args = append(args, filter.IssueDomain)
	}
	if filter.IssueType != "" {
		conds = append(conds, "issue_type = ?")
		args = append(args, filter.IssueType)
	}
	if filter.RootCauseCategory != "" {
		conds = append(conds, "last_root_cause_category = ?")
		args = append(args, filter.RootCauseCategory)
	}
	if filter.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, filter.Status)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query += " WHERE " + strings.Join(conds, " AND ") + " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []caseflow.KnowledgeItem{}
	for rows.Next() {
		item, err := scanKnowledgeRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateKnowledgeEvolutionRun(ctx context.Context, run caseflow.KnowledgeEvolutionRun) (caseflow.KnowledgeEvolutionRun, error) {
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return caseflow.KnowledgeEvolutionRun{}, err
	}
	defer rollback(tx)
	tmpNo := fmt.Sprintf("ke_pending_%d", now.UnixNano())
	res, err := tx.ExecContext(ctx, `
INSERT INTO knowledge_evolution_runs
(run_no, case_id, knowledge_item_id, trigger_type, input_snapshot_json, output_summary, decision, created_knowledge_item, updated_knowledge_item, error_message, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tmpNo, run.CaseID, nullableInt64(run.KnowledgeItemID), run.TriggerType, run.InputSnapshotJSON, nullableString(run.OutputSummary),
		run.Decision, run.CreatedKnowledgeItem, run.UpdatedKnowledgeItem, nullableString(run.ErrorMessage), now)
	if err != nil {
		return caseflow.KnowledgeEvolutionRun{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return caseflow.KnowledgeEvolutionRun{}, err
	}
	runNo := fmt.Sprintf("ke_%s_%06d", now.Format("20060102"), id)
	if _, err := tx.ExecContext(ctx, `UPDATE knowledge_evolution_runs SET run_no = ? WHERE id = ?`, runNo, id); err != nil {
		return caseflow.KnowledgeEvolutionRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return caseflow.KnowledgeEvolutionRun{}, err
	}
	run.ID = id
	run.RunNo = runNo
	run.CreatedAt = now
	return run, nil
}

func (s *Store) ListKnowledgeEvolutionRuns(ctx context.Context, caseID int64) ([]caseflow.KnowledgeEvolutionRun, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, run_no, case_id, knowledge_item_id, trigger_type, input_snapshot_json, output_summary, decision,
       created_knowledge_item, updated_knowledge_item, error_message, created_at
FROM knowledge_evolution_runs WHERE case_id = ? ORDER BY id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []caseflow.KnowledgeEvolutionRun{}
	for rows.Next() {
		item, err := scanEvolutionRunRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func caseSelect() string {
	return `SELECT id, case_no, source, chat_id, thread_id, message_id, reporter_user_id, original_text, ocr_text,
issue_domain, issue_type, status, priority, timezone, occurred_at, created_at, updated_at, closed_at, version FROM cases`
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCase(row scanner) (*caseflow.Case, error) {
	var c caseflow.Case
	var chatID, threadID, messageID, reporterUserID, originalText, ocrText, issueDomain, issueType sql.NullString
	var occurredAt, closedAt sql.NullTime
	if err := row.Scan(&c.ID, &c.CaseNo, &c.Source, &chatID, &threadID, &messageID, &reporterUserID, &originalText, &ocrText,
		&issueDomain, &issueType, &c.Status, &c.Priority, &c.Timezone, &occurredAt, &c.CreatedAt, &c.UpdatedAt, &closedAt, &c.Version); err != nil {
		return nil, normalizeNotFound(err)
	}
	c.ChatID = nullStringValue(chatID)
	c.ThreadID = nullStringValue(threadID)
	c.MessageID = nullStringValue(messageID)
	c.ReporterUserID = nullStringValue(reporterUserID)
	c.OriginalText = nullStringValue(originalText)
	c.OCRText = nullStringValue(ocrText)
	c.IssueDomain = nullStringValue(issueDomain)
	c.IssueType = nullStringValue(issueType)
	if occurredAt.Valid {
		c.OccurredAt = &occurredAt.Time
	}
	if closedAt.Valid {
		c.ClosedAt = &closedAt.Time
	}
	return &c, nil
}

func (s *Store) getInvestigation(ctx context.Context, id int64) (caseflow.Investigation, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, investigation_no, case_id, agent_id, agent_version, model_provider, model_name, status, initial_hypothesis,
       final_summary, confidence, started_at, finished_at, created_at, updated_at
FROM investigations WHERE id = ?`, id)
	var inv caseflow.Investigation
	var agentVersion, modelProvider, modelName, initialHypothesis, finalSummary sql.NullString
	var confidence sql.NullFloat64
	var finishedAt sql.NullTime
	if err := row.Scan(&inv.ID, &inv.InvestigationNo, &inv.CaseID, &inv.AgentID, &agentVersion, &modelProvider, &modelName, &inv.Status,
		&initialHypothesis, &finalSummary, &confidence, &inv.StartedAt, &finishedAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return caseflow.Investigation{}, normalizeNotFound(err)
	}
	inv.AgentVersion = nullStringValue(agentVersion)
	inv.ModelProvider = nullStringValue(modelProvider)
	inv.ModelName = nullStringValue(modelName)
	inv.InitialHypothesis = nullStringValue(initialHypothesis)
	inv.FinalSummary = nullStringValue(finalSummary)
	if confidence.Valid {
		inv.Confidence = &confidence.Float64
	}
	if finishedAt.Valid {
		inv.FinishedAt = &finishedAt.Time
	}
	return inv, nil
}

func scanRootCause(row scanner) (caseflow.RootCause, error) {
	var rc caseflow.RootCause
	var aiReason, ownerService, ownerTeam, fixAction, preventionAction, confirmedBy sql.NullString
	if err := row.Scan(&rc.ID, &rc.CaseID, &aiReason, &rc.HumanConfirmedReason, &rc.RootCauseCategory, &ownerService, &ownerTeam,
		&rc.IsCacheIssue, &rc.IsDataSyncIssue, &rc.IsExternalSourceIssue, &rc.IsFrontendDisplayIssue, &rc.IsUserMisunderstanding,
		&fixAction, &preventionAction, &confirmedBy, &rc.ConfirmedAt, &rc.CreatedAt, &rc.UpdatedAt); err != nil {
		return caseflow.RootCause{}, normalizeNotFound(err)
	}
	rc.AIPredictedReason = nullStringValue(aiReason)
	rc.OwnerService = nullStringValue(ownerService)
	rc.OwnerTeam = nullStringValue(ownerTeam)
	rc.FixAction = nullStringValue(fixAction)
	rc.PreventionAction = nullStringValue(preventionAction)
	rc.ConfirmedBy = nullStringValue(confirmedBy)
	return rc, nil
}

func scanFeedbackRows(rows *sql.Rows) (caseflow.CaseFeedback, error) {
	var item caseflow.CaseFeedback
	var rating sql.NullInt64
	var missingInfo, missingTools, comment, createdBy sql.NullString
	if err := rows.Scan(&item.ID, &item.CaseID, &rating, &item.AIUseful, &item.WrongConclusion, &missingInfo, &missingTools, &comment, &createdBy, &item.CreatedAt); err != nil {
		return caseflow.CaseFeedback{}, err
	}
	if rating.Valid {
		item.Rating = int(rating.Int64)
	}
	item.MissingKeyInformation = nullStringValue(missingInfo)
	item.MissingToolsJSON = nullStringValue(missingTools)
	item.Comment = nullStringValue(comment)
	item.CreatedBy = nullStringValue(createdBy)
	return item, nil
}

func scanAIDecisionLogRows(rows *sql.Rows) (caseflow.AIDecisionLog, error) {
	var item caseflow.AIDecisionLog
	var investigationID sql.NullInt64
	var reason, inputSnapshot, outputSnapshot, selectedTools, errorMessage sql.NullString
	if err := rows.Scan(&item.ID, &item.CaseID, &investigationID, &item.AgentID, &item.DecisionType, &reason,
		&inputSnapshot, &outputSnapshot, &selectedTools, &item.Status, &item.LatencyMS, &errorMessage, &item.CreatedAt); err != nil {
		return caseflow.AIDecisionLog{}, err
	}
	if investigationID.Valid {
		item.InvestigationID = investigationID.Int64
	}
	item.Reason = nullStringValue(reason)
	item.InputSnapshotJSON = nullStringValue(inputSnapshot)
	item.OutputSnapshotJSON = nullStringValue(outputSnapshot)
	item.SelectedToolsJSON = nullStringValue(selectedTools)
	item.ErrorMessage = nullStringValue(errorMessage)
	return item, nil
}

func knowledgeSelect() string {
	return `SELECT id, title, issue_domain, issue_type, typical_description, typical_ocr_features, required_fields_json,
recommended_steps_json, common_causes_json, useful_tools_json, success_case_ids_json, failure_case_ids_json,
confidence, status, observed_case_count, last_root_cause_category, last_confirmed_reason, last_evolved_at, created_at, updated_at FROM knowledge_items`
}

func scanKnowledgeItem(row scanner) (caseflow.KnowledgeItem, error) {
	var item caseflow.KnowledgeItem
	var issueType, typicalDescription, typicalOCR, requiredFields, steps, causes, tools, successIDs, failureIDs sql.NullString
	var rootCategory, confirmedReason sql.NullString
	var confidence sql.NullFloat64
	var lastEvolvedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.Title, &item.IssueDomain, &issueType, &typicalDescription, &typicalOCR, &requiredFields,
		&steps, &causes, &tools, &successIDs, &failureIDs, &confidence, &item.Status, &item.ObservedCaseCount, &rootCategory,
		&confirmedReason, &lastEvolvedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return caseflow.KnowledgeItem{}, normalizeNotFound(err)
	}
	item.IssueType = nullStringValue(issueType)
	item.TypicalDescription = nullStringValue(typicalDescription)
	item.TypicalOCRFeatures = nullStringValue(typicalOCR)
	item.RequiredFieldsJSON = nullStringValue(requiredFields)
	item.RecommendedStepsJSON = nullStringValue(steps)
	item.CommonCausesJSON = nullStringValue(causes)
	item.UsefulToolsJSON = nullStringValue(tools)
	item.SuccessCaseIDsJSON = nullStringValue(successIDs)
	item.FailureCaseIDsJSON = nullStringValue(failureIDs)
	if confidence.Valid {
		item.Confidence = confidence.Float64
	}
	item.LastRootCauseCategory = nullStringValue(rootCategory)
	item.LastConfirmedReason = nullStringValue(confirmedReason)
	if lastEvolvedAt.Valid {
		item.LastEvolvedAt = lastEvolvedAt.Time
	}
	return item, nil
}

func scanKnowledgeRows(rows *sql.Rows) (caseflow.KnowledgeItem, error) {
	return scanKnowledgeItem(rows)
}

func scanEvolutionRunRows(rows *sql.Rows) (caseflow.KnowledgeEvolutionRun, error) {
	var run caseflow.KnowledgeEvolutionRun
	var knowledgeItemID sql.NullInt64
	var outputSummary, errorMessage sql.NullString
	if err := rows.Scan(&run.ID, &run.RunNo, &run.CaseID, &knowledgeItemID, &run.TriggerType, &run.InputSnapshotJSON, &outputSummary,
		&run.Decision, &run.CreatedKnowledgeItem, &run.UpdatedKnowledgeItem, &errorMessage, &run.CreatedAt); err != nil {
		return caseflow.KnowledgeEvolutionRun{}, err
	}
	if knowledgeItemID.Valid {
		run.KnowledgeItemID = knowledgeItemID.Int64
	}
	run.OutputSummary = nullStringValue(outputSummary)
	run.ErrorMessage = nullStringValue(errorMessage)
	return run, nil
}

func knowledgeInsertSQL() string {
	return `INSERT INTO knowledge_items
(title, issue_domain, issue_type, typical_description, typical_ocr_features, required_fields_json, recommended_steps_json,
 common_causes_json, useful_tools_json, success_case_ids_json, failure_case_ids_json, confidence, status, observed_case_count,
 last_root_cause_category, last_confirmed_reason, last_evolved_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
}

func knowledgeInsertArgs(item caseflow.KnowledgeItem, now time.Time) []any {
	return []any{
		item.Title, item.IssueDomain, nullableString(item.IssueType), nullableString(item.TypicalDescription), nullableString(item.TypicalOCRFeatures),
		nullableString(item.RequiredFieldsJSON), nullableString(item.RecommendedStepsJSON), nullableString(item.CommonCausesJSON),
		nullableString(item.UsefulToolsJSON), nullableString(item.SuccessCaseIDsJSON), nullableString(item.FailureCaseIDsJSON),
		item.Confidence, item.Status, item.ObservedCaseCount, nullableString(item.LastRootCauseCategory), nullableString(item.LastConfirmedReason),
		nullableTimeValue(item.LastEvolvedAt), now, now,
	}
}

func knowledgeUpdateSQL() string {
	return `UPDATE knowledge_items SET
title = ?, issue_domain = ?, issue_type = ?, typical_description = ?, typical_ocr_features = ?, required_fields_json = ?,
recommended_steps_json = ?, common_causes_json = ?, useful_tools_json = ?, success_case_ids_json = ?, failure_case_ids_json = ?,
confidence = ?, status = ?, observed_case_count = ?, last_root_cause_category = ?, last_confirmed_reason = ?, last_evolved_at = ?, updated_at = ?
WHERE id = ?`
}

func knowledgeArgs(item caseflow.KnowledgeItem, now time.Time, id int64) []any {
	args := knowledgeInsertArgs(item, now)
	out := append([]any{}, args[:17]...)
	out = append(out, now, id)
	return out
}

func normalizeNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return caseflow.ErrNotFound
	}
	return err
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func nullableTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return *t
}

func nullableTimeValue(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullableInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
