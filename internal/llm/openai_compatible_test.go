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
