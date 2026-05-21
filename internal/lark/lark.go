package lark

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
)

type Event struct {
	ChatID    string   `json:"chat_id"`
	ThreadID  string   `json:"thread_id"`
	MessageID string   `json:"message_id"`
	UserID    string   `json:"user_id"`
	Text      string   `json:"text"`
	ImageKeys []string `json:"image_keys"`
	OCRText   string   `json:"ocr_text"`
}

type Messenger interface {
	SendMessage(ctx context.Context, chatID string, threadID string, text string) error
}

type LogMessenger struct{}

func (LogMessenger) SendMessage(ctx context.Context, chatID string, threadID string, text string) error {
	_ = ctx
	log.Printf("lark_reply chat_id=%s thread_id=%s text=%q", chatID, threadID, text)
	return nil
}

type Handler struct {
	store     caseflow.Store
	queue     queue.Queue
	messenger Messenger
}

func NewHandler(store caseflow.Store, q queue.Queue, messenger Messenger) *Handler {
	if messenger == nil {
		messenger = LogMessenger{}
	}
	return &Handler{store: store, queue: q, messenger: messenger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/lark/events" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	c, err := h.store.CreateCase(r.Context(), caseflow.CreateCaseInput{
		Source:         "lark",
		ChatID:         event.ChatID,
		ThreadID:       event.ThreadID,
		MessageID:      event.MessageID,
		ReporterUserID: event.UserID,
		OriginalText:   event.Text,
		OCRText:        event.OCRText,
		Timezone:       "Asia/Shanghai",
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	_, _ = h.store.AddMessage(r.Context(), caseflow.Message{
		CaseID:        c.ID,
		Role:          "user",
		LarkMessageID: event.MessageID,
		Content:       event.Text,
		ContentType:   "text",
	})
	if err := h.queue.Publish(r.Context(), queue.Event{Type: "case.created", CaseID: c.ID}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	reply := "[" + c.CaseNo + "] 收到，正在检查信息是否足够。"
	_ = h.messenger.SendMessage(r.Context(), event.ChatID, event.ThreadID, reply)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"case_id": c.ID,
		"case_no": c.CaseNo,
		"status":  c.Status,
		"reply":   reply,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
