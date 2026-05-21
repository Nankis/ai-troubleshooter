package evolution

import (
	"context"
	"strings"
	"testing"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
)

func TestConfirmRootCauseEvolvesKnowledge(t *testing.T) {
	ctx := context.Background()
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(ctx, caseflow.CreateCaseInput{
		OriginalText: "BTCUSDT 1m K线价格不一致，手机号 13812345678，api_key: abcdefghijk123",
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err = store.UpdateCase(ctx, c.ID, c.Version, func(next *caseflow.Case) error {
		next.IssueDomain = caseflow.DomainKline
		next.IssueType = "价格不一致"
		next.Status = caseflow.StatusNeedHumanConfirmation
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(store)
	result, err := service.ConfirmRootCause(ctx, c, ConfirmRootCauseInput{
		HumanConfirmedReason:  "行情源短时延迟，联系 user@example.com，secret: zyxwvuts98765",
		RootCauseCategory:     "external_source_delay",
		OwnerService:          "market-service",
		IsExternalSourceIssue: true,
		PreventionAction:      "增加行情源延迟监控，token: preventtoken123",
		ConfirmedBy:           "owner_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Case.Status != caseflow.StatusDone {
		t.Fatalf("expected case DONE, got %s", result.Case.Status)
	}
	if result.KnowledgeItem.ID == 0 || result.EvolutionRun.ID == 0 {
		t.Fatalf("expected knowledge and run ids, got %+v", result)
	}
	if result.KnowledgeItem.ObservedCaseCount != 1 {
		t.Fatalf("expected observed case count 1, got %d", result.KnowledgeItem.ObservedCaseCount)
	}

	item, err := store.FindKnowledgeItem(ctx, caseflow.DomainKline, "价格不一致", "external_source_delay")
	if err != nil {
		t.Fatal(err)
	}
	if item.LastConfirmedReason == "" {
		t.Fatal("expected last confirmed reason")
	}
	knowledgeBlob := item.TypicalDescription + item.CommonCausesJSON + item.RecommendedStepsJSON + item.LastConfirmedReason
	if strings.Contains(knowledgeBlob, "13812345678") || strings.Contains(knowledgeBlob, "abcdefghijk123") ||
		strings.Contains(knowledgeBlob, "user@example.com") || strings.Contains(knowledgeBlob, "zyxwvuts98765") ||
		strings.Contains(knowledgeBlob, "preventtoken123") {
		t.Fatalf("knowledge typical description contains sensitive raw value: %s", item.TypicalDescription)
	}
	if strings.Count(item.CommonCausesJSON, "external_source_delay") != 1 {
		t.Fatalf("expected one common cause, got %s", item.CommonCausesJSON)
	}
	if strings.Count(item.RecommendedStepsJSON, "预防动作") != 1 {
		t.Fatalf("expected one prevention step, got %s", item.RecommendedStepsJSON)
	}
	runs, err := store.ListKnowledgeEvolutionRuns(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one evolution run, got %d", len(runs))
	}
	if strings.Contains(runs[0].InputSnapshotJSON, "13812345678") || strings.Contains(runs[0].InputSnapshotJSON, "abcdefghijk123") ||
		strings.Contains(runs[0].InputSnapshotJSON, "user@example.com") || strings.Contains(runs[0].InputSnapshotJSON, "zyxwvuts98765") ||
		strings.Contains(runs[0].InputSnapshotJSON, "preventtoken123") {
		t.Fatalf("evolution snapshot contains sensitive raw value: %s", runs[0].InputSnapshotJSON)
	}
}
