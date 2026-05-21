package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
)

type OpenAICompatibleClient struct {
	provider string
	baseURL  string
	apiKey   string
	model    string
	client   *http.Client
}

type OpenAICompatibleOptions struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

func NewOpenAICompatibleClient(options OpenAICompatibleOptions) *OpenAICompatibleClient {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OpenAICompatibleClient{
		provider: fallback(options.Provider, "openai_compatible"),
		baseURL:  normalizeChatCompletionsURL(options.BaseURL),
		apiKey:   strings.TrimSpace(options.APIKey),
		model:    fallback(options.Model, "gpt-4.1-mini"),
		client:   &http.Client{Timeout: timeout},
	}
}

func (c *OpenAICompatibleClient) ClassifyIssue(ctx context.Context, input CaseInput) (IssueClassification, error) {
	var out struct {
		IssueDomain string  `json:"issue_domain"`
		IssueType   string  `json:"issue_type"`
		Confidence  float64 `json:"confidence"`
	}
	err := c.completeJSON(ctx, "请将工单分类为业务域和问题类型，只输出 JSON。业务域只能是 kline、asset 或空字符串。", map[string]any{
		"case": input.Case,
	}, &out)
	return IssueClassification{IssueDomain: out.IssueDomain, IssueType: out.IssueType, Confidence: out.Confidence}, err
}

func (c *OpenAICompatibleClient) ExtractEntities(ctx context.Context, input CaseInput) (ExtractedEntities, error) {
	var out struct {
		Entities []struct {
			Type       string  `json:"entity_type"`
			Value      string  `json:"entity_value"`
			Confidence float64 `json:"confidence"`
		} `json:"entities"`
	}
	err := c.completeJSON(ctx, "请抽取排障必要字段，只输出 JSON。字段使用 entity_type/entity_value/confidence。常用字段：symbol、interval、abnormal_time、issue_type、compare_exchange、user_id、account_id、asset_symbol。", map[string]any{
		"case": input.Case,
	}, &out)
	entities := []caseflow.Entity{}
	for _, item := range out.Entities {
		if strings.TrimSpace(item.Type) == "" || strings.TrimSpace(item.Value) == "" {
			continue
		}
		conf := item.Confidence
		if conf <= 0 {
			conf = 0.7
		}
		entities = append(entities, caseflow.Entity{Type: item.Type, Value: item.Value, Source: "llm", Confidence: &conf})
	}
	return ExtractedEntities{Entities: entities}, err
}

func (c *OpenAICompatibleClient) DecideNextAction(ctx context.Context, state caseflow.Case, entities map[string]string, tools []string) (NextAction, error) {
	if len(tools) == 0 {
		tools = defaultToolNames(state.IssueDomain)
	}
	var out struct {
		ToolNames []string `json:"tool_names"`
		Reason    string   `json:"reason"`
	}
	err := c.completeJSON(ctx, "请基于业务域、问题类型和实体选择只读排障工具，只输出 JSON。tool_names 只能从可用工具中选择。", map[string]any{
		"case":            state,
		"entities":        entities,
		"available_tools": tools,
	}, &out)
	return NextAction{ToolNames: out.ToolNames, Reason: out.Reason}, err
}

func defaultToolNames(issueDomain string) []string {
	switch issueDomain {
	case caseflow.DomainKline:
		return []string{"get_internal_kline", "get_external_kline_compare", "get_kline_cache_status", "get_market_source_status", "get_similar_cases"}
	case caseflow.DomainAsset:
		return []string{"get_asset_snapshot", "get_asset_events", "get_user_recent_errors", "get_similar_cases"}
	default:
		return []string{"search_logs_by_service", "get_recent_deployments", "get_similar_cases"}
	}
}

func (c *OpenAICompatibleClient) SummarizeFindings(ctx context.Context, state caseflow.Case, observations []ToolObservation) (InvestigationReport, error) {
	var out struct {
		Summary    string  `json:"summary"`
		Confidence float64 `json:"confidence"`
	}
	err := c.completeJSON(ctx, "请基于有限工具证据生成排障摘要，只输出 JSON。不要编造证据；证据不足时明确说明需要人工确认。", map[string]any{
		"case":         state,
		"observations": observations,
	}, &out)
	return InvestigationReport{Summary: out.Summary, Confidence: out.Confidence}, err
}

func (c *OpenAICompatibleClient) completeJSON(ctx context.Context, instruction string, input any, out any) error {
	if c.baseURL == "" {
		return fmt.Errorf("llm base url is required")
	}
	if c.apiKey == "" {
		return fmt.Errorf("llm api key is required")
	}
	rawInput, err := json.Marshal(input)
	if err != nil {
		return err
	}
	body := map[string]any{
		"model":       c.model,
		"temperature": 0.1,
		"messages": []map[string]any{
			{"role": "system", "content": "你是生产业务工单排障 Agent。严格输出 JSON，不要输出 Markdown。"},
			{"role": "user", "content": instruction + "\n输入：\n" + string(rawInput)},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var response struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if response.Error != nil && response.Error.Message != "" {
			return fmt.Errorf("llm api status=%d error=%s", resp.StatusCode, response.Error.Message)
		}
		return fmt.Errorf("llm api status=%d", resp.StatusCode)
	}
	if len(response.Choices) == 0 {
		return fmt.Errorf("llm api returned no choices")
	}
	content := stripJSONFence(response.Choices[0].Message.Content)
	if content == "" {
		return fmt.Errorf("llm api returned empty content")
	}
	return json.Unmarshal([]byte(content), out)
}

func normalizeChatCompletionsURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return strings.TrimRight(base, "/") + "/chat/completions"
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	return parsed.String()
}

func stripJSONFence(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	return strings.TrimSpace(value)
}
