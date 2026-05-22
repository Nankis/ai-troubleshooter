package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
)

func TestOpenAICompatibleClientClassifiesIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer key_1" {
			t.Fatalf("unexpected auth header %s", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-test" {
			t.Fatalf("unexpected model: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": `{"issue_domain":"kline","issue_type":"价格不一致","confidence":0.91}`},
			}},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(OpenAICompatibleOptions{
		BaseURL: server.URL + "/v1",
		APIKey:  "key_1",
		Model:   "gpt-test",
	})
	got, err := client.ClassifyIssue(context.Background(), CaseInput{Case: caseflow.Case{OriginalText: "BTCUSDT K线价格不一致"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.IssueDomain != caseflow.DomainKline || got.IssueType != "价格不一致" {
		t.Fatalf("unexpected classification: %+v", got)
	}
}

func TestRuleBasedClientNormalizesMinutePrecisionTime(t *testing.T) {
	client := NewRuleBasedClient()
	got, err := client.ExtractEntities(context.Background(), CaseInput{Case: caseflow.Case{
		OriginalText: "BTCUSDT 1m 在 2026-05-21 20:03 最高价和 Binance 不一致",
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, entity := range got.Entities {
		if entity.Type == "abnormal_time" {
			if entity.Value != "2026-05-21T20:03:00+08:00" {
				t.Fatalf("unexpected abnormal_time %q", entity.Value)
			}
			return
		}
	}
	t.Fatalf("abnormal_time not extracted: %+v", got.Entities)
}

func TestRuleBasedClientPrefersHighMismatchOverPossibleDelay(t *testing.T) {
	client := NewRuleBasedClient()
	got, err := client.ClassifyIssue(context.Background(), CaseInput{Case: caseflow.Case{
		OCRText: "Issue: high price mismatch. 可能原因包括数据同步延迟或数据源问题。",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.IssueType != "最高最低不一致" {
		t.Fatalf("unexpected issue type: %+v", got)
	}
}
