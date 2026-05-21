package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPKlineConnectorSendsReadonlyEnvelope(t *testing.T) {
	var captured readonlyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/readonly/market/kline/internal" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Case-Id"); got != "case_1" {
			t.Fatalf("expected case header, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer auth, got %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeTestJSON(w, map[string]any{
			"request_id":      captured.RequestID,
			"source":          "market-service",
			"queried_at":      "2026-05-21T20:00:00+08:00",
			"data_updated_at": "2026-05-21T19:59:58+08:00",
			"version":         "v1",
			"warnings":        []string{},
			"data": map[string]any{
				"candles": []map[string]any{
					{
						"open_time": "2026-05-21T20:00:00+08:00",
						"open":      1,
						"high":      2,
						"low":       1,
						"close":     2,
						"volume":    10,
					},
				},
			},
		})
	}))
	defer server.Close()

	connector, err := NewHTTPKlineConnector(HTTPConfig{BaseURL: server.URL, APIKey: "test-key", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx := ContextWithRequestMeta(context.Background(), RequestMeta{
		RequestID:    "req_1",
		CaseID:       "case_1",
		AgentID:      "agent_1",
		CallerUserID: "ou_1",
		ToolName:     "get_internal_kline",
	})
	result, err := connector.InternalKline(ctx, KlineQuery{
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		StartTime: time.Date(2026, 5, 21, 20, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		EndTime:   time.Date(2026, 5, 21, 20, 10, 0, 0, time.FixedZone("CST", 8*3600)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.CaseID != "case_1" || captured.AgentID != "agent_1" || captured.ToolName != "get_internal_kline" {
		t.Fatalf("readonly envelope lost metadata: %+v", captured)
	}
	if len(result.Candles) != 1 || result.Source != "market-service" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func writeTestJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
