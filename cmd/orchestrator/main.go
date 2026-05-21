package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
	"github.com/ginseng/ai-troubleshooter/internal/llm"
	"github.com/ginseng/ai-troubleshooter/internal/orchestrator"
)

func main() {
	cfg := config.LoadFromEnv()
	store := caseflow.NewInMemoryStore()
	gw, err := gateway.NewFromConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	orch := orchestrator.New(store, llm.NewRuleBasedClient(), gw.LocalClient(), orchestrator.Config{
		AgentID:             "business-troubleshooter-v1",
		ModelProvider:       cfg.LLM.Provider,
		ModelName:           cfg.LLM.Model,
		MaxToolCallsPerCase: cfg.Limits.MaxToolCallsPerCase,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/cases", func(w http.ResponseWriter, r *http.Request) {
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
	})
	mux.HandleFunc("/cases/", func(w http.ResponseWriter, r *http.Request) {
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
		result, err := orch.ProcessCase(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("orchestrator listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
