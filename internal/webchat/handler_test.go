package webchat

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/vision"
)

type fakeProcessor struct{}

func (fakeProcessor) ProcessCase(ctx context.Context, caseID int64) (caseflow.ProcessResult, error) {
	_ = ctx
	return caseflow.ProcessResult{
		CaseID:      caseID,
		Status:      caseflow.StatusWaitingUserReply,
		Reply:       "ok",
		ToolCallIDs: []string{"tc_test"},
	}, nil
}

type fakeVision struct{}

func (fakeVision) AnalyzeImages(ctx context.Context, userText string, images []vision.ImageInput) (vision.Analysis, error) {
	_ = ctx
	_ = userText
	if len(images) != 1 {
		return vision.Analysis{}, nil
	}
	return vision.Analysis{ModelProvider: "test", ModelName: "vision-test", OCRText: "Symbol: BTCUSDT\nInterval: 1m"}, nil
}

func TestServeChatCreatesCaseFromText(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	handler := New(store, fakeProcessor{}, fakeVision{}, Options{})

	req := multipartRequest(t, map[string]string{"message": "BTCUSDT K线最高价不一致"}, nil)
	rec := httptest.NewRecorder()
	handler.ServeChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Case        caseflow.Case `json:"case"`
		Reply       string        `json:"reply"`
		ToolCallIDs []string      `json:"tool_call_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Case.Source != "web" || payload.Case.UID != "web_user" || payload.Reply != "ok" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.ToolCallIDs) != 1 || payload.ToolCallIDs[0] != "tc_test" {
		t.Fatalf("unexpected tool ids: %+v", payload.ToolCallIDs)
	}
}

func TestServeChatAcceptsImageAndStoresOCR(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	handler := New(store, fakeProcessor{}, fakeVision{}, Options{})

	req := multipartRequest(t, map[string]string{"message": "请看截图"}, []filePart{{
		field:       "images",
		filename:    "ticket.png",
		contentType: "image/png",
		data:        []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
	}})
	rec := httptest.NewRecorder()
	handler.ServeChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Case caseflow.Case `json:"case"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload.Case.OCRText, "BTCUSDT") {
		t.Fatalf("expected OCR text on case, got %q", payload.Case.OCRText)
	}
}

type filePart struct {
	field       string
	filename    string
	contentType string
	data        []byte
}

func multipartRequest(t *testing.T, fields map[string]string, files []filePart) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="`+file.field+`"; filename="`+file.filename+`"`)
		header.Set("Content-Type", file.contentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/web/api/chat", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
