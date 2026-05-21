package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/llm"
	"github.com/ginseng/ai-troubleshooter/internal/tool"
)

type ToolClient interface {
	Invoke(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error)
}

type Config struct {
	AgentID             string
	ModelProvider       string
	ModelName           string
	MaxToolCallsPerCase int
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
	return &Orchestrator{store: store, llm: llmClient, tools: toolClient, cfg: cfg}
}

func (o *Orchestrator) ProcessCase(ctx context.Context, caseID int64) (caseflow.ProcessResult, error) {
	c, err := o.store.GetCase(ctx, caseID)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	messages, _ := o.store.ListMessages(ctx, c.ID)
	input := llm.CaseInput{Case: *c, Messages: messages}

	classification, err := o.llm.ClassifyIssue(ctx, input)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	extracted, err := o.llm.ExtractEntities(ctx, input)
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

	inv, err := o.store.CreateInvestigation(ctx, caseflow.Investigation{
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

	action, err := o.llm.DecideNextAction(ctx, *c, entityMap, nil)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	c, err = o.transition(ctx, c, caseflow.StatusWaitingToolResult)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}

	observations := []llm.ToolObservation{}
	toolCallIDs := []string{}
	for i, name := range action.ToolNames {
		if i >= o.cfg.MaxToolCallsPerCase {
			break
		}
		resp, err := o.tools.Invoke(ctx, tool.InvocationRequest{
			CaseID:     c.CaseNo,
			AgentID:    o.cfg.AgentID,
			LarkUserID: c.ReporterUserID,
			ChatID:     c.ChatID,
			ToolName:   name,
			Arguments:  buildToolArgs(name, c, entityMap),
		})
		toolCallIDs = append(toolCallIDs, resp.ToolCallID)
		status := resp.Status
		summary := resp.Summary
		if err != nil && summary == "" {
			summary = err.Error()
		}
		observations = append(observations, llm.ToolObservation{ToolName: name, Summary: summary, Status: status})
	}

	report, err := o.llm.SummarizeFindings(ctx, *c, observations)
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

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}
