package evolution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/masking"
)

type Service struct {
	store caseflow.Store
}

type ConfirmRootCauseInput struct {
	CaseRef                string `json:"case_ref,omitempty"`
	AIPredictedReason      string `json:"ai_predicted_reason,omitempty"`
	HumanConfirmedReason   string `json:"human_confirmed_reason"`
	RootCauseCategory      string `json:"root_cause_category"`
	OwnerService           string `json:"owner_service,omitempty"`
	OwnerTeam              string `json:"owner_team,omitempty"`
	IsCacheIssue           bool   `json:"is_cache_issue"`
	IsDataSyncIssue        bool   `json:"is_data_sync_issue"`
	IsExternalSourceIssue  bool   `json:"is_external_source_issue"`
	IsFrontendDisplayIssue bool   `json:"is_frontend_display_issue"`
	IsUserMisunderstanding bool   `json:"is_user_misunderstanding"`
	FixAction              string `json:"fix_action,omitempty"`
	PreventionAction       string `json:"prevention_action,omitempty"`
	ConfirmedBy            string `json:"confirmed_by,omitempty"`
}

type ConfirmRootCauseResult struct {
	Case          caseflow.Case                  `json:"case"`
	RootCause     caseflow.RootCause             `json:"root_cause"`
	KnowledgeItem caseflow.KnowledgeItem         `json:"knowledge_item"`
	EvolutionRun  caseflow.KnowledgeEvolutionRun `json:"evolution_run"`
}

func NewService(store caseflow.Store) *Service {
	return &Service{store: store}
}

func (s *Service) ConfirmRootCause(ctx context.Context, c *caseflow.Case, input ConfirmRootCauseInput) (ConfirmRootCauseResult, error) {
	if c == nil {
		return ConfirmRootCauseResult{}, fmt.Errorf("case is required")
	}
	if strings.TrimSpace(input.HumanConfirmedReason) == "" {
		return ConfirmRootCauseResult{}, fmt.Errorf("human_confirmed_reason is required")
	}
	if strings.TrimSpace(input.RootCauseCategory) == "" {
		return ConfirmRootCauseResult{}, fmt.Errorf("root_cause_category is required")
	}

	rootCause, err := s.store.UpsertRootCause(ctx, caseflow.RootCause{
		CaseID:                 c.ID,
		AIPredictedReason:      strings.TrimSpace(input.AIPredictedReason),
		HumanConfirmedReason:   strings.TrimSpace(input.HumanConfirmedReason),
		RootCauseCategory:      strings.TrimSpace(input.RootCauseCategory),
		OwnerService:           strings.TrimSpace(input.OwnerService),
		OwnerTeam:              strings.TrimSpace(input.OwnerTeam),
		IsCacheIssue:           input.IsCacheIssue,
		IsDataSyncIssue:        input.IsDataSyncIssue,
		IsExternalSourceIssue:  input.IsExternalSourceIssue,
		IsFrontendDisplayIssue: input.IsFrontendDisplayIssue,
		IsUserMisunderstanding: input.IsUserMisunderstanding,
		FixAction:              strings.TrimSpace(input.FixAction),
		PreventionAction:       strings.TrimSpace(input.PreventionAction),
		ConfirmedBy:            strings.TrimSpace(input.ConfirmedBy),
		ConfirmedAt:            time.Now(),
	})
	if err != nil {
		return ConfirmRootCauseResult{}, err
	}

	item, created, err := s.evolveKnowledge(ctx, *c, rootCause)
	if err != nil {
		run, _ := s.store.CreateKnowledgeEvolutionRun(ctx, caseflow.KnowledgeEvolutionRun{
			CaseID:            c.ID,
			TriggerType:       "root_cause_confirmed",
			InputSnapshotJSON: snapshotJSON(c, rootCause),
			Decision:          "failed",
			ErrorMessage:      err.Error(),
		})
		return ConfirmRootCauseResult{Case: *c, RootCause: rootCause, EvolutionRun: run}, err
	}
	run, err := s.store.CreateKnowledgeEvolutionRun(ctx, caseflow.KnowledgeEvolutionRun{
		CaseID:               c.ID,
		KnowledgeItemID:      item.ID,
		TriggerType:          "root_cause_confirmed",
		InputSnapshotJSON:    snapshotJSON(c, rootCause),
		OutputSummary:        fmt.Sprintf("knowledge item %d evolved from case %s", item.ID, c.CaseNo),
		Decision:             "upserted",
		CreatedKnowledgeItem: created,
		UpdatedKnowledgeItem: !created,
	})
	if err != nil {
		return ConfirmRootCauseResult{}, err
	}

	updatedCase := *c
	if caseflow.CanTransition(c.Status, caseflow.StatusDone) {
		if next, err := s.store.UpdateCase(ctx, c.ID, c.Version, func(current *caseflow.Case) error {
			current.Status = caseflow.StatusDone
			now := time.Now()
			current.ClosedAt = &now
			return nil
		}); err == nil {
			updatedCase = *next
		}
	}
	_, _ = s.store.AddMessage(ctx, caseflow.Message{
		CaseID:  c.ID,
		Role:    "system",
		Content: fmt.Sprintf("root cause confirmed: %s; knowledge item evolved: %d", rootCause.RootCauseCategory, item.ID),
	})
	return ConfirmRootCauseResult{
		Case:          updatedCase,
		RootCause:     rootCause,
		KnowledgeItem: item,
		EvolutionRun:  run,
	}, nil
}

func (s *Service) evolveKnowledge(ctx context.Context, c caseflow.Case, rootCause caseflow.RootCause) (caseflow.KnowledgeItem, bool, error) {
	existing, err := s.store.FindKnowledgeItem(ctx, c.IssueDomain, c.IssueType, rootCause.RootCauseCategory)
	created := false
	if err != nil {
		if err != caseflow.ErrNotFound {
			return caseflow.KnowledgeItem{}, false, err
		}
		created = true
		existing = caseflow.KnowledgeItem{
			Title:                buildTitle(c, rootCause),
			IssueDomain:          c.IssueDomain,
			IssueType:            c.IssueType,
			TypicalDescription:   masking.MaskString(c.OriginalText),
			RequiredFieldsJSON:   mustJSON(requiredFields(c.IssueDomain)),
			RecommendedStepsJSON: mustJSON(maskStrings(recommendedSteps(c.IssueDomain, rootCause))),
			CommonCausesJSON:     mustJSON([]string{commonCauseText(rootCause)}),
			UsefulToolsJSON:      mustJSON(usefulTools(c.IssueDomain)),
			SuccessCaseIDsJSON:   mustJSON([]int64{}),
			FailureCaseIDsJSON:   mustJSON([]int64{}),
			Status:               "active",
			Confidence:           0.55,
		}
	}

	successIDs := appendUniqueInt64(jsonInt64Slice(existing.SuccessCaseIDsJSON), c.ID)
	commonCauses := appendUniqueString(jsonStringSlice(existing.CommonCausesJSON), commonCauseText(rootCause))
	steps := appendUniqueString(jsonStringSlice(existing.RecommendedStepsJSON), maskStrings(recommendedSteps(c.IssueDomain, rootCause))...)
	tools := appendUniqueString(jsonStringSlice(existing.UsefulToolsJSON), usefulTools(c.IssueDomain)...)

	existing.Title = buildTitle(c, rootCause)
	existing.IssueDomain = c.IssueDomain
	existing.IssueType = c.IssueType
	if existing.TypicalDescription == "" {
		existing.TypicalDescription = masking.MaskString(c.OriginalText)
	}
	existing.RequiredFieldsJSON = mustJSON(requiredFields(c.IssueDomain))
	existing.RecommendedStepsJSON = mustJSON(steps)
	existing.CommonCausesJSON = mustJSON(commonCauses)
	existing.UsefulToolsJSON = mustJSON(tools)
	existing.SuccessCaseIDsJSON = mustJSON(successIDs)
	existing.ObservedCaseCount = len(successIDs)
	existing.LastRootCauseCategory = rootCause.RootCauseCategory
	existing.LastConfirmedReason = masking.MaskString(rootCause.HumanConfirmedReason)
	existing.LastEvolvedAt = time.Now()
	existing.Confidence = confidenceFromCases(len(successIDs))
	existing.Status = "active"
	item, err := s.store.UpsertKnowledgeItem(ctx, existing)
	return item, created, err
}

func buildTitle(c caseflow.Case, rootCause caseflow.RootCause) string {
	parts := []string{}
	if c.IssueDomain != "" {
		parts = append(parts, c.IssueDomain)
	}
	if c.IssueType != "" {
		parts = append(parts, c.IssueType)
	}
	if rootCause.RootCauseCategory != "" {
		parts = append(parts, rootCause.RootCauseCategory)
	}
	if len(parts) == 0 {
		return "业务排障经验"
	}
	return strings.Join(parts, " / ")
}

func requiredFields(domain string) []string {
	switch domain {
	case caseflow.DomainKline:
		return []string{"symbol", "interval", "abnormal_time", "issue_type", "compare_exchange"}
	case caseflow.DomainAsset:
		return []string{"user_id_or_account_id", "asset_symbol", "abnormal_time", "issue_type"}
	default:
		return []string{"issue_domain", "issue_type", "abnormal_time"}
	}
}

func usefulTools(domain string) []string {
	switch domain {
	case caseflow.DomainKline:
		return []string{"get_internal_kline", "get_external_kline_compare", "get_kline_cache_status", "get_market_source_status", "get_similar_cases"}
	case caseflow.DomainAsset:
		return []string{"get_asset_snapshot", "get_asset_events", "get_user_recent_errors", "get_similar_cases"}
	default:
		return []string{"search_logs_by_service", "get_recent_deployments", "get_similar_cases"}
	}
}

func recommendedSteps(domain string, rootCause caseflow.RootCause) []string {
	steps := []string{"先确认最小必要字段是否齐全", "每个结论必须绑定工具证据或说明证据不足"}
	switch domain {
	case caseflow.DomainKline:
		steps = append(steps, "对比内部 K线与外部交易所", "检查 K线缓存生成时间、TTL 与数据更新时间", "检查行情源延迟、重连和数据缺口事件")
	case caseflow.DomainAsset:
		steps = append(steps, "查询资产快照并区分可用/冻结/总余额", "查询资产事件流解释余额变化", "检查用户近期资产/订单服务错误摘要")
	}
	if rootCause.IsCacheIssue {
		steps = append(steps, "重点检查缓存 key、版本、生成时间和补偿任务")
	}
	if rootCause.IsDataSyncIssue {
		steps = append(steps, "重点检查数据同步延迟、消息积压和幂等补偿")
	}
	if rootCause.IsExternalSourceIssue {
		steps = append(steps, "重点检查外部源延迟、断连和对照交易所差异")
	}
	if rootCause.IsFrontendDisplayIssue {
		steps = append(steps, "补充前端展示时间、接口响应和后端记录三方对照")
	}
	if rootCause.IsUserMisunderstanding {
		steps = append(steps, "输出解释口径，区分真实数据异常和用户理解偏差")
	}
	if rootCause.PreventionAction != "" {
		steps = append(steps, "预防动作："+rootCause.PreventionAction)
	}
	return steps
}

func confidenceFromCases(count int) float64 {
	switch {
	case count >= 10:
		return 0.9
	case count >= 5:
		return 0.78
	case count >= 2:
		return 0.65
	default:
		return 0.55
	}
}

func snapshotJSON(c *caseflow.Case, rootCause caseflow.RootCause) string {
	return mustJSON(masking.MaskValue(map[string]any{
		"case":       c,
		"root_cause": rootCause,
	}))
}

func mustJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(b)
}

func commonCauseText(rootCause caseflow.RootCause) string {
	return masking.MaskString(rootCause.RootCauseCategory + ": " + rootCause.HumanConfirmedReason)
}

func maskStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, masking.MaskString(value))
	}
	return out
}

func jsonStringSlice(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func jsonInt64Slice(raw string) []int64 {
	if raw == "" {
		return nil
	}
	var out []int64
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func appendUniqueString(values []string, additions ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values)+len(additions))
	for _, value := range append(values, additions...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func appendUniqueInt64(values []int64, additions ...int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(values)+len(additions))
	for _, value := range append(values, additions...) {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
