package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
	"github.com/ginseng/ai-troubleshooter/internal/lark"
	"github.com/ginseng/ai-troubleshooter/internal/llm"
	"github.com/ginseng/ai-troubleshooter/internal/orchestrator"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
	"github.com/ginseng/ai-troubleshooter/internal/worker"
)

func main() {
	cfg := config.LoadFromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := caseflow.NewInMemoryStore()
	q := queue.NewMemoryQueue(256)
	gw := gateway.NewDefault(time.Duration(cfg.Limits.DefaultToolTimeoutSeconds) * time.Second)
	orch := orchestrator.New(store, llm.NewRuleBasedClient(), gw.LocalClient(), orchestrator.Config{
		AgentID:             "business-troubleshooter-v1",
		ModelProvider:       cfg.LLM.Provider,
		ModelName:           cfg.LLM.Model,
		MaxToolCallsPerCase: cfg.Limits.MaxToolCallsPerCase,
	})
	pool := worker.NewPool(q, orch, cfg.Limits.WorkerConcurrency)
	pool.Start(ctx)

	larkHandler := lark.NewHandler(store, q, nil)
	mux := http.NewServeMux()
	mux.Handle("/lark/events", larkHandler)
	mux.Handle("/tools", gw)
	mux.Handle("/tools/", gw)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "dev-server"})
	})
	mux.HandleFunc("/cases/", func(w http.ResponseWriter, r *http.Request) {
		caseRef := strings.TrimPrefix(r.URL.Path, "/cases/")
		c, err := loadCase(r.Context(), store, caseRef)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		entities, _ := store.ListEntities(r.Context(), c.ID)
		messages, _ := store.ListMessages(r.Context(), c.ID)
		writeJSON(w, http.StatusOK, map[string]any{"case": c, "entities": entities, "messages": messages})
	})

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("dev-server listening on http://localhost%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadCase(ctx context.Context, store caseflow.Store, ref string) (*caseflow.Case, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return store.GetCase(ctx, id)
	}
	return store.FindCaseByNo(ctx, ref)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
