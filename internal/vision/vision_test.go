package vision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalClientExtractsPrintableImageBytes(t *testing.T) {
	client := NewLocalClient()
	got, err := client.AnalyzeImages(context.Background(), "用户说 K线不对", []ImageInput{{
		ImageKey:  "img_1",
		MediaType: "text/plain",
		Data:      []byte("BTCUSDT 1m 2026-05-21T20:00:00+08:00 价格不一致"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.OCRText, "BTCUSDT") {
		t.Fatalf("expected printable text in OCR, got %s", got.OCRText)
	}
}

func TestOpenAICompatibleClientSendsImageURLContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compatible-mode/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer key_1" {
			t.Fatalf("unexpected auth header %s", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		messages := body["messages"].([]any)
		user := messages[1].(map[string]any)
		content := user["content"].([]any)
		if content[1].(map[string]any)["type"] != "image_url" {
			t.Fatalf("expected image_url content, got %+v", content)
		}
		imageURL := content[1].(map[string]any)["image_url"].(map[string]any)["url"].(string)
		if !strings.HasPrefix(imageURL, "data:image/png;base64,") {
			t.Fatalf("expected data url, got %s", imageURL)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "OCR: BTCUSDT 1m 价格不一致"}}},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(OpenAICompatibleOptions{
		Provider: "qwen_openai_compatible",
		BaseURL:  server.URL + "/compatible-mode/v1",
		APIKey:   "key_1",
		Model:    "qwen3-vl-plus",
	})
	got, err := client.AnalyzeImages(context.Background(), "用户说 K线不对", []ImageInput{{
		ImageKey:  "img_1",
		MediaType: "image/png",
		Data:      []byte{1, 2, 3},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelName != "qwen3-vl-plus" || !strings.Contains(got.OCRText, "BTCUSDT") {
		t.Fatalf("unexpected analysis: %+v", got)
	}
}
