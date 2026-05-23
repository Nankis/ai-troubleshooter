package llm

import (
	"context"
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
