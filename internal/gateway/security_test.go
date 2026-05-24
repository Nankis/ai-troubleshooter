package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/audit"
	"github.com/Nankis/ai-troubleshooter/internal/config"
	"github.com/Nankis/ai-troubleshooter/internal/policy"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
)

func TestGatewayRequiresBearerToken(t *testing.T) {
	gw := NewDefault(time.Second).WithSecurity(SecurityConfig{
		AuthEnabled:  true,
		BearerTokens: map[string]string{"token_1": "business-troubleshooter-v1"},
		AgentQPS:     10,
		UserQPS:      10,
		ToolQPS:      10,
	})

	req := httptest.NewRequest(http.MethodPost, "/tools/get_asset_snapshot/invoke", strings.NewReader(`{"arguments":{}}`))
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayRejectsAgentMismatch(t *testing.T) {
	gw := NewDefault(time.Second).WithSecurity(SecurityConfig{
		AuthEnabled:  true,
		BearerTokens: map[string]string{"token_1": "business-troubleshooter-v1"},
		AgentQPS:     10,
		UserQPS:      10,
		ToolQPS:      10,
	})

	body := []byte(`{"case_id":"case_1","agent_id":"other-agent","arguments":{"user_id":"u1","asset_symbol":"USDT"}}`)
	req := httptest.NewRequest(http.MethodPost, "/tools/get_asset_snapshot/invoke", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer token_1")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayAllowsAuthenticatedRequest(t *testing.T) {
	gw := NewDefault(time.Second).WithSecurity(SecurityConfig{
		AuthEnabled:  true,
		BearerTokens: map[string]string{"token_1": "business-troubleshooter-v1"},
		AgentQPS:     10,
		UserQPS:      10,
		ToolQPS:      10,
	})

	body := []byte(`{"case_id":"case_1","agent_id":"business-troubleshooter-v1","arguments":{"user_id":"u1","asset_symbol":"USDT"}}`)
	req := httptest.NewRequest(http.MethodPost, "/tools/get_asset_snapshot/invoke", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer token_1")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayUsesConfiguredAgentPolicy(t *testing.T) {
	gw, err := NewFromConfigWithAudit(config.Config{
		Gateway: config.GatewayConfig{
			AuthEnabled:  true,
			AgentID:      "health-agent",
			BearerTokens: map[string]string{"health-token": "health-agent"},
			AgentQPS:     10,
			UserQPS:      10,
			ToolQPS:      10,
			Agents: []config.GatewayAgentConfig{{
				AgentID:       "health-agent",
				Status:        "enabled",
				AllowedScopes: []string{"health_food:user:read"},
				AllowedTools:  []string{"get_health_food_user_profile"},
			}},
		},
		Limits: config.LimitsConfig{DefaultToolTimeoutSeconds: 1},
	}, audit.NewMemorySink())
	if err != nil {
		t.Fatal(err)
	}

	allowedBody := []byte(`{"case_id":"case_1","agent_id":"health-agent","arguments":{"uid":"hf_user_001"}}`)
	allowedReq := httptest.NewRequest(http.MethodPost, "/tools/get_health_food_user_profile/invoke", bytes.NewReader(allowedBody))
	allowedReq.Header.Set("Authorization", "Bearer health-token")
	allowedRec := httptest.NewRecorder()
	gw.ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusOK {
		t.Fatalf("expected configured health tool to be allowed, got %d body=%s", allowedRec.Code, allowedRec.Body.String())
	}

	deniedBody := []byte(`{"case_id":"case_1","agent_id":"health-agent","arguments":{"user_id":"u1","asset_symbol":"USDT"}}`)
	deniedReq := httptest.NewRequest(http.MethodPost, "/tools/get_asset_snapshot/invoke", bytes.NewReader(deniedBody))
	deniedReq.Header.Set("Authorization", "Bearer health-token")
	deniedRec := httptest.NewRecorder()
	gw.ServeHTTP(deniedRec, deniedReq)
	if deniedRec.Code != http.StatusForbidden {
		t.Fatalf("expected asset tool to be denied by configured policy, got %d body=%s", deniedRec.Code, deniedRec.Body.String())
	}
}

func TestGatewayRateLimit(t *testing.T) {
	gw := NewDefault(time.Second).WithSecurity(SecurityConfig{
		AuthEnabled:  true,
		BearerTokens: map[string]string{"token_1": "business-troubleshooter-v1"},
		AgentQPS:     1,
		UserQPS:      10,
		ToolQPS:      10,
	})

	body := []byte(`{"case_id":"case_1","agent_id":"business-troubleshooter-v1","arguments":{"user_id":"u1","asset_symbol":"USDT"}}`)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/tools/get_asset_snapshot/invoke", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer token_1")
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("expected first request 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected second request 429, got %d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func TestGatewayMasksToolOutputAndAuditArguments(t *testing.T) {
	sink := audit.NewMemorySink()
	registry := tool.NewRegistry()
	if err := registry.Register(tool.Spec{
		Name:          "search_logs_by_service",
		RequiredScope: "logs:read_summary",
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		_ = ctx
		_ = req
		return tool.InvocationResponse{
			Status: "success",
			Data: map[string]any{
				"phone": "13812345678",
				"note":  "operator phone 13987654321 token: abcdefghijkl",
			},
		}, nil
	}); err != nil {
		t.Fatal(err)
	}
	gw := New(registry, policy.NewStaticEngine(policy.DefaultAgents()), sink, time.Second)

	resp, err := gw.Invoke(context.Background(), tool.InvocationRequest{
		CaseID:   "case_1",
		AgentID:  "business-troubleshooter-v1",
		ToolName: "search_logs_by_service",
		Arguments: map[string]any{
			"service_name": "market-service",
			"api_key":      "abcdefghijkl",
			"keyword":      "phone 13812345678",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "13812345678") || strings.Contains(string(payload), "13987654321") || strings.Contains(string(payload), "abcdefghijkl") {
		t.Fatalf("tool output was not masked: %s", payload)
	}
	records := sink.Records()
	if len(records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(records))
	}
	if strings.Contains(records[0].ArgumentsSummary, "13812345678") || strings.Contains(records[0].ArgumentsSummary, "abcdefghijkl") {
		t.Fatalf("audit arguments were not masked: %s", records[0].ArgumentsSummary)
	}
}

func TestHealthFoodAIQuotaSummaryDoesNotExposeTokenBalance(t *testing.T) {
	gw := NewDefault(time.Second)
	resp, err := gw.Invoke(context.Background(), tool.InvocationRequest{
		CaseID:   "case_1",
		AgentID:  "business-troubleshooter-v1",
		ToolName: "get_health_food_ai_quota",
		Arguments: map[string]any{
			"uid": "hf_user_001",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.Summary, "tokens=0") {
		t.Fatalf("quota summary exposes token balance: %s", resp.Summary)
	}
	if !strings.Contains(resp.Summary, "available_tokens=<redacted>") {
		t.Fatalf("quota summary should explicitly mark token balance redacted: %s", resp.Summary)
	}
}

func TestGatewayHTTPReturnsGatewayTimeoutOnToolTimeout(t *testing.T) {
	registry := tool.NewRegistry()
	if err := registry.Register(tool.Spec{
		Name:          "search_logs_by_service",
		RequiredScope: "logs:read_summary",
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		_ = req
		<-ctx.Done()
		return tool.InvocationResponse{}, ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	gw := New(registry, policy.NewStaticEngine(policy.DefaultAgents()), audit.NewMemorySink(), 10*time.Millisecond)

	body := []byte(`{"case_id":"case_1","agent_id":"business-troubleshooter-v1","arguments":{"service_name":"market-service"}}`)
	req := httptest.NewRequest(http.MethodPost, "/tools/search_logs_by_service/invoke", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp tool.InvocationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "failed" || !strings.Contains(resp.Summary, context.DeadlineExceeded.Error()) {
		t.Fatalf("expected failed timeout response, got %+v", resp)
	}
}
