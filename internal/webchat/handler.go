package webchat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/vision"
	"github.com/Nankis/ai-troubleshooter/web"
)

type CaseProcessor interface {
	ProcessCase(ctx context.Context, caseID int64) (caseflow.ProcessResult, error)
}

type Options struct {
	MaxImages     int
	MaxImageBytes int64
}

type Handler struct {
	store     caseflow.Store
	processor CaseProcessor
	vision    vision.Client
	options   Options
}

func New(store caseflow.Store, processor CaseProcessor, visionClient vision.Client, options Options) *Handler {
	if options.MaxImages <= 0 {
		options.MaxImages = 3
	}
	if options.MaxImageBytes <= 0 {
		options.MaxImageBytes = 10 * 1024 * 1024
	}
	if visionClient == nil {
		visionClient = vision.NewLocalClient()
	}
	return &Handler{store: store, processor: processor, vision: visionClient, options: options}
}

func (h *Handler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if r.URL.Path != "/" && r.URL.Path != "/web" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, web.IndexHTML)
}

func (h *Handler) ServeChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes(h.options))
	if err := r.ParseMultipartForm(maxRequestBytes(h.options)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid multipart form: " + err.Error()})
		return
	}

	userText := strings.TrimSpace(r.FormValue("message"))
	caseNo := strings.TrimSpace(r.FormValue("case_no"))
	images, err := h.readImages(r.MultipartForm)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if userText == "" && len(images) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "message or image is required"})
		return
	}

	analysis := vision.Analysis{}
	if len(images) > 0 {
		analysis, err = h.vision.AnalyzeImages(r.Context(), userText, images)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "vision analysis failed: " + err.Error()})
			return
		}
	}

	c, err := h.upsertCase(r.Context(), caseNo, userText, analysis)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	result, err := h.processor.ProcessCase(r.Context(), c.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	h.writeCaseResponse(w, r, result)
}

func (h *Handler) upsertCase(ctx context.Context, caseNo string, userText string, analysis vision.Analysis) (*caseflow.Case, error) {
	if caseNo != "" {
		c, err := h.store.FindCaseByNo(ctx, caseNo)
		if err != nil {
			return nil, err
		}
		if _, err := h.store.AddMessage(ctx, caseflow.Message{CaseID: c.ID, Role: "user", Content: messageContent(userText, analysis), ContentType: "text"}); err != nil {
			return nil, err
		}
		return h.store.UpdateCase(ctx, c.ID, c.Version, func(next *caseflow.Case) error {
			next.OriginalText = appendBlock(next.OriginalText, "用户补充", userText)
			next.OCRText = appendBlock(next.OCRText, "图片识别补充", analysis.OCRText)
			return nil
		})
	}
	c, err := h.store.CreateCase(ctx, caseflow.CreateCaseInput{
		UID:            "web_user",
		Source:         "web",
		ChatID:         "web-local",
		ThreadID:       newID("thread"),
		MessageID:      newID("webmsg"),
		ReporterUserID: "web_user",
		OriginalText:   userText,
		OCRText:        analysis.OCRText,
		Timezone:       "Asia/Shanghai",
	})
	if err != nil {
		return nil, err
	}
	if _, err := h.store.AddMessage(ctx, caseflow.Message{CaseID: c.ID, Role: "user", Content: messageContent(userText, analysis), ContentType: "text"}); err != nil {
		return nil, err
	}
	return c, nil
}

func (h *Handler) writeCaseResponse(w http.ResponseWriter, r *http.Request, result caseflow.ProcessResult) {
	c, err := h.store.GetCase(r.Context(), result.CaseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	messages, _ := h.store.ListMessages(r.Context(), c.ID)
	entities, _ := h.store.ListEntities(r.Context(), c.ID)
	decisionLogs, _ := h.store.ListAIDecisionLogs(r.Context(), c.ID, 100)
	writeJSON(w, http.StatusOK, map[string]any{
		"case":             c,
		"reply":            result.Reply,
		"tool_call_ids":    result.ToolCallIDs,
		"missing_fields":   result.MissingFields,
		"entities":         entities,
		"messages":         messages,
		"ai_decision_logs": decisionLogs,
	})
}

func (h *Handler) readImages(form *multipart.Form) ([]vision.ImageInput, error) {
	if form == nil || form.File == nil {
		return nil, nil
	}
	files := append([]*multipart.FileHeader{}, form.File["images"]...)
	files = append(files, form.File["image"]...)
	if len(files) > h.options.MaxImages {
		return nil, fmt.Errorf("too many images: max %d", h.options.MaxImages)
	}
	out := make([]vision.ImageInput, 0, len(files))
	for _, header := range files {
		if header.Size > h.options.MaxImageBytes {
			return nil, fmt.Errorf("image %s exceeds max %d bytes", header.Filename, h.options.MaxImageBytes)
		}
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, h.options.MaxImageBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if int64(len(data)) > h.options.MaxImageBytes {
			return nil, fmt.Errorf("image %s exceeds max %d bytes", header.Filename, h.options.MaxImageBytes)
		}
		mediaType := header.Header.Get("Content-Type")
		if mediaType == "" {
			mediaType = http.DetectContentType(data)
		}
		if !strings.HasPrefix(mediaType, "image/") {
			return nil, fmt.Errorf("file %s is not an image", header.Filename)
		}
		out = append(out, vision.ImageInput{ImageKey: header.Filename, MediaType: mediaType, Data: data})
	}
	return out, nil
}

func maxRequestBytes(options Options) int64 {
	maxImages := options.MaxImages
	if maxImages <= 0 {
		maxImages = 3
	}
	maxImageBytes := options.MaxImageBytes
	if maxImageBytes <= 0 {
		maxImageBytes = 10 * 1024 * 1024
	}
	return int64(maxImages)*maxImageBytes + 1024*1024
}

func messageContent(userText string, analysis vision.Analysis) string {
	parts := []string{}
	if strings.TrimSpace(userText) != "" {
		parts = append(parts, userText)
	}
	if strings.TrimSpace(analysis.OCRText) != "" {
		parts = append(parts, "图片识别：\n"+analysis.OCRText)
	}
	return strings.Join(parts, "\n\n")
}

func appendBlock(current string, label string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return current
	}
	block := fmt.Sprintf("%s：%s", label, value)
	if strings.TrimSpace(current) == "" {
		return block
	}
	return current + "\n" + block
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
