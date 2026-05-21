package lark

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
)

type Event struct {
	Challenge string   `json:"challenge"`
	Token     string   `json:"token"`
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
	options   Options
	messenger Messenger
}

type Options struct {
	VerificationToken string
	EncryptKey        string
	AllowedChatIDs    []string
	RequireMention    bool
	BotMentionText    string
}

func NewHandler(store caseflow.Store, q queue.Queue, messenger Messenger) *Handler {
	if messenger == nil {
		messenger = LogMessenger{}
	}
	return &Handler{store: store, queue: q, messenger: messenger}
}

func (h *Handler) SetOptions(options Options) {
	h.options = options
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/lark/events" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	body, err = decodeEncryptedEventBody(body, h.options.EncryptKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	event, err := ParseEvent(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if event.Challenge != "" {
		if err := h.verifyToken(event.Token); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"challenge": event.Challenge})
		return
	}
	if err := h.verifyToken(event.Token); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	if !h.chatAllowed(event.ChatID) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "chat is not allowed"})
		return
	}
	if h.options.RequireMention && h.options.BotMentionText != "" && !strings.Contains(event.Text, h.options.BotMentionText) {
		writeJSON(w, http.StatusAccepted, map[string]any{"ignored": true, "reason": "bot was not mentioned"})
		return
	}
	if event.MessageID != "" {
		existing, err := h.store.FindCaseByMessageID(r.Context(), "lark", event.MessageID)
		if err == nil {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"case_id":   existing.ID,
				"case_no":   existing.CaseNo,
				"status":    existing.Status,
				"duplicate": true,
				"ignored":   true,
				"reason":    "message_id was already accepted",
			})
			return
		}
		if !errors.Is(err, caseflow.ErrNotFound) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
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

func ParseEvent(body []byte) (Event, error) {
	var simple Event
	if err := json.Unmarshal(body, &simple); err != nil {
		return Event{}, err
	}
	if simple.Challenge != "" || simple.ChatID != "" || simple.Text != "" {
		return simple, nil
	}

	var v2 struct {
		Challenge string `json:"challenge"`
		Token     string `json:"token"`
		Type      string `json:"type"`
		Header    struct {
			Token   string `json:"token"`
			EventID string `json:"event_id"`
		} `json:"header"`
		Event struct {
			Message struct {
				ChatID    string `json:"chat_id"`
				ThreadID  string `json:"thread_id"`
				RootID    string `json:"root_id"`
				MessageID string `json:"message_id"`
				Content   string `json:"content"`
			} `json:"message"`
			Sender struct {
				SenderID struct {
					OpenID string `json:"open_id"`
					UserID string `json:"user_id"`
				} `json:"sender_id"`
			} `json:"sender"`
		} `json:"event"`
	}
	if err := json.Unmarshal(body, &v2); err != nil {
		return Event{}, err
	}
	if v2.Challenge != "" {
		return Event{Challenge: v2.Challenge, Token: v2.Token}, nil
	}
	event := Event{
		Token:     fallback(v2.Header.Token, v2.Token),
		ChatID:    v2.Event.Message.ChatID,
		ThreadID:  fallback(v2.Event.Message.ThreadID, v2.Event.Message.RootID),
		MessageID: fallback(v2.Event.Message.MessageID, v2.Header.EventID),
		UserID:    fallback(v2.Event.Sender.SenderID.OpenID, v2.Event.Sender.SenderID.UserID),
		Text:      extractText(v2.Event.Message.Content),
	}
	if event.ChatID == "" && event.MessageID == "" && event.Text == "" {
		return Event{}, fmt.Errorf("unsupported lark event payload")
	}
	return event, nil
}

func extractText(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var decoded struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &decoded); err == nil && decoded.Text != "" {
		return decoded.Text
	}
	return content
}

func (h *Handler) verifyToken(token string) error {
	if h.options.VerificationToken == "" {
		return nil
	}
	expectedHash := sha256.Sum256([]byte(h.options.VerificationToken))
	tokenHash := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(expectedHash[:], tokenHash[:]) != 1 {
		return errUnauthorized("invalid lark verification token")
	}
	return nil
}

func (h *Handler) chatAllowed(chatID string) bool {
	if len(h.options.AllowedChatIDs) == 0 {
		return true
	}
	for _, allowed := range h.options.AllowedChatIDs {
		if allowed == chatID {
			return true
		}
	}
	return false
}

type errUnauthorized string

func (e errUnauthorized) Error() string {
	return string(e)
}

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
