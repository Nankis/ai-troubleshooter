package webchat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
	"github.com/Nankis/ai-troubleshooter/internal/vision"
	"github.com/Nankis/ai-troubleshooter/web"
)

type CaseProcessor interface {
	ProcessCase(ctx context.Context, caseID int64) (caseflow.ProcessResult, error)
}

type ToolLister interface {
	List() []tool.Spec
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
	tools     ToolLister
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

func (h *Handler) SetToolLister(tools ToolLister) {
	h.tools = tools
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
	if truthy(r.FormValue("async")) {
		go h.processCaseDetached(c.ID)
		h.writeCaseResponse(w, r, caseflow.ProcessResult{
			CaseID: c.ID,
			CaseNo: c.CaseNo,
			Status: c.Status,
			Reply:  "[" + c.CaseNo + "] 已开始排查。",
		}, http.StatusAccepted, true)
		return
	}
	result, err := h.processor.ProcessCase(r.Context(), c.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	h.writeCaseResponse(w, r, result, http.StatusOK, false)
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

func (h *Handler) processCaseDetached(caseID int64) {
	if _, err := h.processor.ProcessCase(context.Background(), caseID); err != nil {
		_, _ = h.store.AddMessage(context.Background(), caseflow.Message{
			CaseID:      caseID,
			Role:        "system",
			Content:     "排查失败：" + err.Error(),
			ContentType: "text",
		})
	}
}

func (h *Handler) writeCaseResponse(w http.ResponseWriter, r *http.Request, result caseflow.ProcessResult, status int, processing bool) {
	c, err := h.store.GetCase(r.Context(), result.CaseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	payload := h.casePayload(r.Context(), c, result, processing)
	writeJSON(w, status, payload)
}

func (h *Handler) casePayload(ctx context.Context, c *caseflow.Case, result caseflow.ProcessResult, processing bool) map[string]any {
	messages, _ := h.store.ListMessages(ctx, c.ID)
	entities, _ := h.store.ListEntities(ctx, c.ID)
	decisionLogs, _ := h.store.ListAIDecisionLogs(ctx, c.ID, 100)
	evolutionRuns, _ := h.store.ListKnowledgeEvolutionRuns(ctx, c.ID)
	if !processing {
		processing = isProcessingStatus(c.Status)
	}
	return map[string]any{
		"case":             c,
		"reply":            result.Reply,
		"tool_call_ids":    result.ToolCallIDs,
		"missing_fields":   result.MissingFields,
		"entities":         entities,
		"messages":         messages,
		"ai_decision_logs": decisionLogs,
		"evolution_runs":   evolutionRuns,
		"progress":         buildProgress(c.Status, decisionLogs),
		"processing":       processing,
	}
}

func (h *Handler) ServeCaseStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	caseRef := strings.Trim(strings.TrimPrefix(r.URL.Path, "/web/api/cases/"), "/")
	if caseRef == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "case not found"})
		return
	}
	c, err := h.loadCase(r.Context(), caseRef)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, h.casePayload(r.Context(), c, caseflow.ProcessResult{CaseID: c.ID, CaseNo: c.CaseNo, Status: c.Status}, false))
}

func (h *Handler) ServeOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	cases, _ := h.store.ListRecentCases(r.Context(), intQuery(r, "case_limit", 30))
	knowledge, _ := h.store.ListKnowledgeItems(r.Context(), caseflow.KnowledgeFilter{Limit: intQuery(r, "knowledge_limit", 30)})
	tools := []tool.Spec{}
	if h.tools != nil {
		tools = h.tools.List()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cases":     cases,
		"tools":     tools,
		"knowledge": knowledge,
		"now":       time.Now(),
	})
}

func (h *Handler) ServeKnowledge(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.store.ListKnowledgeItems(r.Context(), caseflow.KnowledgeFilter{
			IssueDomain: r.URL.Query().Get("issue_domain"),
			IssueType:   r.URL.Query().Get("issue_type"),
			Status:      r.URL.Query().Get("status"),
			Limit:       intQuery(r, "limit", 50),
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var input knowledgeInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		item, err := input.toKnowledgeItem()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		saved, err := h.store.UpsertKnowledgeItem(r.Context(), item)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (h *Handler) ServeKnowledgeItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	rawID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/web/api/knowledge/"), "/")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid knowledge id"})
		return
	}
	if err := h.store.DeleteKnowledgeItem(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, caseflow.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

func (h *Handler) loadCase(ctx context.Context, ref string) (*caseflow.Case, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return h.store.GetCase(ctx, id)
	}
	return h.store.FindCaseByNo(ctx, ref)
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

type progressStep struct {
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func buildProgress(status caseflow.Status, logs []caseflow.AIDecisionLog) []progressStep {
	steps := []progressStep{
		{Key: "classify_issue", Title: "识别问题领域和类型"},
		{Key: "extract_entities", Title: "抽取 uid / trace / 时间窗等实体"},
		{Key: "required_fields_check", Title: "检查必要信息是否齐全"},
		{Key: "knowledge_retrieval", Title: "查询平台沉淀经验"},
		{Key: "decide_next_action", Title: "选择下一步工具和排查策略"},
		{Key: "tool_invocation", Title: "调用只读工具收集证据"},
		{Key: "summarize_findings", Title: "总结证据并输出结论"},
	}
	byKey := map[string][]caseflow.AIDecisionLog{}
	for _, item := range logs {
		byKey[item.DecisionType] = append(byKey[item.DecisionType], item)
	}
	firstPending := -1
	for idx := range steps {
		items := byKey[steps[idx].Key]
		if len(items) == 0 {
			steps[idx].Status = "pending"
			if firstPending < 0 {
				firstPending = idx
			}
			continue
		}
		latest := items[len(items)-1]
		steps[idx].CreatedAt = latest.CreatedAt
		steps[idx].Reason = latest.Reason
		steps[idx].Status = "done"
		if latest.Status == "failed" || latest.Status == "timeout" {
			steps[idx].Status = "failed"
		} else if latest.Status == "skipped" || latest.Status == "need_more_info" {
			steps[idx].Status = latest.Status
		}
	}
	if isProcessingStatus(status) && firstPending >= 0 {
		steps[firstPending].Status = "running"
	}
	if status == caseflow.StatusNeedHumanConfirmation || status == caseflow.StatusDone {
		for idx := range steps {
			if steps[idx].Status == "pending" {
				steps[idx].Status = "skipped"
			}
		}
	}
	return steps
}

func isProcessingStatus(status caseflow.Status) bool {
	switch status {
	case caseflow.StatusReadyToInvestigate, caseflow.StatusInvestigating, caseflow.StatusWaitingToolResult:
		return true
	default:
		return false
	}
}

type knowledgeInput struct {
	ID                 int64    `json:"id"`
	Title              string   `json:"title"`
	IssueDomain        string   `json:"issue_domain"`
	IssueType          string   `json:"issue_type"`
	TypicalDescription string   `json:"typical_description"`
	RecommendedSteps   []string `json:"recommended_steps"`
	UsefulTools        []string `json:"useful_tools"`
}

func (in knowledgeInput) toKnowledgeItem() (caseflow.KnowledgeItem, error) {
	title := strings.TrimSpace(in.Title)
	domain := strings.TrimSpace(in.IssueDomain)
	if title == "" {
		return caseflow.KnowledgeItem{}, fmt.Errorf("title is required")
	}
	if domain == "" {
		return caseflow.KnowledgeItem{}, fmt.Errorf("issue_domain is required")
	}
	return caseflow.KnowledgeItem{
		ID:                   in.ID,
		Title:                title,
		IssueDomain:          domain,
		IssueType:            strings.TrimSpace(in.IssueType),
		TypicalDescription:   strings.TrimSpace(in.TypicalDescription),
		RecommendedStepsJSON: jsonString(in.RecommendedSteps),
		UsefulToolsJSON:      jsonString(in.UsefulTools),
		Confidence:           0.7,
		Status:               "active",
		ObservedCaseCount:    1,
		LastEvolvedAt:        time.Now(),
	}, nil
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func intQuery(r *http.Request, key string, def int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return value
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
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
