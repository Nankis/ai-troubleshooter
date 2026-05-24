package decisionbaseline

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

// Runner is the Go phase-0 decision baseline used for local smoke tests and
// fallback. The target Agent orchestration lives in apps/decision-engine.
type Runner struct {
	store caseflow.Store
	llm   llm.LLMClient
	tools ToolClient
	cfg   Config
}

func New(store caseflow.Store, llmClient llm.LLMClient, toolClient ToolClient, cfg Config) *Runner {
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
	return &Runner{store: store, llm: llmClient, tools: toolClient, cfg: cfg}
}

func (o *Runner) ProcessCase(parent context.Context, caseID int64) (result caseflow.ProcessResult, err error) {
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
			Reason:        "baseline decision runner stopped and finalized the case",
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

	extractedMap := caseflow.EntityMap(extracted.Entities)
	c, err = o.store.UpdateCase(ctx, c.ID, c.Version, func(next *caseflow.Case) error {
		next.IssueDomain = classification.IssueDomain
		next.IssueType = classification.IssueType
		if businessUID := fallback(extractedMap["uid"], extractedMap["user_id"]); businessUID != "" {
			next.UID = businessUID
		}
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

	knowledgeItems, knowledgeErr := o.store.ListKnowledgeItems(ctx, caseflow.KnowledgeFilter{
		IssueDomain: c.IssueDomain,
		IssueType:   c.IssueType,
		Status:      "active",
		Limit:       3,
	})
	if knowledgeErr == nil && len(knowledgeItems) > 0 {
		top := knowledgeItems[0]
		direct := shouldAnswerFromKnowledge(c, top)
		o.recordDecision(ctx, decisionRecord{
			Case:         c,
			DecisionType: "knowledge_retrieval",
			Reason:       "retrieve platform knowledge before querying downstream business tools",
			Input:        map[string]any{"issue_domain": c.IssueDomain, "issue_type": c.IssueType},
			Output: map[string]any{
				"matched":                 true,
				"top_knowledge_item_id":   top.ID,
				"top_knowledge_title":     top.Title,
				"confidence":              top.Confidence,
				"observed_case_count":     top.ObservedCaseCount,
				"answer_directly":         direct,
				"requires_realtime_check": requiresRealtimeEvidence(c),
			},
			Status: "success",
		})
		if direct {
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
				InitialHypothesis: "high-confidence platform knowledge matched",
			})
			if err != nil {
				return caseflow.ProcessResult{}, err
			}
			inv = &invValue
			reply := buildKnowledgeReply(c.CaseNo, top)
			_, _ = o.store.FinishInvestigation(ctx, inv.ID, "finished", reply, &top.Confidence)
			_, _ = o.store.AddMessage(ctx, caseflow.Message{CaseID: c.ID, Role: "agent", Content: reply})
			c, err = o.transition(ctx, c, caseflow.StatusNeedHumanConfirmation)
			if err != nil {
				return caseflow.ProcessResult{}, err
			}
			return caseflow.ProcessResult{CaseID: c.ID, CaseNo: c.CaseNo, Status: c.Status, Reply: reply}, nil
		}
	} else if knowledgeErr != nil {
		o.recordDecision(ctx, decisionRecord{
			Case:         c,
			DecisionType: "knowledge_retrieval",
			Reason:       "platform knowledge retrieval failed; continue with bounded gateway tools",
			Input:        map[string]any{"issue_domain": c.IssueDomain, "issue_type": c.IssueType},
			Status:       "failed",
			Err:          knowledgeErr,
		})
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
	selectedTools = ensureEvidenceTools(*c, selectedTools, o.cfg.MaxToolCallsPerCase)
	o.recordDecision(ctx, decisionRecord{
		Case:          c,
		Investigation: inv,
		DecisionType:  "decide_next_action",
		Reason:        fallback(action.Reason, "choose readonly tools based on issue domain and extracted entities"),
		Input:         map[string]any{"issue_domain": c.IssueDomain, "issue_type": c.IssueType, "entities": entityMap, "max_tool_calls": o.cfg.MaxToolCallsPerCase},
		Output:        map[string]any{"requested_tools": action.ToolNames, "selected_tools": selectedTools, "augmented": len(selectedTools) > len(action.ToolNames), "truncated": len(selectedTools) < len(action.ToolNames)},
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
			CaseID:       c.CaseNo,
			AgentID:      o.cfg.AgentID,
			CallerUserID: fallback(c.UID, c.ReporterUserID),
			LarkUserID:   c.ReporterUserID,
			ChatID:       c.ChatID,
			ToolName:     name,
			Arguments:    args,
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

func (o *Runner) recordDecision(ctx context.Context, record decisionRecord) {
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

func (o *Runner) claimForProcessing(ctx context.Context, c *caseflow.Case) (*caseflow.Case, error) {
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

func (o *Runner) refreshProcessingClaim(ctx context.Context, c *caseflow.Case) (*caseflow.Case, error) {
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

func (o *Runner) skipProcessing(ctx context.Context, c *caseflow.Case, reason string) caseflow.ProcessResult {
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

func (o *Runner) failRunningCase(ctx context.Context, c *caseflow.Case, inv *caseflow.Investigation, reason string) {
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

func (o *Runner) transition(ctx context.Context, c *caseflow.Case, next caseflow.Status) (*caseflow.Case, error) {
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

func shouldAnswerFromKnowledge(c *caseflow.Case, item caseflow.KnowledgeItem) bool {
	if item.Confidence < 0.88 {
		return false
	}
	if item.ObservedCaseCount < 2 {
		return false
	}
	return !requiresRealtimeEvidence(c)
}

func requiresRealtimeEvidence(c *caseflow.Case) bool {
	text := strings.ToLower(c.IssueType + " " + c.OriginalText + " " + c.OCRText)
	if containsAny(text, "当前", "现在", "实时", "余额", "资产", "冻结", "充值", "提现", "延迟", "超时", "失败", "报错") {
		return true
	}
	return false
}

func buildKnowledgeReply(caseNo string, item caseflow.KnowledgeItem) string {
	lines := []string{
		"[" + caseNo + "] 命中平台历史经验，先给出高置信排查建议：",
		"",
		"经验：" + item.Title,
	}
	if item.LastConfirmedReason != "" {
		lines = append(lines, "历史确认根因："+item.LastConfirmedReason)
	}
	if item.RecommendedStepsJSON != "" {
		lines = append(lines, "建议步骤："+item.RecommendedStepsJSON)
	}
	lines = append(lines, "", "说明：本结论来自平台经验库，仍建议业务 owner 根据当前现场确认最终根因。")
	return strings.Join(lines, "\n")
}

func humanField(field string) string {
	switch field {
	case "symbol":
		return "币对，例如 BTCUSDT"
	case "interval":
		return "K线周期，例如 1m、5m、1h"
	case "abnormal_time":
		return "异常大概发生时间，例如今天上午、昨晚 8 点、2026-05-23 10:00；不确定可以说“刚刚/今天”。默认按北京时间（UTC+8）理解，如果你反馈的是 UTC 时间请注明"
	case "issue_type":
		return "异常现象，例如价格不一致、余额减少、今日推荐没生成、token 消耗不对"
	case "asset_symbol":
		return "资产币种，例如 USDT、BTC"
	case "user_id 或 account_id":
		return "user_id 或 account_id"
	case "user_id 或 uid":
		return "用户 uid，例如 uid:123456"
	case "issue_domain":
		return "业务领域或服务名，例如 health-food、资产、K线"
	default:
		return field
	}
}

func investigationTarget(domain string) string {
	switch domain {
	case caseflow.DomainHealthFood:
		return "health-food 用户上下文、AI配额、餐食数据、推荐任务和服务日志"
	case caseflow.DomainKline:
		return "K线数据、缓存状态和外部交易所对比"
	case caseflow.DomainAsset:
		return "资产快照、资产事件流和用户相关错误"
	default:
		return "业务数据"
	}
}

func buildToolArgs(toolName string, c *caseflow.Case, entities map[string]string) map[string]any {
	start, end := timeRangeForCase(c, entities)
	args := map[string]any{}
	switch toolName {
	case "get_internal_kline", "get_external_kline_compare", "get_market_source_status":
		args["symbol"] = entities["symbol"]
		args["interval"] = entities["interval"]
		args["exchange"] = fallback(entities["compare_exchange"], "binance")
		args["start_time"] = start.Format(time.RFC3339)
		args["end_time"] = end.Format(time.RFC3339)
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
		args["start_time"] = start.Format(time.RFC3339)
		args["end_time"] = end.Format(time.RFC3339)
	case "get_user_recent_errors":
		args["user_id"] = entities["user_id"]
		args["account_id"] = entities["account_id"]
		args["service_names"] = []string{"asset-service", "order-service"}
		args["start_time"] = start.Format(time.RFC3339)
		args["end_time"] = end.Format(time.RFC3339)
	case "get_health_food_user_profile", "get_health_food_ai_quota":
		args["user_id"] = fallback(entities["user_id"], entities["uid"])
		args["uid"] = fallback(entities["uid"], entities["user_id"])
		args["at_time"] = start.Format(time.RFC3339)
		args["trace_id"] = entities["trace_id"]
	case "get_health_food_meal_records":
		args["user_id"] = fallback(entities["user_id"], entities["uid"])
		args["uid"] = fallback(entities["uid"], entities["user_id"])
		args["start_time"] = start.Format(time.RFC3339)
		args["end_time"] = end.Format(time.RFC3339)
		args["limit"] = 50
	case "get_health_food_recommendation_status":
		args["user_id"] = fallback(entities["user_id"], entities["uid"])
		args["uid"] = fallback(entities["uid"], entities["user_id"])
		args["start_time"] = start.Format(time.RFC3339)
		args["end_time"] = end.Format(time.RFC3339)
		args["recommendation_date"] = start.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
	case "get_similar_cases":
		args["issue_domain"] = c.IssueDomain
		args["issue_type"] = c.IssueType
		args["text"] = c.OriginalText
		args["entities"] = map[string]any{}
		for key, value := range entities {
			args["entities"].(map[string]any)[key] = value
		}
		args["limit"] = 5
	case "search_logs_by_service":
		logStart, logEnd := logTimeRange(entities["abnormal_time"])
		args["service_name"] = fallback(entities["service_name"], serviceNameForDomain(c.IssueDomain))
		args["keyword"] = fallback(entities["trace_id"], c.IssueType)
		args["trace_id"] = entities["trace_id"]
		args["level"] = "error"
		args["limit"] = 20
		args["start_time"] = logStart.Format(time.RFC3339)
		args["end_time"] = logEnd.Format(time.RFC3339)
	}
	return args
}

func serviceNameForDomain(domain string) string {
	switch domain {
	case caseflow.DomainHealthFood:
		return "health-food"
	case caseflow.DomainAsset:
		return "asset-service"
	case caseflow.DomainKline:
		return "market-service"
	default:
		return ""
	}
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

func timeRangeForCase(c *caseflow.Case, entities map[string]string) (time.Time, time.Time) {
	if c != nil && c.IssueDomain == caseflow.DomainHealthFood && usesHealthFoodDailyWindow(c) {
		if start, end, ok := dayRangeFromEntity(entities["abnormal_date"], entities["abnormal_time"]); ok {
			return start, end
		}
	}
	if start, end := timeRange(entities["abnormal_time"]); entities["abnormal_time"] != "" {
		return start, end
	}
	if c != nil && c.IssueDomain == caseflow.DomainHealthFood {
		now := time.Now().In(beijingLocation())
		text := strings.ToLower(c.OriginalText + "\n" + c.OCRText + "\n" + c.IssueType)
		if containsAny(text, "今日", "今天", "当天", "当日", "today", "每日推荐", "token") {
			start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			return start, now
		}
	}
	return timeRange("")
}

func usesHealthFoodDailyWindow(c *caseflow.Case) bool {
	if c == nil || c.IssueDomain != caseflow.DomainHealthFood {
		return false
	}
	text := strings.ToLower(c.OriginalText + "\n" + c.OCRText + "\n" + c.IssueType)
	return containsAny(text,
		"今日", "今天", "当天", "当日", "today",
		"每日推荐", "今日推荐", "推荐数据", "推荐不准", "推荐不准确", "健康目标",
		"token", "配额", "消耗", "用量",
	)
}

func dayRangeFromEntity(dateText string, timeText string) (time.Time, time.Time, bool) {
	loc := beijingLocation()
	if dateText != "" {
		if t, err := time.ParseInLocation("2006-01-02", dateText, loc); err == nil {
			return dayRange(t)
		}
	}
	if timeText != "" {
		if t, err := time.Parse(time.RFC3339, timeText); err == nil {
			return dayRange(t.In(loc))
		}
	}
	return time.Time{}, time.Time{}, false
}

func dayRange(t time.Time) (time.Time, time.Time, bool) {
	loc := beijingLocation()
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	now := time.Now().In(loc)
	if sameDay(start, now) && end.After(now) {
		end = now
	}
	return start, end, true
}

func sameDay(a time.Time, b time.Time) bool {
	aa := a.In(beijingLocation())
	bb := b.In(beijingLocation())
	return aa.Year() == bb.Year() && aa.YearDay() == bb.YearDay()
}

func logTimeRange(abnormalTime string) (time.Time, time.Time) {
	if abnormalTime != "" {
		return timeRange(abnormalTime)
	}
	now := time.Now()
	return now.Add(-30 * time.Minute), now
}

func beijingLocation() *time.Location {
	return time.FixedZone("UTC+8", 8*3600)
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

func ensureEvidenceTools(c caseflow.Case, selected []string, max int) []string {
	if max <= 0 {
		max = 10
	}
	out := append([]string{}, selected...)
	add := func(names ...string) {
		for _, name := range names {
			if name == "" || containsTool(out, name) || len(out) >= max {
				continue
			}
			out = append(out, name)
		}
	}
	switch c.IssueDomain {
	case caseflow.DomainHealthFood:
		add("get_health_food_user_profile")
		text := strings.ToLower(c.IssueType + " " + c.OriginalText + " " + c.OCRText)
		switch {
		case containsAny(text, "推荐", "recommend"):
			add("get_health_food_meal_records", "get_health_food_recommendation_status", "search_logs_by_service", "get_similar_cases")
		case containsAny(text, "token", "quota", "配额", "消耗"):
			add("get_health_food_ai_quota", "search_logs_by_service", "get_similar_cases")
		default:
			add("search_logs_by_service", "get_similar_cases")
		}
	}
	return out
}

func containsTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
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

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
