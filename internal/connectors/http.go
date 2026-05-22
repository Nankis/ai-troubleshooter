package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
	Client  *http.Client
}

type HTTPKlineConnector struct {
	client readonlyHTTPClient
}

type HTTPAssetConnector struct {
	client readonlyHTTPClient
}

type HTTPOpsConnector struct {
	client readonlyHTTPClient
}

type HTTPHealthFoodConnector struct {
	client readonlyHTTPClient
}

func NewHTTPKlineConnector(cfg HTTPConfig) (*HTTPKlineConnector, error) {
	client, err := newReadonlyHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return &HTTPKlineConnector{client: client}, nil
}

func NewHTTPAssetConnector(cfg HTTPConfig) (*HTTPAssetConnector, error) {
	client, err := newReadonlyHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return &HTTPAssetConnector{client: client}, nil
}

func NewHTTPOpsConnector(cfg HTTPConfig) (*HTTPOpsConnector, error) {
	client, err := newReadonlyHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return &HTTPOpsConnector{client: client}, nil
}

func NewHTTPHealthFoodConnector(cfg HTTPConfig) (*HTTPHealthFoodConnector, error) {
	client, err := newReadonlyHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return &HTTPHealthFoodConnector{client: client}, nil
}

func (c *HTTPKlineConnector) InternalKline(ctx context.Context, q KlineQuery) (InternalKlineResult, error) {
	var out readonlyResponse[struct {
		Candles []Candle `json:"candles"`
	}]
	err := c.client.post(ctx, "/v1/readonly/market/kline/internal", q, &out)
	return InternalKlineResult{
		Candles:       out.Data.Candles,
		DataUpdatedAt: out.DataUpdatedAt,
		Source:        out.Source,
		QueriedAt:     out.QueriedAt,
	}, err
}

func (c *HTTPKlineConnector) ExternalKlineCompare(ctx context.Context, q KlineQuery) (KlineCompareResult, error) {
	var out readonlyResponse[KlineCompareResult]
	err := c.client.post(ctx, "/v1/readonly/market/kline/compare", q, &out)
	return out.Data, err
}

func (c *HTTPKlineConnector) KlineCacheStatus(ctx context.Context, q KlineQuery) (CacheStatus, error) {
	var out readonlyResponse[CacheStatus]
	err := c.client.post(ctx, "/v1/readonly/market/kline/cache-status", q, &out)
	return out.Data, err
}

func (c *HTTPKlineConnector) MarketSourceStatus(ctx context.Context, q KlineQuery) (MarketSourceStatus, error) {
	var out readonlyResponse[MarketSourceStatus]
	err := c.client.post(ctx, "/v1/readonly/market/source/status", q, &out)
	return out.Data, err
}

func (c *HTTPAssetConnector) AssetSnapshot(ctx context.Context, q AssetQuery) (AssetSnapshot, error) {
	var out readonlyResponse[AssetSnapshot]
	err := c.client.post(ctx, "/v1/readonly/asset/snapshot", q, &out)
	return out.Data, err
}

func (c *HTTPAssetConnector) AssetEvents(ctx context.Context, q AssetQuery) (AssetEventsResult, error) {
	var out readonlyResponse[AssetEventsResult]
	err := c.client.post(ctx, "/v1/readonly/asset/events", q, &out)
	return out.Data, err
}

func (c *HTTPAssetConnector) UserRecentErrors(ctx context.Context, q AssetQuery) (UserErrorsResult, error) {
	var out readonlyResponse[UserErrorsResult]
	err := c.client.post(ctx, "/v1/readonly/asset/user-errors", q, &out)
	return out.Data, err
}

func (c *HTTPOpsConnector) SearchLogs(ctx context.Context, q LogQuery) (LogSearchResult, error) {
	var out readonlyResponse[LogSearchResult]
	err := c.client.post(ctx, "/v1/readonly/ops/logs/search", q, &out)
	return out.Data, err
}

func (c *HTTPOpsConnector) RecentDeployments(ctx context.Context, serviceName string, startTime time.Time, endTime time.Time) (DeploymentResult, error) {
	var out readonlyResponse[DeploymentResult]
	err := c.client.post(ctx, "/v1/readonly/ops/deployments/recent", map[string]any{
		"service_name": serviceName,
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     endTime.Format(time.RFC3339),
	}, &out)
	return out.Data, err
}

func (c *HTTPOpsConnector) SimilarCases(ctx context.Context, issueDomain string, issueType string, text string, entities map[string]any, limit int) (SimilarCaseResult, error) {
	var out readonlyResponse[SimilarCaseResult]
	err := c.client.post(ctx, "/v1/readonly/ops/cases/similar", map[string]any{
		"issue_domain": issueDomain,
		"issue_type":   issueType,
		"text":         text,
		"entities":     entities,
		"limit":        limit,
	}, &out)
	return out.Data, err
}

func (c *HTTPHealthFoodConnector) UserProfile(ctx context.Context, q HealthFoodQuery) (HealthFoodUserProfile, error) {
	var out readonlyResponse[HealthFoodUserProfile]
	err := c.client.post(ctx, "/v1/readonly/health-food/user/profile", healthFoodParams(q), &out)
	return out.Data, err
}

func (c *HTTPHealthFoodConnector) AIQuota(ctx context.Context, q HealthFoodQuery) (HealthFoodAIQuota, error) {
	var out readonlyResponse[HealthFoodAIQuota]
	err := c.client.post(ctx, "/v1/readonly/health-food/ai/quota", healthFoodParams(q), &out)
	return out.Data, err
}

func (c *HTTPHealthFoodConnector) MealRecords(ctx context.Context, q HealthFoodQuery) (HealthFoodMealRecords, error) {
	var out readonlyResponse[HealthFoodMealRecords]
	err := c.client.post(ctx, "/v1/readonly/health-food/meals/range", healthFoodParams(q), &out)
	return out.Data, err
}

func (c *HTTPHealthFoodConnector) RecommendationStatus(ctx context.Context, q HealthFoodQuery) (HealthFoodRecommendationStatus, error) {
	var out readonlyResponse[HealthFoodRecommendationStatus]
	err := c.client.post(ctx, "/v1/readonly/health-food/recommendation/status", healthFoodParams(q), &out)
	return out.Data, err
}

func healthFoodParams(q HealthFoodQuery) map[string]any {
	params := map[string]any{
		"user_id": q.UserID,
		"uid":     q.UID,
	}
	if q.StartTime.IsZero() {
		params["start_time"] = ""
	} else {
		params["start_time"] = q.StartTime.Format(time.RFC3339)
	}
	if q.EndTime.IsZero() {
		params["end_time"] = ""
	} else {
		params["end_time"] = q.EndTime.Format(time.RFC3339)
	}
	if !q.AtTime.IsZero() {
		params["at_time"] = q.AtTime.Format(time.RFC3339)
	}
	if q.RecommendationDate != "" {
		params["recommendation_date"] = q.RecommendationDate
	}
	if q.TraceID != "" {
		params["trace_id"] = q.TraceID
	}
	if q.Limit > 0 {
		params["limit"] = q.Limit
	}
	return params
}

type readonlyHTTPClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	timeout time.Duration
}

type readonlyRequest struct {
	RequestID    string `json:"request_id"`
	CaseID       string `json:"case_id,omitempty"`
	AgentID      string `json:"agent_id,omitempty"`
	CallerUserID string `json:"caller_user_id,omitempty"`
	ToolName     string `json:"tool_name,omitempty"`
	TimeoutMS    int64  `json:"timeout_ms,omitempty"`
	Params       any    `json:"params"`
}

type readonlyResponse[T any] struct {
	RequestID     string    `json:"request_id"`
	Source        string    `json:"source"`
	QueriedAt     time.Time `json:"queried_at"`
	DataUpdatedAt time.Time `json:"data_updated_at"`
	Version       string    `json:"version"`
	Data          T         `json:"data"`
	Warnings      []string  `json:"warnings"`
}

type readonlyErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func newReadonlyHTTPClient(cfg HTTPConfig) (readonlyHTTPClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return readonlyHTTPClient{}, fmt.Errorf("readonly connector base url is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return readonlyHTTPClient{}, fmt.Errorf("invalid readonly connector base url %q: %w", baseURL, err)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return readonlyHTTPClient{baseURL: baseURL, apiKey: cfg.APIKey, client: client, timeout: timeout}, nil
}

func (c readonlyHTTPClient) post(ctx context.Context, path string, params any, out any) error {
	meta := RequestMetaFromContext(ctx)
	payload := readonlyRequest{
		RequestID:    meta.RequestID,
		CaseID:       meta.CaseID,
		AgentID:      meta.AgentID,
		CallerUserID: meta.CallerUserID,
		ToolName:     meta.ToolName,
		TimeoutMS:    c.timeout.Milliseconds(),
		Params:       params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", meta.RequestID)
	req.Header.Set("X-Case-Id", meta.CaseID)
	req.Header.Set("X-Agent-Id", meta.AgentID)
	req.Header.Set("X-Caller-User-Id", meta.CallerUserID)
	req.Header.Set("X-Tool-Name", meta.ToolName)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp readonlyErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error == "" {
			errResp.Error = resp.Status
		}
		if errResp.Code != "" {
			return fmt.Errorf("readonly api %s failed: %s (%s)", path, errResp.Error, errResp.Code)
		}
		return fmt.Errorf("readonly api %s failed: %s", path, errResp.Error)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}
