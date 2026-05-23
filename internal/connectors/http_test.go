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
	params, ok := captured.Params.(map[string]any)
	if !ok {
		t.Fatalf("unexpected params type: %T", captured.Params)
	}
	if params["symbol"] != "BTCUSDT" || params["start_time"] == nil || params["StartTime"] != nil {
		t.Fatalf("expected snake_case kline params, got %+v", params)
	}
	if len(result.Candles) != 1 || result.Source != "market-service" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHTTPOpsConnectorSendsSnakeCaseLogParams(t *testing.T) {
	var captured readonlyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/readonly/ops/logs/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeTestJSON(w, map[string]any{
			"request_id":      captured.RequestID,
			"source":          "health-food-admin-log",
			"queried_at":      "2026-05-23T10:02:00+08:00",
			"data_updated_at": "2026-05-23T10:02:00+08:00",
			"version":         "v1",
			"warnings":        []string{},
			"data": map[string]any{
				"service_name": "health-food",
				"total":        1,
				"samples": []map[string]any{
					{"time": "2026-05-23T10:00:00+08:00", "level": "error", "service": "health-food", "message": "boom"},
				},
			},
		})
	}))
	defer server.Close()

	connector, err := NewHTTPOpsConnector(HTTPConfig{BaseURL: server.URL, APIKey: "test-key", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	_, err = connector.SearchLogs(context.Background(), LogQuery{
		ServiceName: "health-food",
		StartTime:   time.Date(2026, 5, 23, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		EndTime:     time.Date(2026, 5, 23, 10, 10, 0, 0, time.FixedZone("CST", 8*3600)),
		Level:       "error",
		Keyword:     "recommend",
		TraceID:     "trace_1",
		Limit:       5,
	})
	if err != nil {
		t.Fatal(err)
	}
	params, ok := captured.Params.(map[string]any)
	if !ok {
		t.Fatalf("unexpected params type: %T", captured.Params)
	}
	if params["service_name"] != "health-food" || params["trace_id"] != "trace_1" || params["ServiceName"] != nil {
		t.Fatalf("expected snake_case log params, got %+v", params)
	}
}

func TestHTTPHealthFoodConnectorUsesReadonlyEnvelope(t *testing.T) {
	var captured readonlyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/readonly/health-food/recommendation/status" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Tool-Name"); got != "get_health_food_recommendation_status" {
			t.Fatalf("expected tool header, got %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeTestJSON(w, map[string]any{
			"request_id":      captured.RequestID,
			"source":          "health-food",
			"queried_at":      "2026-05-23T10:01:00+08:00",
			"data_updated_at": "2026-05-23T10:00:00+08:00",
			"version":         "v1",
			"warnings":        []string{},
			"data": map[string]any{
				"uid":                   "hf_user_001",
				"recommend_date":        "2026-05-23",
				"has_recommendation":    false,
				"job_status":            "failed",
				"meal_count":            2,
				"meal_data_fingerprint": "stale",
				"failure_reason":        "mock failure",
				"source_meal_ids":       []string{"meal_1", "meal_2"},
			},
		})
	}))
	defer server.Close()

	connector, err := NewHTTPHealthFoodConnector(HTTPConfig{BaseURL: server.URL, APIKey: "test-key", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx := ContextWithRequestMeta(context.Background(), RequestMeta{
		RequestID:    "req_hf_1",
		CaseID:       "case_hf_1",
		AgentID:      "agent_1",
		CallerUserID: "ou_1",
		ToolName:     "get_health_food_recommendation_status",
	})
	result, err := connector.RecommendationStatus(ctx, HealthFoodQuery{
		UID:                "hf_user_001",
		RecommendationDate: "2026-05-23",
	})
	if err != nil {
		t.Fatal(err)
	}
	params, ok := captured.Params.(map[string]any)
	if !ok {
		t.Fatalf("unexpected params type: %T", captured.Params)
	}
	if captured.CaseID != "case_hf_1" || params["uid"] != "hf_user_001" {
		t.Fatalf("readonly envelope lost health-food metadata: captured=%+v params=%+v", captured, params)
	}
	if result.HasRecommendation || result.JobStatus != "failed" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func writeTestJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
