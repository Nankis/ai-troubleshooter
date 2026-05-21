package gateway

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
