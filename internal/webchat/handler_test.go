package webchat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/capability"
	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
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

type fakeToolLister struct{}

func (fakeToolLister) List() []tool.Spec {
	return []tool.Spec{{Name: "search_logs_by_service", Description: "查服务日志", RequiredScope: "logs:read_summary"}}
}

func TestServeChatCreatesCaseFromText(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	handler := New(store, fakeProcessor{}, fakeVision{}, Options{})

	req := multipartRequest(t, map[string]string{"message": "BTCUSDT K线最高价不一致", "title": "行情异常排查"}, nil)
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
	if payload.Case.Source != "web" || payload.Case.UID != "web_user" || payload.Case.Title != "行情异常排查" || payload.Reply != "ok" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.ToolCallIDs) != 1 || payload.ToolCallIDs[0] != "tc_test" {
		t.Fatalf("unexpected tool ids: %+v", payload.ToolCallIDs)
	}
}

func TestServeCaseRenameAndDelete(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	handler := New(store, fakeProcessor{}, fakeVision{}, Options{})
	c, err := store.CreateCase(context.Background(), caseflow.CreateCaseInput{
		Title:        "旧标题",
		UID:          "web_user",
		Source:       "web",
		ChatID:       "web-local",
		MessageID:    "msg_case",
		OriginalText: "用户反馈 token 数量不对",
	})
	if err != nil {
		t.Fatal(err)
	}

	renameReq := httptest.NewRequest(http.MethodPatch, "/web/api/cases/"+c.CaseNo, strings.NewReader(`{"title":"新标题"}`))
	renameReq.Header.Set("Content-Type", "application/json")
	renameRec := httptest.NewRecorder()
	handler.ServeCaseStatus(renameRec, renameReq)
	if renameRec.Code != http.StatusOK {
		t.Fatalf("unexpected rename status %d body=%s", renameRec.Code, renameRec.Body.String())
	}
	var renamed caseflow.Case
	if err := json.Unmarshal(renameRec.Body.Bytes(), &renamed); err != nil {
		t.Fatal(err)
	}
	if renamed.Title != "新标题" {
		t.Fatalf("expected renamed title, got %+v", renamed)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/web/api/cases/"+c.CaseNo, nil)
	deleteRec := httptest.NewRecorder()
	handler.ServeCaseStatus(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("unexpected delete status %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	if _, err := store.FindCaseByNo(context.Background(), c.CaseNo); !errors.Is(err, caseflow.ErrNotFound) {
		t.Fatalf("expected deleted case not found, got %v", err)
	}
}

func TestServeChatAsyncReturnsStatusAndProgress(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	processor := &waitingProcessor{store: store, started: make(chan struct{}), release: make(chan struct{})}
	handler := New(store, processor, fakeVision{}, Options{})

	req := multipartRequest(t, map[string]string{"message": "health-food 今日推荐没生成", "async": "1"}, nil)
	rec := httptest.NewRecorder()
	handler.ServeChat(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status %d body=%s", rec.Code, rec.Body.String())
	}
	var initial struct {
		Case       caseflow.Case `json:"case"`
		Processing bool          `json:"processing"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if !initial.Processing || initial.Case.CaseNo == "" {
		t.Fatalf("expected async case, got %+v", initial)
	}

	select {
	case <-processor.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}

	reqStatus := httptest.NewRequest(http.MethodGet, "/web/api/cases/"+initial.Case.CaseNo, nil)
	recStatus := httptest.NewRecorder()
	handler.ServeCaseStatus(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var status struct {
		Processing bool `json:"processing"`
		Progress   []struct {
			Key    string `json:"key"`
			Status string `json:"status"`
		} `json:"progress"`
	}
	if err := json.Unmarshal(recStatus.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Processing {
		t.Fatalf("expected processing status: %+v", status)
	}
	if len(status.Progress) == 0 || status.Progress[0].Key != "classify_issue" || status.Progress[0].Status != "done" {
		t.Fatalf("expected progress from decision logs, got %+v", status.Progress)
	}

	close(processor.release)
}

func TestServeOverviewAndKnowledgeMutation(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	capStore := capability.NewMemoryStore()
	handler := New(store, fakeProcessor{}, fakeVision{}, Options{})
	handler.SetToolLister(fakeToolLister{})
	handler.SetCapabilityStore(capStore)
	if _, err := capStore.UpsertToolCapability(context.Background(), capability.ToolCapability{
		ToolName:         "get_demo_status",
		Description:      "查询 demo 状态",
		ServiceName:      "demo",
		SourceType:       capability.SourceHTTPAdapter,
		InputSchemaJSON:  `{"type":"object"}`,
		RequiredScope:    "dynamic:read",
		BackendHandler:   "dynamic_http.get_demo_status",
		ReadonlyBaseURL:  "http://127.0.0.1:19081",
		ReadonlyPath:     "/v1/readonly/demo/status",
		HTTPMethod:       "POST",
		SafetyStatus:     capability.SafetyReadonlyCandidate,
		ApprovalStatus:   "pending",
		ValidationStatus: "not_run",
		ToolStatus:       capability.StatusDraft,
	}); err != nil {
		t.Fatal(err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/web/api/knowledge", strings.NewReader(`{
		"title":"health-food 推荐缺失",
		"issue_domain":"health_food",
		"issue_type":"每日推荐缺失",
		"typical_description":"有餐食但没有推荐",
		"recommended_steps":["查餐食","查推荐状态"],
		"useful_tools":["get_health_food_recommendation_status"]
	}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.ServeKnowledge(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("unexpected create status %d body=%s", createRec.Code, createRec.Body.String())
	}
	var item caseflow.KnowledgeItem
	if err := json.Unmarshal(createRec.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID == 0 || !strings.Contains(item.RecommendedStepsJSON, "查餐食") {
		t.Fatalf("unexpected knowledge item: %+v", item)
	}
	updateReq := httptest.NewRequest(http.MethodPut, "/web/api/knowledge/"+strconvFormat(item.ID), strings.NewReader(`{
		"title":"health-food 推荐缺失更新",
		"issue_domain":"health_food",
		"issue_type":"每日推荐缺失",
		"typical_description":"有餐食但没有推荐，且日志有 quota 告警",
		"recommended_steps":["查用户资料","查推荐状态","查日志"],
		"common_causes":["额度耗尽"],
		"useful_tools":["get_health_food_user_profile","search_logs_by_service"]
	}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	handler.ServeKnowledgeItem(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("unexpected update status %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updated caseflow.KnowledgeItem
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != item.ID || !strings.Contains(updated.Title, "更新") || !strings.Contains(updated.CommonCausesJSON, "额度耗尽") {
		t.Fatalf("unexpected updated knowledge item: %+v", updated)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/web/api/knowledge/"+strconvFormat(item.ID), nil)
	getRec := httptest.NewRecorder()
	handler.ServeKnowledgeItem(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status %d body=%s", getRec.Code, getRec.Body.String())
	}

	overviewReq := httptest.NewRequest(http.MethodGet, "/web/api/overview", nil)
	overviewRec := httptest.NewRecorder()
	handler.ServeOverview(overviewRec, overviewReq)
	if overviewRec.Code != http.StatusOK {
		t.Fatalf("unexpected overview status %d body=%s", overviewRec.Code, overviewRec.Body.String())
	}
	var overview struct {
		Tools        []tool.Spec                 `json:"tools"`
		Capabilities []capability.ToolCapability `json:"capabilities"`
		Knowledge    []caseflow.KnowledgeItem    `json:"knowledge"`
	}
	if err := json.Unmarshal(overviewRec.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Tools) != 1 || len(overview.Knowledge) != 1 || len(overview.Capabilities) != 1 {
		t.Fatalf("unexpected overview: %+v", overview)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/web/api/knowledge/"+strconvFormat(item.ID), nil)
	deleteRec := httptest.NewRecorder()
	handler.ServeKnowledgeItem(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("unexpected delete status %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	items, err := store.ListKnowledgeItems(context.Background(), caseflow.KnowledgeFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected soft-deleted knowledge to be hidden, got %+v", items)
	}
}

func TestServeCapabilityImportAndPublish(t *testing.T) {
	store := caseflow.NewInMemoryStore()
	capStore := capability.NewMemoryStore()
	handler := New(store, fakeProcessor{}, fakeVision{}, Options{})
	handler.SetCapabilityStore(capStore)
	reloads := 0
	handler.SetCapabilityReloader(func(ctx context.Context) error {
		_ = ctx
		reloads++
		return nil
	})

	importReq := httptest.NewRequest(http.MethodPost, "/web/api/capabilities/import", strings.NewReader(`{
		"raw_config": "{\"service\":{\"service_name\":\"demo\",\"base_url\":\"http://127.0.0.1:19081\"},\"capabilities\":[{\"tool_name\":\"get_demo_status\",\"description\":\"查询 demo 状态\",\"path\":\"/v1/readonly/demo/status\",\"required_params\":[\"uid\"]}]}"
	}`))
	importReq.Header.Set("Content-Type", "application/json")
	importRec := httptest.NewRecorder()
	handler.ServeCapabilityImport(importRec, importReq)
	if importRec.Code != http.StatusCreated {
		t.Fatalf("unexpected import status %d body=%s", importRec.Code, importRec.Body.String())
	}
	var imported capability.ImportResult
	if err := json.Unmarshal(importRec.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if len(imported.Capabilities) != 1 || imported.Capabilities[0].ToolStatus != capability.StatusDraft {
		t.Fatalf("unexpected imported capabilities: %+v", imported.Capabilities)
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/web/api/capabilities/"+strconvFormat(imported.Capabilities[0].ID)+"/publish", nil)
	publishRec := httptest.NewRecorder()
	handler.ServeCapabilityItem(publishRec, publishReq)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("unexpected publish status %d body=%s", publishRec.Code, publishRec.Body.String())
	}
	if reloads != 1 {
		t.Fatalf("expected one reload, got %d", reloads)
	}
	var published capability.ToolCapability
	if err := json.Unmarshal(publishRec.Body.Bytes(), &published); err != nil {
		t.Fatal(err)
	}
	if published.ToolStatus != capability.StatusEnabled || published.PublishedAt == nil {
		t.Fatalf("expected enabled published capability, got %+v", published)
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

type waitingProcessor struct {
	store   caseflow.Store
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *waitingProcessor) ProcessCase(ctx context.Context, caseID int64) (caseflow.ProcessResult, error) {
	_ = ctx
	c, err := p.store.GetCase(context.Background(), caseID)
	if err != nil {
		return caseflow.ProcessResult{}, err
	}
	_, _ = p.store.UpdateCase(context.Background(), caseID, c.Version, func(next *caseflow.Case) error {
		next.Status = caseflow.StatusInvestigating
		return nil
	})
	_, _ = p.store.AddAIDecisionLog(context.Background(), caseflow.AIDecisionLog{
		CaseID:       caseID,
		AgentID:      "test-agent",
		DecisionType: "classify_issue",
		Reason:       "test progress",
		Status:       "success",
	})
	p.once.Do(func() { close(p.started) })
	<-p.release
	return caseflow.ProcessResult{CaseID: caseID, CaseNo: c.CaseNo, Status: caseflow.StatusNeedHumanConfirmation, Reply: "done"}, nil
}

func strconvFormat(value int64) string {
	return strconv.FormatInt(value, 10)
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
