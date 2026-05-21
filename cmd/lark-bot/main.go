package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/lark"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
	"github.com/ginseng/ai-troubleshooter/internal/storage"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.ValidateForLarkBot(); err != nil {
		log.Fatal(err)
	}
	openedStore, err := storage.Open(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer openedStore.Close()
	store := openedStore.Store
	q := queue.NewMemoryQueue(256)
	var messenger lark.Messenger
	if cfg.Lark.AppID != "" && cfg.Lark.AppSecret != "" {
		messenger = lark.NewBotMessenger(lark.BotMessengerOptions{
			AppID:     cfg.Lark.AppID,
			AppSecret: cfg.Lark.AppSecret,
		})
	}
	handler := lark.NewHandler(store, q, messenger)
	handler.SetOptions(lark.Options{
		VerificationToken: cfg.Lark.VerificationToken,
		EncryptKey:        cfg.Lark.EncryptKey,
		AllowedChatIDs:    cfg.Lark.AllowedChatIDs,
	})
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("lark-bot listening on http://localhost%s", addr)
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
