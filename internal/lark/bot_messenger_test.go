package lark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBotMessengerRepliesThenFallsBackToChat(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.String())
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			writeTestJSON(w, map[string]any{"code": 0, "tenant_access_token": "token_1", "expire": 7200})
		case "/open-apis/im/v1/messages/om_1/reply":
			if r.Header.Get("Authorization") != "Bearer token_1" {
				t.Fatalf("missing authorization header: %s", r.Header.Get("Authorization"))
			}
			writeTestJSON(w, map[string]any{"code": 230001, "msg": "bad message id"})
		case "/open-apis/im/v1/messages":
			if r.URL.Query().Get("receive_id_type") != "chat_id" {
				t.Fatalf("expected chat_id receive id type, got %s", r.URL.RawQuery)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["receive_id"] != "oc_1" || body["msg_type"] != "text" {
				t.Fatalf("unexpected send body: %+v", body)
			}
			writeTestJSON(w, map[string]any{"code": 0, "msg": "ok"})
		default:
			t.Fatalf("unexpected path %s", r.URL.String())
		}
	}))
	defer server.Close()

	messenger := NewBotMessenger(BotMessengerOptions{
		AppID:     "app_1",
		AppSecret: "secret_1",
		BaseURL:   server.URL,
	})
	if err := messenger.SendMessage(context.Background(), "oc_1", "om_1", "hello"); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected token, reply and fallback send calls, got %v", paths)
	}
}

func TestBotMessengerDefaultsToLarkOpenAPIBaseURL(t *testing.T) {
	messenger := NewBotMessenger(BotMessengerOptions{
		AppID:     "app_1",
		AppSecret: "secret_1",
	})
	if messenger.baseURL != "https://open.larksuite.com" {
		t.Fatalf("expected lark base URL by default, got %s", messenger.baseURL)
	}
}

func TestBotMessengerDownloadsMessageImageResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			writeTestJSON(w, map[string]any{"code": 0, "tenant_access_token": "token_1", "expire": 7200})
		case "/open-apis/im/v1/messages/om_1/resources/img_1":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer token_1" {
				t.Fatalf("missing authorization header: %s", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("fake image bytes BTCUSDT"))
		default:
			t.Fatalf("unexpected path %s", r.URL.String())
		}
	}))
	defer server.Close()

	messenger := NewBotMessenger(BotMessengerOptions{
		AppID:     "app_1",
		AppSecret: "secret_1",
		BaseURL:   server.URL,
	})
	image, err := messenger.DownloadImage(context.Background(), "om_1", "img_1")
	if err != nil {
		t.Fatal(err)
	}
	if image.MediaType != "image/png" || !strings.Contains(string(image.Data), "BTCUSDT") {
		t.Fatalf("unexpected downloaded image: %+v", image)
	}
}

func writeTestJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
