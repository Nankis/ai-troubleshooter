package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/lark"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
	"github.com/ginseng/ai-troubleshooter/internal/storage"
)

func main() {
	cfg := config.LoadFromEnv()
	openedStore, err := storage.Open(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer openedStore.Close()
	store := openedStore.Store
	q := queue.NewMemoryQueue(256)
	handler := lark.NewHandler(store, q, nil)
	handler.SetOptions(lark.Options{
		VerificationToken: cfg.Lark.VerificationToken,
		AllowedChatIDs:    cfg.Lark.AllowedChatIDs,
	})
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("lark-bot listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
