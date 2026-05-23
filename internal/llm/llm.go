package llm

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
)

type CaseInput struct {
	Case     caseflow.Case
	Messages []caseflow.Message
}

type IssueClassification struct {
	IssueDomain string
	IssueType   string
	Confidence  float64
}

type ExtractedEntities struct {
	Entities []caseflow.Entity
}

type NextAction struct {
	ToolNames []string
	Reason    string
}

type ToolObservation struct {
	ToolName string
	Summary  string
	Status   string
}

type InvestigationReport struct {
	Summary    string
	Confidence float64
}

type LLMClient interface {
	ClassifyIssue(ctx context.Context, input CaseInput) (IssueClassification, error)
	ExtractEntities(ctx context.Context, input CaseInput) (ExtractedEntities, error)
	DecideNextAction(ctx context.Context, state caseflow.Case, entities map[string]string, tools []string) (NextAction, error)
	SummarizeFindings(ctx context.Context, state caseflow.Case, observations []ToolObservation) (InvestigationReport, error)
}

type RuleBasedClient struct{}

func NewRuleBasedClient() RuleBasedClient {
	return RuleBasedClient{}
}

func (RuleBasedClient) ClassifyIssue(ctx context.Context, input CaseInput) (IssueClassification, error) {
	_ = ctx
	text := normalize(input.Case.OriginalText + "\n" + input.Case.OCRText)
	switch {
	case isHealthFoodText(text):
		return IssueClassification{IssueDomain: caseflow.DomainHealthFood, IssueType: classifyHealthFoodType(text), Confidence: 0.82}, nil
	case containsAny(text, "余额", "资产", "冻结", "充值", "提现", "划转", "balance"):
		return IssueClassification{IssueDomain: caseflow.DomainAsset, IssueType: classifyAssetType(text), Confidence: 0.82}, nil
	case containsAny(text, "k线", "kline", "行情", "成交量", "最高价", "最低价", "价格不一致", "不显示", "延迟"):
		return IssueClassification{IssueDomain: caseflow.DomainKline, IssueType: classifyKlineType(text), Confidence: 0.82}, nil
	default:
		return IssueClassification{IssueDomain: "", IssueType: "", Confidence: 0.2}, nil
	}
}

func (RuleBasedClient) ExtractEntities(ctx context.Context, input CaseInput) (ExtractedEntities, error) {
	_ = ctx
	text := input.Case.OriginalText + "\n" + input.Case.OCRText
	conf := 0.78
	entities := []caseflow.Entity{}
	add := func(typ, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			entities = append(entities, caseflow.Entity{Type: typ, Value: value, Source: "rules", Confidence: &conf})
		}
	}

	if match := firstMatch(text, `\b([A-Z]{2,12}USDT)\b`); match != "" {
		add("symbol", strings.ToUpper(match))
	}
	if match := firstMatch(strings.ToLower(text), `\b(1m|3m|5m|15m|30m|1h|4h|1d)\b`); match != "" {
		add("interval", match)
	}
	if match := firstMatch(text, `(?i)\b(user_id|uid|用户|用户id)[:：\s]*([A-Za-z0-9_\-]+)`); match != "" {
		value := secondGroup(text, `(?i)\b(user_id|uid|用户|用户id)[:：\s]*([A-Za-z0-9_\-]+)`)
		add("user_id", value)
		add("uid", value)
	} else if value := firstMatch(text, `(?i)\buid([0-9][A-Za-z0-9_\-]*)\b`); value != "" {
		add("user_id", value)
		add("uid", value)
	}
	if match := firstMatch(text, `(?i)\b(account_id|账户|账户id)[:：\s]*([A-Za-z0-9_\-]+)`); match != "" {
		add("account_id", secondGroup(text, `(?i)\b(account_id|账户|账户id)[:：\s]*([A-Za-z0-9_\-]+)`))
	}
	if match := firstMatch(text, `\b(USDT|BTC|ETH|BNB|SOL|USDC)\b`); match != "" {
		add("asset_symbol", strings.ToUpper(match))
	}
	if match := firstMatch(text, `(?i)\b(Binance|OKX|Bybit)\b`); match != "" {
		add("compare_exchange", strings.ToLower(match))
	}
	if match := firstTime(text); match != "" {
		add("abnormal_time", match)
	}
	if match := firstMatch(text, `(?i)\b(trace_id|trace|链路id|请求id)[:：\s]*([A-Za-z0-9_\-]+)`); match != "" {
		add("trace_id", secondGroup(text, `(?i)\b(trace_id|trace|链路id|请求id)[:：\s]*([A-Za-z0-9_\-]+)`))
	}

	lower := normalize(text)
	if issueType := classifyHealthFoodType(lower); issueType != "" && isHealthFoodText(lower) {
		add("issue_type", issueType)
		add("service_name", "health-food")
	}
	if issueType := classifyKlineType(lower); issueType != "" && containsAny(lower, "k线", "kline", "行情", "成交量", "最高价", "最低价", "价格") {
		add("issue_type", issueType)
	}
	if issueType := classifyAssetType(lower); issueType != "" && containsAny(lower, "余额", "资产", "冻结", "充值", "提现", "划转", "balance") {
		add("issue_type", issueType)
	}
	return ExtractedEntities{Entities: entities}, nil
}

func (RuleBasedClient) DecideNextAction(ctx context.Context, state caseflow.Case, entities map[string]string, tools []string) (NextAction, error) {
	_ = ctx
	_ = entities
	_ = tools
	switch state.IssueDomain {
	case caseflow.DomainHealthFood:
		switch state.IssueType {
		case "AI配额异常", "AI对话失败":
			return NextAction{
				ToolNames: []string{
					"get_health_food_user_profile",
					"get_health_food_ai_quota",
					"search_logs_by_service",
					"get_similar_cases",
				},
				Reason: "health-food AI问题需要核对用户资料、会员配额和服务错误日志。",
			}, nil
		case "每日推荐缺失", "周报生成异常":
			return NextAction{
				ToolNames: []string{
					"get_health_food_user_profile",
					"get_health_food_meal_records",
					"get_health_food_recommendation_status",
					"search_logs_by_service",
					"get_similar_cases",
				},
				Reason: "health-food 推荐问题需要核对餐食输入、推荐任务状态和服务日志。",
			}, nil
		default:
			return NextAction{
				ToolNames: []string{
					"get_health_food_user_profile",
					"search_logs_by_service",
					"get_similar_cases",
				},
				Reason: "health-food 通用问题先核对用户上下文、日志和历史相似 case。",
			}, nil
		}
	case caseflow.DomainKline:
		return NextAction{
			ToolNames: []string{
				"get_internal_kline",
				"get_external_kline_compare",
				"get_kline_cache_status",
				"get_market_source_status",
				"get_similar_cases",
			},
			Reason: "K线问题需要对比内部 K线、外部交易所、缓存和行情源状态。",
		}, nil
	case caseflow.DomainAsset:
		return NextAction{
			ToolNames: []string{
				"get_asset_snapshot",
				"get_asset_events",
				"get_user_recent_errors",
				"get_similar_cases",
			},
			Reason: "资产问题需要核对快照、事件流、用户相关错误和历史相似 case。",
		}, nil
	default:
		return NextAction{}, nil
	}
}

func (RuleBasedClient) SummarizeFindings(ctx context.Context, state caseflow.Case, observations []ToolObservation) (InvestigationReport, error) {
	_ = ctx
	lines := []string{
		"初步排查结论：",
		"",
		"问题类型：" + fallback(state.IssueType, "待确认"),
		"涉及领域：" + fallback(state.IssueDomain, "待确认"),
		"",
		"已查证据：",
	}
	success := 0
	for _, observation := range observations {
		if observation.Status == "success" {
			success++
		}
		lines = append(lines, "- "+observation.ToolName+"："+fallback(observation.Summary, "工具返回成功"))
	}
	if len(observations) == 0 {
		lines = append(lines, "- 尚无成功工具结果，不能给出确定结论。")
	}
	lines = append(lines, "", "建议下一步：请业务 owner 根据上述证据确认最终根因，并回填 root cause。")
	confidence := 0.45
	if success >= 3 {
		confidence = 0.72
	}
	return InvestigationReport{Summary: strings.Join(lines, "\n"), Confidence: confidence}, nil
}

func classifyKlineType(text string) string {
	switch {
	case containsAny(text, "不显示"):
		return "不显示"
	case containsAny(text, "成交量"):
		return "成交量不一致"
	case containsAny(text, "最高", "最低", "high", "low"):
		return "最高最低不一致"
	case containsAny(text, "延迟"):
		return "延迟"
	case containsAny(text, "不一致", "不对", "价格"):
		return "价格不一致"
	default:
		return ""
	}
}

func classifyAssetType(text string) string {
	switch {
	case containsAny(text, "冻结"):
		return "冻结异常"
	case containsAny(text, "展示"):
		return "展示不一致"
	case containsAny(text, "流水", "缺失"):
		return "流水缺失"
	case containsAny(text, "少", "减少", "balance"):
		return "余额减少"
	default:
		return ""
	}
}

func isHealthFoodText(text string) bool {
	return containsAny(text,
		"health-food", "food-health", "健康饮食", "饮食", "餐食", "食物", "营养", "每日推荐", "今日推荐", "周报",
		"recommendation", "daily recommend", "today-recommend-food", "meal", "nutrition", "weekly report",
		"ai对话", "ai 对话", "qianwen", "千问", "token账户", "token 账户", "token消耗", "token 消耗",
		"token数量", "token 数量", "token用量", "token 用量", "token count", "token usage", "quota", "配额",
	)
}

func classifyHealthFoodType(text string) string {
	switch {
	case containsAny(text, "每日推荐", "今日推荐", "推荐缺失", "没有推荐", "不出推荐", "recommendation missing", "missing recommendation", "daily recommend", "today-recommend-food"):
		return "每日推荐缺失"
	case containsAny(text, "周报", "weekly", "weekly report"):
		return "周报生成异常"
	case containsAny(text, "配额", "token账户", "token 账户", "token消耗", "token 消耗", "token数量", "token 数量", "token用量", "token 用量", "token count", "token usage", "次数", "余额不足", "quota", "消耗", "用量", "扣减", "扣除"):
		return "AI配额异常"
	case containsAny(text, "ai对话", "ai 对话", "qianwen", "千问", "模型", "识别失败", "vision failed", "model failed"):
		return "AI对话失败"
	case containsAny(text, "餐食", "食物", "饮食记录", "meal", "food record"):
		return "餐食记录异常"
	default:
		return "health-food业务异常"
	}
}

func firstTime(text string) string {
	patterns := []string{
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+\-]\d{2}:\d{2})`,
		`\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}`,
		`\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}`,
	}
	for _, pattern := range patterns {
		if match := firstMatch(text, pattern); match != "" {
			if t, err := time.Parse(time.RFC3339, match); err == nil {
				return t.Format(time.RFC3339)
			}
			return normalizeLocalTime(match)
		}
	}
	return ""
}

func normalizeLocalTime(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), " ", "T")
	if value == "" {
		return ""
	}
	timePart := value
	if parts := strings.SplitN(value, "T", 2); len(parts) == 2 {
		timePart = parts[1]
	}
	if strings.Count(timePart, ":") == 1 {
		value += ":00"
	}
	return value + "+08:00"
}

func firstMatch(text string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if len(match) == 0 {
		return ""
	}
	if len(match) > 1 {
		return match[1]
	}
	return match[0]
}

func secondGroup(text string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if len(match) > 2 {
		return match[2]
	}
	return ""
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}
