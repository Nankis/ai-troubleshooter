package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/config"
	"github.com/Nankis/ai-troubleshooter/internal/gateway"
	"github.com/Nankis/ai-troubleshooter/internal/httpauth"
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
	reloader := gateway.NewCapabilityReloader(gw.Registry(), openedStore.CapabilityStore)
	if err := reloader.Reload(context.Background()); err != nil {
		log.Printf("dynamic capability reload skipped: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/tools", gw)
	mux.Handle("/tools/", gw)
	mux.Handle("/healthz", gw)
	controlAuth := httpauth.Config{AuthEnabled: cfg.ControlAPI.AuthEnabled, BearerTokens: cfg.ControlAPI.BearerTokens}
	mux.Handle("/admin/capabilities/reload", httpauth.RequireFunc(controlAuth, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := reloader.Reload(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("investigation-gateway listening on http://localhost%s", addr)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
