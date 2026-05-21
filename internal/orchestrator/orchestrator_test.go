package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/llm"
	"github.com/ginseng/ai-troubleshooter/internal/tool"
)

func TestProcessCaseRecordsDecisionsAndStopsAfterToolFailureLimit(t *testing.T) {
	ctx := context.Background()
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(ctx, caseflow.CreateCaseInput{
		ChatID:         "oc_dev",
		ReporterUserID: "ou_dev",
		OriginalText:   "BTCUSDT 1m K线价格不一致，异常时间 2026-05-21T20:00:00+08:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	toolClient := &fakeToolClient{err: errors.New("downstream unavailable")}
	orch := New(store, fakeLLM{}, toolClient, Config{
		MaxToolCallsPerCase:     10,
		MaxToolFailuresPerCase:  1,
		MaxInvestigationSeconds: 5,
	})

	result, err := orch.ProcessCase(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != caseflow.StatusNeedHumanConfirmation {
		t.Fatalf("expected NEED_HUMAN_CONFIRMATION, got %s", result.Status)
	}
	if toolClient.calls != 1 {
		t.Fatalf("expected one tool call before failure limit, got %d", toolClient.calls)
	}
	logs, err := store.ListAIDecisionLogs(ctx, c.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDecision(logs, "classify_issue") || !hasDecision(logs, "decide_next_action") || !hasDecision(logs, "tool_query_stopped") || !hasDecision(logs, "summarize_findings") {
		t.Fatalf("missing expected decision logs: %+v", logs)
	}
}

func TestProcessCaseTimeoutFailsCase(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(context.Background(), caseflow.CreateCaseInput{
		OriginalText: "BTCUSDT 1m K线价格不一致",
	})
	if err != nil {
		t.Fatal(err)
	}
	orch := New(store, fakeLLM{waitForContext: true}, &fakeToolClient{}, Config{
		MaxToolCallsPerCase:     10,
		MaxToolFailuresPerCase:  1,
		MaxInvestigationSeconds: 120,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = orch.ProcessCase(ctx, c.ID)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	updated, err := store.GetCase(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != caseflow.StatusFailed {
		t.Fatalf("expected FAILED, got %s", updated.Status)
	}
	logs, err := store.ListAIDecisionLogs(context.Background(), c.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDecisionStatus(logs, "process_failure", "timeout") {
		t.Fatalf("expected timeout process_failure log, got %+v", logs)
	}
}

type fakeLLM struct {
	waitForContext bool
}

func (f fakeLLM) ClassifyIssue(ctx context.Context, input llm.CaseInput) (llm.IssueClassification, error) {
	if f.waitForContext {
		<-ctx.Done()
		return llm.IssueClassification{}, ctx.Err()
	}
	return llm.IssueClassification{IssueDomain: caseflow.DomainKline, IssueType: "价格不一致", Confidence: 0.9}, nil
}

func (fakeLLM) ExtractEntities(ctx context.Context, input llm.CaseInput) (llm.ExtractedEntities, error) {
	_ = ctx
	conf := 0.9
	return llm.ExtractedEntities{Entities: []caseflow.Entity{
		{Type: "symbol", Value: "BTCUSDT", Source: "test", Confidence: &conf},
		{Type: "interval", Value: "1m", Source: "test", Confidence: &conf},
		{Type: "abnormal_time", Value: "2026-05-21T20:00:00+08:00", Source: "test", Confidence: &conf},
		{Type: "issue_type", Value: "价格不一致", Source: "test", Confidence: &conf},
	}}, nil
}

func (fakeLLM) DecideNextAction(ctx context.Context, state caseflow.Case, entities map[string]string, tools []string) (llm.NextAction, error) {
	_ = ctx
	_ = state
	_ = entities
	_ = tools
	return llm.NextAction{
		ToolNames: []string{"get_internal_kline", "get_external_kline_compare", "get_kline_cache_status"},
		Reason:    "K线问题需要有限查询内部、外部和缓存证据。",
	}, nil
}

func (fakeLLM) SummarizeFindings(ctx context.Context, state caseflow.Case, observations []llm.ToolObservation) (llm.InvestigationReport, error) {
	_ = ctx
	_ = state
	return llm.InvestigationReport{Summary: "证据不足，等待人工确认。", Confidence: 0.3}, nil
}

type fakeToolClient struct {
	calls int
	err   error
}

func (c *fakeToolClient) Invoke(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
	c.calls++
	if c.err != nil {
		return tool.InvocationResponse{ToolCallID: "tc_failed", Status: "failed", Summary: c.err.Error()}, c.err
	}
	return tool.InvocationResponse{ToolCallID: "tc_ok", Status: "success", Summary: "ok"}, nil
}

func hasDecision(logs []caseflow.AIDecisionLog, decisionType string) bool {
	for _, log := range logs {
		if log.DecisionType == decisionType {
			return true
		}
	}
	return false
}

func hasDecisionStatus(logs []caseflow.AIDecisionLog, decisionType string, status string) bool {
	for _, log := range logs {
		if log.DecisionType == decisionType && log.Status == status {
			return true
		}
	}
	return false
}
