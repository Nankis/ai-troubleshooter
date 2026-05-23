package decisionbaseline

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/llm"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
)

func TestProcessCaseRecordsDecisionsAndStopsAfterToolFailureLimit(t *testing.T) {
	ctx := context.Background()
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(ctx, caseflow.CreateCaseInput{
		ChatID:         "oc_dev",
		ReporterUserID: "ou_dev",
		OriginalText:   "BTCUSDT 1m K线价格不一致，异常时间 2026-05-21T20:00:00+08:00，手机号 13812345678，api_key: abcdefghijk123",
	})
	if err != nil {
		t.Fatal(err)
	}
	toolClient := &fakeToolClient{err: errors.New("downstream unavailable")}
	runner := New(store, fakeLLM{}, toolClient, Config{
		MaxToolCallsPerCase:     10,
		MaxToolFailuresPerCase:  1,
		MaxInvestigationSeconds: 5,
	})

	result, err := runner.ProcessCase(ctx, c.ID)
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
	logBlob := decisionLogBlob(logs)
	if strings.Contains(logBlob, "13812345678") || strings.Contains(logBlob, "abcdefghijk123") {
		t.Fatalf("decision snapshots contain sensitive raw value: %s", logBlob)
	}
	if !strings.Contains(logBlob, "138****5678") || !strings.Contains(logBlob, "api_key=[REDACTED]") {
		t.Fatalf("decision snapshots were not masked as expected: %s", logBlob)
	}

	second, err := runner.ProcessCase(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != caseflow.StatusNeedHumanConfirmation {
		t.Fatalf("expected duplicate processing to keep NEED_HUMAN_CONFIRMATION, got %s", second.Status)
	}
	if toolClient.calls != 1 {
		t.Fatalf("expected no extra tool calls after duplicate processing, got %d", toolClient.calls)
	}
	logs, err = store.ListAIDecisionLogs(ctx, c.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDecisionStatus(logs, "process_skipped", "skipped") {
		t.Fatalf("expected process_skipped decision log, got %+v", logs)
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
	runner := New(store, fakeLLM{waitForContext: true}, &fakeToolClient{}, Config{
		MaxToolCallsPerCase:     10,
		MaxToolFailuresPerCase:  1,
		MaxInvestigationSeconds: 120,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = runner.ProcessCase(ctx, c.ID)
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

func TestProcessCaseCanAnswerFromHighConfidenceKnowledge(t *testing.T) {
	ctx := context.Background()
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(ctx, caseflow.CreateCaseInput{
		OriginalText: "BTCUSDT 1m K线价格不一致，异常时间 2026-05-21T20:00:00+08:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertKnowledgeItem(ctx, caseflow.KnowledgeItem{
		Title:                "Binance high price tolerance mismatch",
		IssueDomain:          caseflow.DomainKline,
		IssueType:            "价格不一致",
		RecommendedStepsJSON: `["检查聚合精度","确认外部 high 值"]`,
		Confidence:           0.93,
		ObservedCaseCount:    3,
		LastConfirmedReason:  "历史同类 case 已确认由聚合精度差异导致。",
	}); err != nil {
		t.Fatal(err)
	}
	toolClient := &fakeToolClient{}
	runner := New(store, fakeLLM{}, toolClient, Config{
		MaxToolCallsPerCase:     10,
		MaxToolFailuresPerCase:  1,
		MaxInvestigationSeconds: 5,
	})

	result, err := runner.ProcessCase(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != caseflow.StatusNeedHumanConfirmation {
		t.Fatalf("expected NEED_HUMAN_CONFIRMATION, got %s", result.Status)
	}
	if toolClient.calls != 0 {
		t.Fatalf("expected knowledge answer to bypass tools, got %d calls", toolClient.calls)
	}
	if !strings.Contains(result.Reply, "命中平台历史经验") {
		t.Fatalf("unexpected reply: %s", result.Reply)
	}
	logs, err := store.ListAIDecisionLogs(ctx, c.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDecision(logs, "knowledge_retrieval") {
		t.Fatalf("expected knowledge_retrieval log, got %+v", logs)
	}
}

func TestProcessHealthFoodTokenComplaintDoesNotAskForTimezone(t *testing.T) {
	ctx := context.Background()
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(ctx, caseflow.CreateCaseInput{
		OriginalText: "uid:123456 用户反馈 今日 token消耗 数量不对",
	})
	if err != nil {
		t.Fatal(err)
	}
	toolClient := &fakeToolClient{}
	runner := New(store, llm.NewRuleBasedClient(), toolClient, Config{
		MaxToolCallsPerCase:     10,
		MaxToolFailuresPerCase:  3,
		MaxInvestigationSeconds: 5,
	})

	result, err := runner.ProcessCase(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status == caseflow.StatusWaitingUserReply {
		t.Fatalf("should not ask user for internal time fields: %s", result.Reply)
	}
	if result.Status != caseflow.StatusNeedHumanConfirmation {
		t.Fatalf("expected NEED_HUMAN_CONFIRMATION, got %s reply=%s", result.Status, result.Reply)
	}
	if strings.Contains(strings.ToLower(result.Reply), "timezone") || strings.Contains(result.Reply, "Asia/Shanghai") {
		t.Fatalf("reply leaked internal timezone wording: %s", result.Reply)
	}
	if toolClient.calls == 0 {
		t.Fatal("expected health-food readonly tools to be called")
	}
}

func TestMissingInfoReplyUsesUserFriendlyTimeWording(t *testing.T) {
	reply := buildMissingInfoReply("case_1", []string{"abnormal_time"})
	lower := strings.ToLower(reply)
	if strings.Contains(lower, "timezone") || strings.Contains(reply, "Asia/Shanghai") {
		t.Fatalf("reply should not expose internal timezone wording: %s", reply)
	}
	if !strings.Contains(reply, "北京时间（UTC+8）") {
		t.Fatalf("reply should explain default user-facing timezone: %s", reply)
	}
}

func TestMissingInfoReplyDoesNotExposeIssueDomainField(t *testing.T) {
	reply := buildMissingInfoReply("case_1", []string{"issue_domain"})
	if strings.Contains(reply, "issue_domain") {
		t.Fatalf("reply should not expose internal issue_domain field: %s", reply)
	}
	if !strings.Contains(reply, "业务领域或服务名") {
		t.Fatalf("reply should ask in user-facing language: %s", reply)
	}
}

func TestHealthFoodToolArgsDoNotLeakDayWindowToPointLookups(t *testing.T) {
	c := &caseflow.Case{
		IssueDomain:  caseflow.DomainHealthFood,
		IssueType:    "AI配额异常",
		OriginalText: "uid:123456 用户反馈 今日 token消耗 数量不对",
	}
	entities := map[string]string{"uid": "123456", "issue_type": "AI配额异常", "service_name": "health-food"}

	quotaArgs := buildToolArgs("get_health_food_ai_quota", c, entities)
	if _, ok := quotaArgs["start_time"]; ok {
		t.Fatalf("quota point lookup must not receive start_time: %+v", quotaArgs)
	}
	if _, ok := quotaArgs["end_time"]; ok {
		t.Fatalf("quota point lookup must not receive end_time: %+v", quotaArgs)
	}
	if quotaArgs["uid"] != "123456" || quotaArgs["at_time"] == "" {
		t.Fatalf("unexpected quota args: %+v", quotaArgs)
	}

	similarArgs := buildToolArgs("get_similar_cases", c, entities)
	if _, ok := similarArgs["start_time"]; ok {
		t.Fatalf("similar case query must not receive start_time: %+v", similarArgs)
	}

	logArgs := buildToolArgs("search_logs_by_service", c, entities)
	start, err := time.Parse(time.RFC3339, logArgs["start_time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	end, err := time.Parse(time.RFC3339, logArgs["end_time"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if end.Sub(start) > 30*time.Minute {
		t.Fatalf("log query range must stay within gateway limit, got %s", end.Sub(start))
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

func decisionLogBlob(logs []caseflow.AIDecisionLog) string {
	parts := []string{}
	for _, log := range logs {
		parts = append(parts, log.InputSnapshotJSON, log.OutputSnapshotJSON, log.SelectedToolsJSON)
	}
	return strings.Join(parts, "\n")
}
