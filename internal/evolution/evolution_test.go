package evolution

import (
	"context"
	"testing"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
)

func TestConfirmRootCauseEvolvesKnowledge(t *testing.T) {
	ctx := context.Background()
	store := caseflow.NewInMemoryStore()
	c, err := store.CreateCase(ctx, caseflow.CreateCaseInput{
		OriginalText: "BTCUSDT 1m K线价格不一致",
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
		HumanConfirmedReason:  "行情源短时延迟，补偿任务完成前用户看到旧 high",
		RootCauseCategory:     "external_source_delay",
		OwnerService:          "market-service",
		IsExternalSourceIssue: true,
		PreventionAction:      "增加行情源延迟监控",
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
}
