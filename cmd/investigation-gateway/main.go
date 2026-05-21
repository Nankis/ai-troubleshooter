package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ginseng/ai-troubleshooter/internal/config"
	"github.com/ginseng/ai-troubleshooter/internal/gateway"
)

func main() {
	cfg := config.LoadFromEnv()
	gw, err := gateway.NewFromConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Printf("investigation-gateway listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, gw); err != nil {
		log.Fatal(err)
	}
}
