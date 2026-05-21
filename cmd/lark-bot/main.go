package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/lark"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
)

func main() {
	cfg := config.LoadFromEnv()
	store := caseflow.NewInMemoryStore()
	q := queue.NewMemoryQueue(256)
	handler := lark.NewHandler(store, q, nil)
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("lark-bot listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
