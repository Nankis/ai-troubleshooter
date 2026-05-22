package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/config"
	"github.com/Nankis/ai-troubleshooter/internal/decisionbaseline"
	"github.com/Nankis/ai-troubleshooter/internal/gateway"
	"github.com/Nankis/ai-troubleshooter/internal/httpauth"
	"github.com/Nankis/ai-troubleshooter/internal/llm"
	"github.com/Nankis/ai-troubleshooter/internal/storage"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.ValidateForBaselineOrchestrator(); err != nil {
		log.Fatal(err)
	}
	openedStore, err := storage.Open(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer openedStore.Close()
	store := openedStore.Store
	gw, err := gateway.NewFromConfigWithAudit(cfg, openedStore.AuditSink)
	if err != nil {
		log.Fatal(err)
	}
	runner := decisionbaseline.New(store, llm.NewFromConfig(cfg.LLM), gw.LocalClient(), decisionbaseline.Config{
		AgentID:                 "business-troubleshooter-v1",
		ModelProvider:           cfg.LLM.Provider,
		ModelName:               cfg.LLM.Model,
		MaxToolCallsPerCase:     cfg.Limits.MaxToolCallsPerCase,
		MaxToolFailuresPerCase:  cfg.Limits.MaxToolFailuresPerCase,
		MaxInvestigationSeconds: cfg.Limits.MaxInvestigationSeconds,
	})

	mux := http.NewServeMux()
	controlAuth := httpauth.Config{AuthEnabled: cfg.ControlAPI.AuthEnabled, BearerTokens: cfg.ControlAPI.BearerTokens}
	mux.Handle("/cases", httpauth.RequireFunc(controlAuth, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		var input caseflow.CreateCaseInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		c, err := store.CreateCase(r.Context(), input)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, c)
	}))
	mux.Handle("/cases/", httpauth.RequireFunc(controlAuth, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/process") {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
			return
		}
		ref := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/cases/"), "/process")
		id, err := strconv.ParseInt(ref, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "case id must be numeric in this service"})
			return
		}
		result, err := runner.ProcessCase(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}))

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("baseline-orchestrator listening on http://localhost%s", addr)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
