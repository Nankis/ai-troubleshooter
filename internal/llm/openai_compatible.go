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

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
)

type OpenAICompatibleClient struct {
	provider          string
	baseURL           string
	apiKey            string
	model             string
	allowRuleFallback bool
	client            *http.Client
}

type OpenAICompatibleOptions struct {
	Provider          string
	BaseURL           string
	APIKey            string
	Model             string
	Timeout           time.Duration
	AllowRuleFallback bool
}

func NewOpenAICompatibleClient(options OpenAICompatibleOptions) *OpenAICompatibleClient {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OpenAICompatibleClient{
		provider:          fallback(options.Provider, "openai_compatible"),
		baseURL:           normalizeChatCompletionsURL(options.BaseURL),
		apiKey:            strings.TrimSpace(options.APIKey),
		model:             fallback(options.Model, "gpt-4.1-mini"),
		allowRuleFallback: options.AllowRuleFallback,
		client:            &http.Client{Timeout: timeout},
	}
}

func (c *OpenAICompatibleClient) ClassifyIssue(ctx context.Context, input CaseInput) (IssueClassification, error) {
	var raw map[string]any
	err := c.completeJSON(ctx, "请将工单分类为业务域和问题类型，只输出 JSON。必须使用这个 schema：{\"issue_domain\":\"health_food\",\"issue_type\":\"每日推荐缺失\",\"confidence\":0.9}。业务域 issue_domain 只能是 kline、asset、health_food 或空字符串。health-food、饮食、餐食、每日推荐、token 配额都属于 health_food。", map[string]any{
		"case": input.Case,
	}, &raw)
	out := IssueClassification{
		IssueDomain: normalizeIssueDomain(firstStringValue(raw, "issue_domain", "issueDomain", "domain", "business_domain", "业务域")),
		IssueType:   firstStringValue(raw, "issue_type", "issueType", "type", "category", "问题类型", "异常类型"),
		Confidence:  firstFloatValue(raw, "confidence", "score", "置信度"),
	}
	if out.Confidence <= 0 && strings.TrimSpace(out.IssueDomain) != "" {
		out.Confidence = 0.7
	}
	if err != nil || strings.TrimSpace(out.IssueDomain) == "" || out.Confidence <= 0 {
		if !c.allowRuleFallback {
			if err != nil {
				return IssueClassification{}, err
			}
			return IssueClassification{}, fmt.Errorf("llm returned invalid classification: domain=%q confidence=%f raw=%s", out.IssueDomain, out.Confidence, compactJSON(raw))
		}
		ruleResult, ruleErr := NewRuleBasedClient().ClassifyIssue(ctx, input)
		if ruleErr != nil {
			return out, err
		}
		if strings.TrimSpace(out.IssueType) != "" {
			ruleResult.IssueType = out.IssueType
		}
		return ruleResult, nil
	}
	return out, nil
}

func (c *OpenAICompatibleClient) ExtractEntities(ctx context.Context, input CaseInput) (ExtractedEntities, error) {
	var raw map[string]any
	err := c.completeJSON(ctx, "请抽取排障必要字段，只输出 JSON。必须使用这个 schema：{\"entities\":[{\"entity_type\":\"uid\",\"entity_value\":\"123456\",\"confidence\":0.9}]}。字段使用 entity_type/entity_value/confidence。常用字段：symbol、interval、abnormal_time、abnormal_date、issue_type、compare_exchange、user_id、uid、account_id、asset_symbol、service_name。", map[string]any{
		"case": input.Case,
	}, &raw)
	entities := []caseflow.Entity{}
	for _, item := range entityMaps(raw) {
		entityType := firstStringValue(item, "entity_type", "entityType", "type", "name", "field", "字段", "实体类型")
		entityValue := firstStringValue(item, "entity_value", "entityValue", "value", "text", "字段值", "实体值")
		if strings.TrimSpace(entityType) == "" || strings.TrimSpace(entityValue) == "" {
			continue
		}
		conf := firstFloatValue(item, "confidence", "score", "置信度")
		if conf <= 0 {
			conf = 0.7
		}
		entities = append(entities, caseflow.Entity{Type: entityType, Value: entityValue, Source: "llm", Confidence: &conf})
	}
	if err != nil || len(entities) == 0 {
		if !c.allowRuleFallback {
			if err != nil {
				return ExtractedEntities{}, err
			}
			return ExtractedEntities{}, fmt.Errorf("llm returned no entities raw=%s", compactJSON(raw))
		}
		return NewRuleBasedClient().ExtractEntities(ctx, input)
	}
	return ExtractedEntities{Entities: entities}, nil
}

func (c *OpenAICompatibleClient) DecideNextAction(ctx context.Context, state caseflow.Case, entities map[string]string, tools []string) (NextAction, error) {
	if len(tools) == 0 {
		tools = defaultToolNames(state.IssueDomain)
	}
	var raw map[string]any
	err := c.completeJSON(ctx, "请基于业务域、问题类型和实体选择只读排障工具，只输出 JSON。必须使用这个 schema：{\"tool_names\":[\"get_health_food_user_profile\"],\"reason\":\"先核对用户资料\"}。tool_names 只能从 available_tools 中选择。", map[string]any{
		"case":            state,
		"entities":        entities,
		"available_tools": tools,
	}, &raw)
	out := NextAction{
		ToolNames: firstStringSliceValue(raw, "tool_names", "toolNames", "tools", "selected_tools", "selectedTools", "工具列表"),
		Reason:    firstStringValue(raw, "reason", "rationale", "理由"),
	}
	if len(out.ToolNames) == 0 {
		if toolName := firstStringValue(raw, "tool_name", "toolName", "selected_tool", "工具"); toolName != "" {
			out.ToolNames = []string{toolName}
		}
	}
	out.ToolNames = filterAllowedTools(out.ToolNames, tools)
	if err != nil || len(out.ToolNames) == 0 {
		if !c.allowRuleFallback {
			if err != nil {
				return NextAction{}, err
			}
			return NextAction{}, fmt.Errorf("llm returned no valid tool_names raw=%s", compactJSON(raw))
		}
		return NewRuleBasedClient().DecideNextAction(ctx, state, entities, tools)
	}
	return out, nil
}

func defaultToolNames(issueDomain string) []string {
	switch issueDomain {
	case caseflow.DomainKline:
		return []string{"get_internal_kline", "get_external_kline_compare", "get_kline_cache_status", "get_market_source_status", "get_similar_cases"}
	case caseflow.DomainAsset:
		return []string{"get_asset_snapshot", "get_asset_events", "get_user_recent_errors", "get_similar_cases"}
	case caseflow.DomainHealthFood:
		return []string{"get_health_food_user_profile", "get_health_food_meal_records", "get_health_food_recommendation_status", "get_health_food_ai_quota", "search_logs_by_service", "get_similar_cases"}
	default:
		return []string{"search_logs_by_service", "get_recent_deployments", "get_similar_cases"}
	}
}

func (c *OpenAICompatibleClient) SummarizeFindings(ctx context.Context, state caseflow.Case, observations []ToolObservation) (InvestigationReport, error) {
	var raw map[string]any
	err := c.completeJSON(ctx, "请基于有限工具证据生成排障摘要，只输出 JSON。必须使用这个 schema：{\"summary\":\"结论和证据摘要\",\"confidence\":0.8}。不要编造证据；证据不足时明确说明需要人工确认。", map[string]any{
		"case":         state,
		"observations": observations,
	}, &raw)
	out := InvestigationReport{
		Summary:    firstStringValue(raw, "summary", "answer", "conclusion", "结论", "摘要"),
		Confidence: firstFloatValue(raw, "confidence", "score", "置信度"),
	}
	if out.Confidence <= 0 && strings.TrimSpace(out.Summary) != "" {
		out.Confidence = 0.7
	}
	if err != nil || strings.TrimSpace(out.Summary) == "" {
		if !c.allowRuleFallback {
			if err != nil {
				return InvestigationReport{}, err
			}
			return InvestigationReport{}, fmt.Errorf("llm returned empty summary raw=%s", compactJSON(raw))
		}
		return NewRuleBasedClient().SummarizeFindings(ctx, state, observations)
	}
	return out, nil
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
		"model":           c.model,
		"temperature":     0.1,
		"response_format": map[string]string{"type": "json_object"},
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

func normalizeIssueDomain(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch {
	case normalized == caseflow.DomainHealthFood || strings.Contains(normalized, "health_food") || strings.Contains(normalized, "health") || strings.Contains(normalized, "food") || strings.Contains(value, "饮食") || strings.Contains(value, "推荐"):
		return caseflow.DomainHealthFood
	case normalized == caseflow.DomainKline || strings.Contains(normalized, "kline") || strings.Contains(normalized, "market") || strings.Contains(value, "K线") || strings.Contains(value, "行情"):
		return caseflow.DomainKline
	case normalized == caseflow.DomainAsset || strings.Contains(normalized, "asset") || strings.Contains(value, "资产"):
		return caseflow.DomainAsset
	default:
		return normalized
	}
}

func entityMaps(raw map[string]any) []map[string]any {
	values, ok := firstArrayValue(raw, "entities", "entity_list", "fields", "实体")
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}

func firstStringValue(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			if text := stringValue(value); text != "" {
				return text
			}
		}
	}
	for _, value := range raw {
		if nested, ok := value.(map[string]any); ok {
			if text := firstStringValue(nested, keys...); text != "" {
				return text
			}
		}
	}
	return ""
}

func firstFloatValue(raw map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			if score, ok := floatValue(value); ok {
				return score
			}
		}
	}
	for _, value := range raw {
		if nested, ok := value.(map[string]any); ok {
			if score := firstFloatValue(nested, keys...); score > 0 {
				return score
			}
		}
	}
	return 0
}

func firstStringSliceValue(raw map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if out := stringSliceValue(value); len(out) > 0 {
			return out
		}
	}
	for _, value := range raw {
		if nested, ok := value.(map[string]any); ok {
			if out := firstStringSliceValue(nested, keys...); len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

func firstArrayValue(raw map[string]any, keys ...string) ([]any, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if values, ok := value.([]any); ok {
			return values, true
		}
	}
	for _, value := range raw {
		if nested, ok := value.(map[string]any); ok {
			if values, ok := firstArrayValue(nested, keys...); ok {
				return values, true
			}
		}
	}
	return nil, false
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	case bool:
		return fmt.Sprintf("%t", typed)
	default:
		return ""
	}
}

func floatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%f", &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := stringValue(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return typed
	case string:
		parts := strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == '，' || r == '\n'
		})
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	default:
		return nil
	}
}

func filterAllowedTools(toolNames []string, allowed []string) []string {
	allowedSet := map[string]bool{}
	for _, name := range allowed {
		allowedSet[name] = true
	}
	out := make([]string, 0, len(toolNames))
	seen := map[string]bool{}
	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] || !allowedSet[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func compactJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	if len(raw) > 600 {
		return string(raw[:600]) + "...truncated"
	}
	return string(raw)
}
