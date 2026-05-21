package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
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
	log.Printf("worker started with memory queue; use cmd/dev-server for an end-to-end local loop")
	<-ctx.Done()
	pool.Wait()
}
