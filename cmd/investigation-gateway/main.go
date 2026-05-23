package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/config"
	"github.com/Nankis/ai-troubleshooter/internal/gateway"
	"github.com/Nankis/ai-troubleshooter/internal/storage"
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
	if err := gateway.NewCapabilityReloader(gw.Registry(), openedStore.CapabilityStore).Reload(context.Background()); err != nil {
		log.Printf("dynamic capability reload skipped: %v", err)
	}
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("investigation-gateway listening on http://localhost%s", addr)
	server := &http.Server{Addr: addr, Handler: gw, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
