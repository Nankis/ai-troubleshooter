package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
	"github.com/ginseng/ai-troubleshooter/internal/storage"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.ValidateForGateway(); err != nil {
		log.Fatal(err)
	}
	openedStore, err := storage.Open(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer openedStore.Close()
	gw, err := gateway.NewFromConfigWithAudit(cfg, openedStore.AuditSink)
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("investigation-gateway listening on http://localhost%s", addr)
	server := &http.Server{Addr: addr, Handler: gw, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
