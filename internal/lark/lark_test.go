package lark

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/queue"
	"github.com/Nankis/ai-troubleshooter/internal/vision"
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

func TestHandlerRespondsToEncryptedLarkChallenge(t *testing.T) {
	encryptKey := "encrypt_key_1"
	handler := NewHandler(caseflow.NewInMemoryStore(), queue.NewMemoryQueue(1), nil)
	handler.SetOptions(Options{VerificationToken: "token_1", EncryptKey: encryptKey})
	body := encryptedTestEnvelope(t, encryptKey, []byte(`{"token":"token_1","challenge":"challenge_1"}`))

	req := httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "challenge_1") {
		t.Fatalf("expected challenge response, got %s", rec.Body.String())
	}
}

func TestHandlerAcceptsEncryptedLarkMessage(t *testing.T) {
	encryptKey := "encrypt_key_1"
	store := caseflow.NewInMemoryStore()
	q := &recordingQueue{}
	handler := NewHandler(store, q, nil)
	handler.SetOptions(Options{VerificationToken: "token_1", EncryptKey: encryptKey})
	body := encryptedTestEnvelope(t, encryptKey, []byte(`{
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

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(body)))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if q.published != 1 {
		t.Fatalf("expected one queue publish, got %d", q.published)
	}
	c, err := store.FindCaseByMessageID(context.Background(), "lark", "msg_1")
	if err != nil {
		t.Fatal(err)
	}
	if c.ChatID != "oc_1" || c.ReporterUserID != "ou_1" {
		t.Fatalf("unexpected case from encrypted message: %+v", c)
	}
}

func TestHandlerAcceptsFeishuEventPath(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	q := &recordingQueue{}
	handler := NewHandler(store, q, nil)
	body := `{"chat_id":"oc_1","message_id":"msg_feishu_1","user_id":"ou_1","text":"@bot BTCUSDT 1m K线价格不一致"}`

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/feishu/events", strings.NewReader(body)))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := store.FindCaseByMessageID(context.Background(), "feishu", "msg_feishu_1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FindCaseByMessageID(context.Background(), "lark", "msg_feishu_1"); err == nil {
		t.Fatal("expected feishu event to use feishu source, but found lark source")
	}
}

func TestHandlerRejectsEncryptedPayloadWithoutKey(t *testing.T) {
	body := encryptedTestEnvelope(t, "encrypt_key_1", []byte(`{"token":"token_1","challenge":"challenge_1"}`))
	handler := NewHandler(caseflow.NewInMemoryStore(), queue.NewMemoryQueue(1), nil)
	handler.SetOptions(Options{VerificationToken: "token_1"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(body)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRejectsPlainPayloadWhenEncryptKeyConfigured(t *testing.T) {
	handler := NewHandler(caseflow.NewInMemoryStore(), queue.NewMemoryQueue(1), nil)
	handler.SetOptions(Options{VerificationToken: "token_1", EncryptKey: "encrypt_key_1"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(`{"token":"token_1","challenge":"challenge_1"}`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
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

func TestParseEventSupportsLarkV2ImageMessage(t *testing.T) {
	event, err := ParseEvent([]byte(`{
		"schema":"2.0",
		"header":{"event_id":"evt_img","token":"token_1"},
		"event":{
			"sender":{"sender_id":{"open_id":"ou_1"}},
			"message":{
				"message_id":"msg_img",
				"chat_id":"oc_1",
				"root_id":"root_1",
				"message_type":"image",
				"content":"{\"image_key\":\"img_1\"}"
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if event.MessageID != "msg_img" || len(event.ImageKeys) != 1 || event.ImageKeys[0] != "img_1" {
		t.Fatalf("unexpected image event: %+v", event)
	}
}

func TestHandlerDownloadsAndAnalyzesImages(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	q := &recordingQueue{}
	handler := NewHandler(store, q, nil)
	handler.SetImageProcessor(fakeImageDownloader{}, fakeVisionClient{}, ImageOptions{MaxImages: 3, MaxImageBytes: 1024})
	body := `{"chat_id":"oc_1","message_id":"msg_img","user_id":"ou_1","text":"@bot 帮忙看截图","image_keys":["img_1"]}`

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(body)))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	c, err := store.FindCaseByMessageID(context.Background(), "lark", "msg_img")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.OCRText, "BTCUSDT") || !strings.Contains(c.OCRText, "qwen-test") {
		t.Fatalf("expected image OCR in case, got %s", c.OCRText)
	}
}

func TestHandlerIgnoresDuplicateMessageID(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	q := &recordingQueue{}
	handler := NewHandler(store, q, nil)
	body := `{"chat_id":"oc_1","message_id":"msg_1","user_id":"ou_1","text":"@bot BTCUSDT 1m K线价格不一致"}`

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(body)))
	if first.Code != http.StatusAccepted {
		t.Fatalf("expected first event to be accepted, got %d body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/lark/events", strings.NewReader(body)))
	if second.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate event to be accepted idempotently, got %d body=%s", second.Code, second.Body.String())
	}
	if q.published != 1 {
		t.Fatalf("expected one queue publish, got %d", q.published)
	}
	var duplicateResp map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &duplicateResp); err != nil {
		t.Fatal(err)
	}
	if duplicateResp["duplicate"] != true || duplicateResp["ignored"] != true {
		t.Fatalf("expected duplicate ignored response, got %s", second.Body.String())
	}
}

type recordingQueue struct {
	published int
	events    []queue.Event
}

func (q *recordingQueue) Publish(ctx context.Context, event queue.Event) error {
	_ = ctx
	q.published++
	q.events = append(q.events, event)
	return nil
}

func (q *recordingQueue) Consume(ctx context.Context) (queue.Event, error) {
	<-ctx.Done()
	return queue.Event{}, ctx.Err()
}

type fakeImageDownloader struct{}

func (fakeImageDownloader) DownloadImage(ctx context.Context, messageID string, imageKey string) (DownloadedImage, error) {
	_ = ctx
	return DownloadedImage{ImageKey: imageKey, MediaType: "image/png", Data: []byte("image bytes")}, nil
}

type fakeVisionClient struct{}

func (fakeVisionClient) AnalyzeImages(ctx context.Context, userText string, images []vision.ImageInput) (vision.Analysis, error) {
	_ = ctx
	_ = userText
	return vision.Analysis{ModelProvider: "qwen-test", ModelName: "qwen-vl-test", OCRText: "截图文字：BTCUSDT 1m 价格不一致"}, nil
}

func encryptedTestEnvelope(t *testing.T, encryptKey string, plaintext []byte) string {
	t.Helper()
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(ciphertext, padded)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	body, err := json.Marshal(map[string]string{"encrypt": encoded})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func pkcs7Pad(value []byte, blockSize int) []byte {
	padding := blockSize - len(value)%blockSize
	out := make([]byte, 0, len(value)+padding)
	out = append(out, value...)
	for i := 0; i < padding; i++ {
		out = append(out, byte(padding))
	}
	return out
}
