package lark

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/queue"
)

func TestHandlerRejectsDisallowedChat(t *testing.T) {
	handler := NewHandler(caseflow.NewInMemoryStore(), queue.NewMemoryQueue(1), nil)
	handler.SetOptions(Options{AllowedChatIDs: []string{"oc_allowed"}})

	req := httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(`{"chat_id":"oc_denied","text":"@bot hi"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRespondsToLarkChallenge(t *testing.T) {
	handler := NewHandler(caseflow.NewInMemoryStore(), queue.NewMemoryQueue(1), nil)
	handler.SetOptions(Options{VerificationToken: "token_1"})

	req := httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(`{"token":"token_1","challenge":"challenge_1"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "challenge_1") {
		t.Fatalf("expected challenge response, got %s", rec.Body.String())
	}
}

func TestParseEventSupportsLarkV2Message(t *testing.T) {
	event, err := ParseEvent([]byte(`{
		"schema":"2.0",
		"header":{"event_id":"evt_1","token":"token_1"},
		"event":{
			"sender":{"sender_id":{"open_id":"ou_1"}},
			"message":{
				"message_id":"msg_1",
				"chat_id":"oc_1",
				"root_id":"root_1",
				"content":"{\"text\":\"@排障机器人 BTCUSDT 1m K线不对\"}"
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if event.Token != "token_1" || event.ChatID != "oc_1" || event.ThreadID != "root_1" || event.UserID != "ou_1" {
		t.Fatalf("unexpected event metadata: %+v", event)
	}
	if event.Text != "@排障机器人 BTCUSDT 1m K线不对" {
		t.Fatalf("unexpected text %q", event.Text)
	}
}
