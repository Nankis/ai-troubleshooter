package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/config"
	"github.com/Nankis/ai-troubleshooter/internal/decisionbaseline"
	"github.com/Nankis/ai-troubleshooter/internal/evolution"
	"github.com/Nankis/ai-troubleshooter/internal/gateway"
	"github.com/Nankis/ai-troubleshooter/internal/httpauth"
	"github.com/Nankis/ai-troubleshooter/internal/lark"
	"github.com/Nankis/ai-troubleshooter/internal/llm"
	"github.com/Nankis/ai-troubleshooter/internal/queue"
	"github.com/Nankis/ai-troubleshooter/internal/storage"
	"github.com/Nankis/ai-troubleshooter/internal/vision"
	"github.com/Nankis/ai-troubleshooter/internal/webchat"
	"github.com/Nankis/ai-troubleshooter/internal/worker"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.ValidateForDevServer(); err != nil {
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
	evolver := evolution.NewService(store)
	runner := decisionbaseline.New(store, llm.NewFromConfig(cfg.LLM), gw.LocalClient(), decisionbaseline.Config{
		AgentID:                 cfg.Gateway.AgentID,
		ModelProvider:           cfg.LLM.Provider,
		ModelName:               cfg.LLM.Model,
		MaxToolCallsPerCase:     cfg.Limits.MaxToolCallsPerCase,
		MaxToolFailuresPerCase:  cfg.Limits.MaxToolFailuresPerCase,
		MaxInvestigationSeconds: cfg.Limits.MaxInvestigationSeconds,
	})
	pool := worker.NewPool(q, runner, cfg.Limits.WorkerConcurrency)
	pool.Start(ctx)

	var messenger lark.Messenger
	var imageDownloader lark.ImageDownloader
	if cfg.Lark.AppID != "" && cfg.Lark.AppSecret != "" {
		bot := lark.NewBotMessenger(lark.BotMessengerOptions{
			AppID:     cfg.Lark.AppID,
			AppSecret: cfg.Lark.AppSecret,
			BaseURL:   cfg.Lark.APIBaseURL,
		})
		messenger = bot
		imageDownloader = bot
	}
	larkHandler := lark.NewHandler(store, q, messenger)
	larkHandler.SetOptions(lark.Options{
		VerificationToken: cfg.Lark.VerificationToken,
		EncryptKey:        cfg.Lark.EncryptKey,
		AllowedChatIDs:    cfg.Lark.AllowedChatIDs,
	})
	larkHandler.SetImageProcessor(imageDownloader, vision.NewFromConfigs(cfg.Vision, cfg.LLM), lark.ImageOptions{
		MaxImages:     cfg.Vision.MaxImagesPerMessage,
		MaxImageBytes: cfg.Vision.MaxImageBytes,
	})
	visionClient := vision.NewFromConfigs(cfg.Vision, cfg.LLM)
	webChat := webchat.New(store, runner, visionClient, webchat.Options{
		MaxImages:     cfg.Vision.MaxImagesPerMessage,
		MaxImageBytes: int64(cfg.Vision.MaxImageBytes),
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/", webChat.ServeIndex)
	mux.HandleFunc("/web", webChat.ServeIndex)
	mux.HandleFunc("/web/api/chat", webChat.ServeChat)
	mux.Handle("/lark/events", larkHandler)
	mux.Handle("/feishu/events", larkHandler)
	mux.Handle("/tools", gw)
	mux.Handle("/tools/", gw)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "dev-server"})
	})
	controlAuth := httpauth.Config{AuthEnabled: cfg.ControlAPI.AuthEnabled, BearerTokens: cfg.ControlAPI.BearerTokens}
	mux.Handle("/cases/", httpauth.RequireFunc(controlAuth, func(w http.ResponseWriter, r *http.Request) {
		caseRef, action := splitCasePath(r.URL.Path)
		c, err := loadCase(r.Context(), store, caseRef)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		switch action {
		case "":
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
				return
			}
		case "root-cause":
			handleRootCause(w, r, store, evolver, c)
			return
		case "feedback":
			handleFeedback(w, r, store, c)
			return
		case "evolution-runs":
			handleEvolutionRuns(w, r, store, c)
			return
		case "ai-decisions":
			handleAIDecisionLogs(w, r, store, c)
			return
		default:
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
			return
		}
		entities, _ := store.ListEntities(r.Context(), c.ID)
		messages, _ := store.ListMessages(r.Context(), c.ID)
		rootCause, _ := store.GetRootCause(r.Context(), c.ID)
		runs, _ := store.ListKnowledgeEvolutionRuns(r.Context(), c.ID)
		decisionLogs, _ := store.ListAIDecisionLogs(r.Context(), c.ID, intQuery(r, "decision_log_limit", 100))
		writeJSON(w, http.StatusOK, map[string]any{"case": c, "entities": entities, "messages": messages, "root_cause": rootCause, "evolution_runs": runs, "ai_decision_logs": decisionLogs})
	}))
	mux.Handle("/knowledge", httpauth.RequireFunc(controlAuth, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		items, err := store.ListKnowledgeItems(r.Context(), caseflow.KnowledgeFilter{
			IssueDomain:       r.URL.Query().Get("issue_domain"),
			IssueType:         r.URL.Query().Get("issue_type"),
			RootCauseCategory: r.URL.Query().Get("root_cause_category"),
			Status:            r.URL.Query().Get("status"),
			Limit:             intQuery(r, "limit", 50),
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}))

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("dev-server listening on http://localhost%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadCase(ctx context.Context, store caseflow.Store, ref string) (*caseflow.Case, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return store.GetCase(ctx, id)
	}
	return store.FindCaseByNo(ctx, ref)
}

func splitCasePath(path string) (string, string) {
	trimmed := strings.Trim(strings.TrimPrefix(path, "/cases/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func handleRootCause(w http.ResponseWriter, r *http.Request, store caseflow.Store, evolver *evolution.Service, c *caseflow.Case) {
	switch r.Method {
	case http.MethodGet:
		rootCause, err := store.GetRootCause(r.Context(), c.ID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rootCause)
	case http.MethodPost:
		var input evolution.ConfirmRootCauseInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		result, err := evolver.ConfirmRootCause(r.Context(), c, input)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func handleFeedback(w http.ResponseWriter, r *http.Request, store caseflow.Store, c *caseflow.Case) {
	switch r.Method {
	case http.MethodPost:
		var feedback caseflow.CaseFeedback
		if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		feedback.CaseID = c.ID
		saved, err := store.AddCaseFeedback(r.Context(), feedback)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	case http.MethodGet:
		items, err := store.ListCaseFeedback(r.Context(), c.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func handleEvolutionRuns(w http.ResponseWriter, r *http.Request, store caseflow.Store, c *caseflow.Case) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	runs, err := store.ListKnowledgeEvolutionRuns(r.Context(), c.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": runs})
}

func handleAIDecisionLogs(w http.ResponseWriter, r *http.Request, store caseflow.Store, c *caseflow.Case) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	items, err := store.ListAIDecisionLogs(r.Context(), c.ID, intQuery(r, "limit", 100))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func intQuery(r *http.Request, key string, def int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return def
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
