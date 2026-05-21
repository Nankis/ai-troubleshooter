package llm

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
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
		add("user_id", secondGroup(text, `(?i)\b(user_id|uid|用户|用户id)[:：\s]*([A-Za-z0-9_\-]+)`))
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

	lower := normalize(text)
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
	case containsAny(text, "延迟"):
		return "延迟"
	case containsAny(text, "成交量"):
		return "成交量不一致"
	case containsAny(text, "最高", "最低", "high", "low"):
		return "最高最低不一致"
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
			return strings.ReplaceAll(match, " ", "T") + "+08:00"
		}
	}
	return ""
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
