package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/llm"
	"github.com/Nankis/ai-troubleshooter/internal/masking"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
)

type ToolClient interface {
	Invoke(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error)
}

type Config struct {
	AgentID                 string
	ModelProvider           string
	ModelName               string
	MaxToolCallsPerCase     int
	MaxToolFailuresPerCase  int
	MaxInvestigationSeconds int
}

type Orchestrator struct {
	store caseflow.Store
	llm   llm.LLMClient
	tools ToolClient
	cfg   Config
}

func New(store caseflow.Store, llmClient llm.LLMClient, toolClient ToolClient, cfg Config) *Orchestrator {
	if cfg.AgentID == "" {
		cfg.AgentID = "business-troubleshooter-v1"
	}
	if cfg.MaxToolCallsPerCase <= 0 {
		cfg.MaxToolCallsPerCase = 10
	}
	if cfg.MaxToolFailuresPerCase <= 0 {
		cfg.MaxToolFailuresPerCase = 3
	}
	if cfg.MaxInvestigationSeconds <= 0 {
		cfg.MaxInvestigationSeconds = 120
	}
	return &Orchestrator{store: store, llm: llmClient, tools: toolClient, cfg: cfg}
}

func (o *Orchestrator) ProcessCase(parent context.Context, caseID int64) (result caseflow.ProcessResult, err error) {
	ctx := parent
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok && o.cfg.MaxInvestigationSeconds > 0 {
		ctx, cancel = context.WithTimeout(parent, time.Duration(o.cfg.MaxInvestigationSeconds)*time.Second)
	}
	defer cancel()

	var c *caseflow.Case
	var inv *caseflow.Investigation
	defer func() {
		if err == nil {
			return
		}
		reason := err.Error()
		if ctx.Err() != nil {
			reason = ctx.Err().Error()
		}
		status := "failed"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			status = "timeout"
		}
		o.recordDecision(context.Background(), decisionRecord{
			Case:          c,
			Investigation: inv,
			DecisionType:  "process_failure",
			Reason:        "orchestrator stopped and finalized the case",
			Output:        map[string]any{"error": reason},
			Status:        status,
			Err:           err,
		})
		o.failRunningCase(context.Background(), c, inv, reason)
	}()

	c, err = o.store.GetCase(ctx, caseID)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	staleAfter := processingStaleAfter(o.cfg)
	switch {
	case canStartProcessing(c.Status):
		c, err = o.claimForProcessing(ctx, c)
		if err != nil {
			var skipped skipErr
			if errors.As(err, &skipped) {
				return skipped.result, nil
			}
			return caseflow.ProcessResult{}, err
		}
	case c.Status == caseflow.StatusReadyToInvestigate && isProcessingStale(c, staleAfter):
		c, err = o.refreshProcessingClaim(ctx, c)
		if err != nil {
			var skipped skipErr
			if errors.As(err, &skipped) {
				return skipped.result, nil
			}
			return caseflow.ProcessResult{}, err
		}
	case isActiveProcessingStatus(c.Status) && isProcessingStale(c, staleAfter):
		reason := fmt.Sprintf("case stayed in %s longer than %s", c.Status, staleAfter)
		o.recordDecision(ctx, decisionRecord{
			Case:         c,
			DecisionType: "process_stale_timeout",
			Reason:       reason,
			Input:        map[string]any{"case_no": c.CaseNo, "status": c.Status, "stale_after": staleAfter.String()},
			Status:       "timeout",
		})
		o.failRunningCase(ctx, c, nil, reason)
		latest, latestErr := o.store.GetCase(ctx, c.ID)
		if latestErr == nil {
			c = latest
		}
		return caseflow.ProcessResult{CaseID: c.ID, CaseNo: c.CaseNo, Status: c.Status, Reply: "[" + c.CaseNo + "] 排查已停止：" + reason}, nil
	default:
		return o.skipProcessing(ctx, c, "case is already being processed or has reached a non-entry status"), nil
	}
	messages, _ := o.store.ListMessages(ctx, c.ID)
	input := llm.CaseInput{Case: *c, Messages: messages}

	stepStart := time.Now()
	classification, err := o.llm.ClassifyIssue(ctx, input)
	o.recordDecision(ctx, decisionRecord{
		Case:         c,
		DecisionType: "classify_issue",
		Reason:       "classify issue domain and issue type from original text and OCR",
		Input:        map[string]any{"case_no": c.CaseNo, "original_text": c.OriginalText, "ocr_text": c.OCRText},
		Output:       classification,
		Status:       statusForErr(err),
		Latency:      time.Since(stepStart),
		Err:          err,
	})
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	stepStart = time.Now()
	extracted, err := o.llm.ExtractEntities(ctx, input)
	o.recordDecision(ctx, decisionRecord{
		Case:         c,
		DecisionType: "extract_entities",
		Reason:       "extract minimum troubleshooting fields before querying tools",
		Input:        map[string]any{"case_no": c.CaseNo, "original_text": c.OriginalText, "ocr_text": c.OCRText},
		Output:       extracted.Entities,
		Status:       statusForErr(err),
		Latency:      time.Since(stepStart),
		Err:          err,
	})
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	if err := o.store.AddEntities(ctx, c.ID, extracted.Entities); err != nil {
		return caseflow.ProcessResult{}, err
	}

	c, err = o.store.UpdateCase(ctx, c.ID, c.Version, func(next *caseflow.Case) error {
		next.IssueDomain = classification.IssueDomain
		next.IssueType = classification.IssueType
		return nil
	})
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	entities, _ := o.store.ListEntities(ctx, c.ID)
	entityMap := caseflow.EntityMap(entities)
	if c.IssueType != "" && entityMap["issue_type"] == "" {
		entityMap["issue_type"] = c.IssueType
		_ = o.store.AddEntities(ctx, c.ID, []caseflow.Entity{{Type: "issue_type", Value: c.IssueType, Source: "rules"}})
	}

	missing := caseflow.MissingRequiredFields(c.IssueDomain, entityMap)
	requiredStatus := "success"
	requiredReason := "all required fields are present; investigation can start"
	if len(missing) > 0 {
		requiredStatus = "need_more_info"
		requiredReason = "required fields are missing; ask the user before querying downstream services"
	}
	o.recordDecision(ctx, decisionRecord{
		Case:         c,
		DecisionType: "required_fields_check",
		Reason:       requiredReason,
		Input:        map[string]any{"issue_domain": c.IssueDomain, "entities": entityMap},
		Output:       map[string]any{"missing_fields": missing},
		Status:       requiredStatus,
	})
	if len(missing) > 0 {
		c, err = o.transition(ctx, c, caseflow.StatusNeedMoreInfo)
		if err != nil {
			return caseflow.ProcessResult{}, err
		}
		reply := buildMissingInfoReply(c.CaseNo, missing)
		if _, err := o.store.AddMessage(ctx, caseflow.Message{CaseID: c.ID, Role: "bot", Content: reply}); err != nil {
			return caseflow.ProcessResult{}, err
		}
		c, err = o.transition(ctx, c, caseflow.StatusWaitingUserReply)
		if err != nil {
			return caseflow.ProcessResult{}, err
		}
		return caseflow.ProcessResult{
			CaseID:        c.ID,
			CaseNo:        c.CaseNo,
			Status:        c.Status,
			Reply:         reply,
			MissingFields: missing,
		}, nil
	}

	c, err = o.transition(ctx, c, caseflow.StatusReadyToInvestigate)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	startReply := fmt.Sprintf("[%s] 信息已足够，开始排查 %s。", c.CaseNo, investigationTarget(c.IssueDomain))
	_, _ = o.store.AddMessage(ctx, caseflow.Message{CaseID: c.ID, Role: "bot", Content: startReply})
	c, err = o.transition(ctx, c, caseflow.StatusInvestigating)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}

	invValue, err := o.store.CreateInvestigation(ctx, caseflow.Investigation{
		CaseID:            c.ID,
		AgentID:           o.cfg.AgentID,
		AgentVersion:      "phase1-mvp",
		ModelProvider:     o.cfg.ModelProvider,
		ModelName:         o.cfg.ModelName,
		Status:            "running",
		InitialHypothesis: fmt.Sprintf("domain=%s issue_type=%s", c.IssueDomain, c.IssueType),
	})
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	inv = &invValue

	stepStart = time.Now()
	action, err := o.llm.DecideNextAction(ctx, *c, entityMap, nil)
	selectedTools := boundedToolNames(action.ToolNames, o.cfg.MaxToolCallsPerCase)
	o.recordDecision(ctx, decisionRecord{
		Case:          c,
		Investigation: inv,
		DecisionType:  "decide_next_action",
		Reason:        fallback(action.Reason, "choose readonly tools based on issue domain and extracted entities"),
		Input:         map[string]any{"issue_domain": c.IssueDomain, "issue_type": c.IssueType, "entities": entityMap, "max_tool_calls": o.cfg.MaxToolCallsPerCase},
		Output:        map[string]any{"requested_tools": action.ToolNames, "selected_tools": selectedTools, "truncated": len(selectedTools) < len(action.ToolNames)},
		SelectedTools: selectedTools,
		Status:        statusForErr(err),
		Latency:       time.Since(stepStart),
		Err:           err,
	})
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	c, err = o.transition(ctx, c, caseflow.StatusWaitingToolResult)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}

	observations := []llm.ToolObservation{}
	toolCallIDs := []string{}
	toolFailures := 0
	for _, name := range selectedTools {
		if ctx.Err() != nil {
			return caseflow.ProcessResult{}, ctx.Err()
		}
		if toolFailures >= o.cfg.MaxToolFailuresPerCase {
			reason := fmt.Sprintf("stopped tool queries after %d failures", toolFailures)
			observations = append(observations, llm.ToolObservation{ToolName: "tool_query_limit", Status: "stopped", Summary: reason})
			o.recordDecision(ctx, decisionRecord{
				Case:          c,
				Investigation: inv,
				DecisionType:  "tool_query_stopped",
				Reason:        reason,
				Input:         map[string]any{"max_tool_failures_per_case": o.cfg.MaxToolFailuresPerCase, "remaining_tool": name},
				Output:        map[string]any{"tool_failures": toolFailures},
				Status:        "stopped",
			})
			break
		}
		args := buildToolArgs(name, c, entityMap)
		stepStart = time.Now()
		resp, err := o.tools.Invoke(ctx, tool.InvocationRequest{
			CaseID:     c.CaseNo,
			AgentID:    o.cfg.AgentID,
			LarkUserID: c.ReporterUserID,
			ChatID:     c.ChatID,
			ToolName:   name,
			Arguments:  args,
		})
		toolCallIDs = append(toolCallIDs, resp.ToolCallID)
		status := resp.Status
		summary := resp.Summary
		if err != nil && summary == "" {
			summary = err.Error()
		}
		if status == "" {
			if err != nil {
				status = "failed"
			} else {
				status = "success"
			}
		}
		if err != nil || status != "success" {
			toolFailures++
		}
		o.recordDecision(ctx, decisionRecord{
			Case:          c,
			Investigation: inv,
			DecisionType:  "tool_invocation",
			Reason:        "invoke a registered readonly tool through gateway",
			Input:         map[string]any{"tool_name": name, "arguments": args},
			Output:        map[string]any{"tool_call_id": resp.ToolCallID, "query_id": resp.QueryID, "status": status, "summary": summary},
			SelectedTools: []string{name},
			Status:        status,
			Latency:       time.Since(stepStart),
			Err:           err,
		})
		if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return caseflow.ProcessResult{}, firstErr(err, ctx.Err())
		}
		observations = append(observations, llm.ToolObservation{ToolName: name, Summary: summary, Status: status})
	}

	stepStart = time.Now()
	report, err := o.llm.SummarizeFindings(ctx, *c, observations)
	o.recordDecision(ctx, decisionRecord{
		Case:          c,
		Investigation: inv,
		DecisionType:  "summarize_findings",
		Reason:        "summarize bounded tool observations and ask human owner for root cause confirmation",
		Input:         map[string]any{"observations": observations},
		Output:        report,
		Status:        statusForErr(err),
		Latency:       time.Since(stepStart),
		Err:           err,
	})
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	_, _ = o.store.FinishInvestigation(ctx, inv.ID, "finished", report.Summary, &report.Confidence)
	_, _ = o.store.AddMessage(ctx, caseflow.Message{CaseID: c.ID, Role: "agent", Content: "[" + c.CaseNo + "] " + report.Summary})
	c, err = o.transition(ctx, c, caseflow.StatusNeedHumanConfirmation)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	return caseflow.ProcessResult{
		CaseID:      c.ID,
		CaseNo:      c.CaseNo,
		Status:      c.Status,
		Reply:       "[" + c.CaseNo + "] " + report.Summary,
		ToolCallIDs: toolCallIDs,
	}, nil
}

type decisionRecord struct {
	Case          *caseflow.Case
	Investigation *caseflow.Investigation
	DecisionType  string
	Reason        string
	Input         any
	Output        any
	SelectedTools []string
	Status        string
	Latency       time.Duration
	Err           error
}

type skipErr struct {
	result caseflow.ProcessResult
}

func (e skipErr) Error() string {
	return "processing skipped"
}

func (o *Orchestrator) recordDecision(ctx context.Context, record decisionRecord) {
	if record.Case == nil {
		return
	}
	status := record.Status
	if status == "" {
		status = statusForErr(record.Err)
	}
	errorMessage := ""
	if record.Err != nil {
		errorMessage = record.Err.Error()
	}
	investigationID := int64(0)
	if record.Investigation != nil {
		investigationID = record.Investigation.ID
	}
	_, _ = o.store.AddAIDecisionLog(ctx, caseflow.AIDecisionLog{
		CaseID:             record.Case.ID,
		InvestigationID:    investigationID,
		AgentID:            o.cfg.AgentID,
		DecisionType:       record.DecisionType,
		Reason:             record.Reason,
		InputSnapshotJSON:  jsonSnapshot(record.Input),
		OutputSnapshotJSON: jsonSnapshot(record.Output),
		SelectedToolsJSON:  jsonSnapshot(record.SelectedTools),
		Status:             status,
		LatencyMS:          record.Latency.Milliseconds(),
		ErrorMessage:       errorMessage,
	})
}

func (o *Orchestrator) claimForProcessing(ctx context.Context, c *caseflow.Case) (*caseflow.Case, error) {
	claimed, err := o.transition(ctx, c, caseflow.StatusReadyToInvestigate)
	if err == nil {
		return claimed, nil
	}
	if errors.Is(err, caseflow.ErrVersionConflict) {
		latest, latestErr := o.store.GetCase(ctx, c.ID)
		if latestErr != nil {
			return nil, latestErr
		}
		return nil, skipErr{result: o.skipProcessing(ctx, latest, "case version changed before processing was claimed")}
	}
	return nil, err
}

func (o *Orchestrator) refreshProcessingClaim(ctx context.Context, c *caseflow.Case) (*caseflow.Case, error) {
	refreshed, err := o.store.UpdateCase(ctx, c.ID, c.Version, func(next *caseflow.Case) error {
		next.Status = caseflow.StatusReadyToInvestigate
		return nil
	})
	if err == nil {
		o.recordDecision(ctx, decisionRecord{
			Case:         refreshed,
			DecisionType: "process_stale_claim_recovered",
			Reason:       "case was left in READY_TO_INVESTIGATE beyond stale window and was reclaimed",
			Input:        map[string]any{"case_no": refreshed.CaseNo, "status": refreshed.Status},
			Status:       "recovered",
		})
		return refreshed, nil
	}
	if errors.Is(err, caseflow.ErrVersionConflict) {
		latest, latestErr := o.store.GetCase(ctx, c.ID)
		if latestErr != nil {
			return nil, latestErr
		}
		return nil, skipErr{result: o.skipProcessing(ctx, latest, "stale case was claimed by another worker")}
	}
	return nil, err
}

func (o *Orchestrator) skipProcessing(ctx context.Context, c *caseflow.Case, reason string) caseflow.ProcessResult {
	o.recordDecision(ctx, decisionRecord{
		Case:         c,
		DecisionType: "process_skipped",
		Reason:       reason,
		Input:        map[string]any{"case_no": c.CaseNo, "status": c.Status},
		Status:       "skipped",
	})
	reply := fmt.Sprintf("[%s] 当前状态为 %s，跳过重复排障。", c.CaseNo, c.Status)
	return caseflow.ProcessResult{
		CaseID: c.ID,
		CaseNo: c.CaseNo,
		Status: c.Status,
		Reply:  reply,
	}
}

func (o *Orchestrator) failRunningCase(ctx context.Context, c *caseflow.Case, inv *caseflow.Investigation, reason string) {
	if inv != nil {
		_, _ = o.store.FinishInvestigation(ctx, inv.ID, "failed", reason, nil)
	}
	if c == nil {
		return
	}
	latest, err := o.store.GetCase(ctx, c.ID)
	if err != nil {
		return
	}
	if !caseflow.CanTransition(latest.Status, caseflow.StatusFailed) {
		return
	}
	failed, err := o.transition(ctx, latest, caseflow.StatusFailed)
	if err == nil {
		_, _ = o.store.AddMessage(ctx, caseflow.Message{
			CaseID:  failed.ID,
			Role:    "system",
			Content: "[" + failed.CaseNo + "] 排查已停止：" + reason,
		})
	}
}

func (o *Orchestrator) transition(ctx context.Context, c *caseflow.Case, next caseflow.Status) (*caseflow.Case, error) {
	if !caseflow.CanTransition(c.Status, next) {
		return nil, fmt.Errorf("invalid case transition %s -> %s", c.Status, next)
	}
	return o.store.UpdateCase(ctx, c.ID, c.Version, func(updated *caseflow.Case) error {
		updated.Status = next
		if next == caseflow.StatusDone || next == caseflow.StatusFailed || next == caseflow.StatusCancelled {
			now := time.Now()
			updated.ClosedAt = &now
		}
		return nil
	})
}

func buildMissingInfoReply(caseNo string, missing []string) string {
	questions := make([]string, 0, len(missing))
	for i, field := range missing {
		if i >= 3 {
			break
		}
		questions = append(questions, fmt.Sprintf("%d. 请补充 %s。", i+1, humanField(field)))
	}
	return fmt.Sprintf("[%s] 还需要补充 %d 个关键信息：\n%s", caseNo, len(missing), strings.Join(questions, "\n"))
}

func humanField(field string) string {
	switch field {
	case "symbol":
		return "币对，例如 BTCUSDT"
	case "interval":
		return "K线周期，例如 1m、5m、1h"
	case "abnormal_time":
		return "异常发生的大概时间，并带 timezone，默认 Asia/Shanghai"
	case "issue_type":
		return "异常类型，例如不显示、延迟、价格不一致、余额减少"
	case "asset_symbol":
		return "资产币种，例如 USDT、BTC"
	case "user_id 或 account_id":
		return "user_id 或 account_id"
	default:
		return field
	}
}

func investigationTarget(domain string) string {
	switch domain {
	case caseflow.DomainKline:
		return "K线数据、缓存状态和外部交易所对比"
	case caseflow.DomainAsset:
		return "资产快照、资产事件流和用户相关错误"
	default:
		return "业务数据"
	}
}

func buildToolArgs(toolName string, c *caseflow.Case, entities map[string]string) map[string]any {
	start, end := timeRange(entities["abnormal_time"])
	args := map[string]any{
		"start_time": start.Format(time.RFC3339),
		"end_time":   end.Format(time.RFC3339),
	}
	switch toolName {
	case "get_internal_kline", "get_external_kline_compare", "get_market_source_status":
		args["symbol"] = entities["symbol"]
		args["interval"] = entities["interval"]
		args["exchange"] = fallback(entities["compare_exchange"], "binance")
	case "get_kline_cache_status":
		args["symbol"] = entities["symbol"]
		args["interval"] = entities["interval"]
		args["time_bucket"] = start.Format(time.RFC3339)
	case "get_asset_snapshot":
		args["user_id"] = entities["user_id"]
		args["account_id"] = entities["account_id"]
		args["asset_symbol"] = entities["asset_symbol"]
		args["at_time"] = start.Format(time.RFC3339)
	case "get_asset_events":
		args["user_id"] = entities["user_id"]
		args["account_id"] = entities["account_id"]
		args["asset_symbol"] = entities["asset_symbol"]
	case "get_user_recent_errors":
		args["user_id"] = entities["user_id"]
		args["account_id"] = entities["account_id"]
		args["service_names"] = []string{"asset-service", "order-service"}
	case "get_similar_cases":
		args["issue_domain"] = c.IssueDomain
		args["issue_type"] = c.IssueType
		args["text"] = c.OriginalText
		args["entities"] = map[string]any{}
		for key, value := range entities {
			args["entities"].(map[string]any)[key] = value
		}
		args["limit"] = 5
	}
	return args
}

func timeRange(abnormalTime string) (time.Time, time.Time) {
	if abnormalTime != "" {
		if t, err := time.Parse(time.RFC3339, abnormalTime); err == nil {
			return t.Add(-10 * time.Minute), t.Add(10 * time.Minute)
		}
	}
	now := time.Now()
	return now.Add(-20 * time.Minute), now
}

func boundedToolNames(names []string, max int) []string {
	if max <= 0 {
		max = 10
	}
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
		if len(out) >= max {
			break
		}
	}
	return out
}

func canStartProcessing(status caseflow.Status) bool {
	switch status {
	case caseflow.StatusNew, caseflow.StatusNeedMoreInfo, caseflow.StatusWaitingUserReply:
		return true
	default:
		return false
	}
}

func isActiveProcessingStatus(status caseflow.Status) bool {
	switch status {
	case caseflow.StatusInvestigating, caseflow.StatusWaitingToolResult:
		return true
	default:
		return false
	}
}

func isProcessingStale(c *caseflow.Case, staleAfter time.Duration) bool {
	if c == nil || staleAfter <= 0 || c.UpdatedAt.IsZero() {
		return false
	}
	return time.Since(c.UpdatedAt) > staleAfter
}

func processingStaleAfter(cfg Config) time.Duration {
	seconds := cfg.MaxInvestigationSeconds * 2
	if seconds < 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func statusForErr(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	return "failed"
}

func jsonSnapshot(value any) string {
	if value == nil {
		return ""
	}
	b, err := json.Marshal(masking.MaskValue(value))
	if err != nil {
		return "{}"
	}
	return string(b)
}

func firstErr(values ...error) error {
	for _, err := range values {
		if err != nil {
			return err
		}
	}
	return nil
}

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}
