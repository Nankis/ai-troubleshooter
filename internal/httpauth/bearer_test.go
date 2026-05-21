package httpauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBearerRejectsMissingToken(t *testing.T) {
	handler := Require(Config{AuthEnabled: true, BearerTokens: []string{"token_1"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cases/case_1", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequireBearerAllowsConfiguredToken(t *testing.T) {
	handler := Require(Config{AuthEnabled: true, BearerTokens: []string{"token_1"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/cases/case_1", nil)
	req.Header.Set("Authorization", "Bearer token_1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequireBearerDisabledPassesThrough(t *testing.T) {
	handler := Require(Config{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cases/case_1", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
}
