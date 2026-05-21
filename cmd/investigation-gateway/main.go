package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
)

func main() {
	cfg := config.LoadFromEnv()
	gw := gateway.NewDefault(time.Duration(cfg.Limits.DefaultToolTimeoutSeconds) * time.Second)
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("investigation-gateway listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, gw); err != nil {
		log.Fatal(err)
	}
}
