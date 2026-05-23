package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
)

func TestRuleBasedClientClassifiesHealthFoodRecommendation(t *testing.T) {
	client := NewRuleBasedClient()
	input := CaseInput{Case: caseflow.Case{
		OriginalText: "health-food uid:hf_user_001 2026-05-23 10:00 今日推荐没有生成",
	}}

	classification, err := client.ClassifyIssue(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if classification.IssueDomain != caseflow.DomainHealthFood || classification.IssueType != "每日推荐缺失" {
		t.Fatalf("unexpected classification: %+v", classification)
	}

	entities, err := client.ExtractEntities(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	got := caseflow.EntityMap(entities.Entities)
	if got["uid"] != "hf_user_001" || got["issue_type"] != "每日推荐缺失" || got["service_name"] != "health-food" {
		t.Fatalf("unexpected health-food entities: %+v", got)
	}
}

func TestRuleBasedClientClassifiesEnglishHealthFoodRecommendation(t *testing.T) {
	client := NewRuleBasedClient()
	input := CaseInput{Case: caseflow.Case{
		OriginalText: "health-food recommendation missing uid:hf_user_002 abnormal time 2026-05-23 10:00:00",
	}}

	classification, err := client.ClassifyIssue(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if classification.IssueDomain != caseflow.DomainHealthFood || classification.IssueType != "每日推荐缺失" {
		t.Fatalf("unexpected classification: %+v", classification)
	}

	entities, err := client.ExtractEntities(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	got := caseflow.EntityMap(entities.Entities)
	if got["uid"] != "hf_user_002" || got["issue_type"] != "每日推荐缺失" || got["service_name"] != "health-food" {
		t.Fatalf("unexpected health-food entities: %+v", got)
	}
}

func TestRuleBasedClientClassifiesChineseTokenConsumptionComplaint(t *testing.T) {
	client := NewRuleBasedClient()
	input := CaseInput{Case: caseflow.Case{
		OriginalText: "uid:123456 用户反馈 今日 token消耗 数量不对",
	}}

	classification, err := client.ClassifyIssue(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if classification.IssueDomain != caseflow.DomainHealthFood || classification.IssueType != "AI配额异常" {
		t.Fatalf("unexpected classification: %+v", classification)
	}

	entities, err := client.ExtractEntities(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	got := caseflow.EntityMap(entities.Entities)
	if got["uid"] != "123456" || got["issue_type"] != "AI配额异常" || got["service_name"] != "health-food" {
		t.Fatalf("unexpected token consumption entities: %+v", got)
	}
}

func TestRuleBasedClientClassifiesHealthFoodInaccurateRecommendation(t *testing.T) {
	client := NewRuleBasedClient()
	input := CaseInput{Case: caseflow.Case{
		OriginalText: "uid:2054603630081875968 用户反馈 2026-05-23 推荐数据不准，没有按照预设的健康目标推荐",
	}}

	classification, err := client.ClassifyIssue(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if classification.IssueDomain != caseflow.DomainHealthFood || classification.IssueType != "推荐不准确" {
		t.Fatalf("unexpected classification: %+v", classification)
	}
	entities, err := client.ExtractEntities(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	got := caseflow.EntityMap(entities.Entities)
	if got["abnormal_date"] != "2026-05-23" {
		t.Fatalf("expected explicit abnormal_date, got %+v", got)
	}

	action, err := client.DecideNextAction(context.Background(), caseflow.Case{
		IssueDomain: classification.IssueDomain,
		IssueType:   classification.IssueType,
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(action.ToolNames, "get_health_food_recommendation_status") {
		t.Fatalf("expected recommendation tool in plan, got %+v", action.ToolNames)
	}
}

func TestRuleBasedClientClassifiesPastedEnglishTokenCountComplaint(t *testing.T) {
	client := NewRuleBasedClient()
	input := CaseInput{Case: caseflow.Case{
		OriginalText: "uid123456 pasted image validation token count wrong",
	}}

	classification, err := client.ClassifyIssue(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if classification.IssueDomain != caseflow.DomainHealthFood || classification.IssueType != "AI配额异常" {
		t.Fatalf("unexpected classification: %+v", classification)
	}

	entities, err := client.ExtractEntities(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	got := caseflow.EntityMap(entities.Entities)
	if got["uid"] != "123456" || got["issue_type"] != "AI配额异常" || got["service_name"] != "health-food" {
		t.Fatalf("unexpected pasted token count entities: %+v", got)
	}
}

func TestRuleBasedClientSummarizesMissingHealthFoodUID(t *testing.T) {
	client := NewRuleBasedClient()
	report, err := client.SummarizeFindings(context.Background(), caseflow.Case{
		IssueDomain: caseflow.DomainHealthFood,
		IssueType:   "每日推荐缺失",
		UID:         "web_user",
	}, []ToolObservation{
		{ToolName: "get_health_food_user_profile", Status: "success", Summary: "health-food user 999999999999 registered=false membership_level=0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(report.Summary, "uid 999999999999", "用户表中不存在", "确认并补充正确 uid") {
		t.Fatalf("unexpected summary: %s", report.Summary)
	}
}

func TestRuleBasedClientSummarizesRecommendationSourceDateMismatch(t *testing.T) {
	client := NewRuleBasedClient()
	report, err := client.SummarizeFindings(context.Background(), caseflow.Case{
		IssueDomain: caseflow.DomainHealthFood,
		IssueType:   "推荐不准确",
	}, []ToolObservation{
		{ToolName: "get_health_food_user_profile", Status: "success", Summary: "health-food user 2054603630081875968 registered=true membership_level=0"},
		{ToolName: "get_health_food_recommendation_status", Status: "success", Summary: "health-food recommendation date=2026-05-23 exists=true job_status=source_date_mismatch reason=recommendation source_meal_ids include meals outside 2026-05-23"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(report.Summary, "source_meal_ids", "RecommendFoodJob") {
		t.Fatalf("unexpected summary: %s", report.Summary)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
