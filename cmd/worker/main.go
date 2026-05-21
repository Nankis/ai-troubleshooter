package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
	"github.com/ginseng/ai-troubleshooter/internal/llm"
	"github.com/ginseng/ai-troubleshooter/internal/orchestrator"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
	"github.com/ginseng/ai-troubleshooter/internal/storage"
	"github.com/ginseng/ai-troubleshooter/internal/worker"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.ValidateForWorker(); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	openedStore, err := storage.Open(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer openedStore.Close()
	store := openedStore.Store
	q := queue.NewMemoryQueue(256)
	gw, err := gateway.NewFromConfigWithAudit(cfg, openedStore.AuditSink)
	if err != nil {
		log.Fatal(err)
	}
	orch := orchestrator.New(store, llm.NewRuleBasedClient(), gw.LocalClient(), orchestrator.Config{
		AgentID:                 "business-troubleshooter-v1",
		ModelProvider:           cfg.LLM.Provider,
		ModelName:               cfg.LLM.Model,
		MaxToolCallsPerCase:     cfg.Limits.MaxToolCallsPerCase,
		MaxToolFailuresPerCase:  cfg.Limits.MaxToolFailuresPerCase,
		MaxInvestigationSeconds: cfg.Limits.MaxInvestigationSeconds,
	})
	pool := worker.NewPool(q, orch, cfg.Limits.WorkerConcurrency)
	pool.Start(ctx)
	log.Printf("worker started with memory queue; use cmd/dev-server for an end-to-end local loop")
	<-ctx.Done()
	pool.Wait()
}
