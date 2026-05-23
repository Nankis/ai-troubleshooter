package policy

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const DefaultAgentID = "business-troubleshooter-v1"

type Engine interface {
	Authorize(ctx context.Context, req Request) (Decision, error)
}

type Request struct {
	CaseID              string         `json:"case_id"`
	AgentID             string         `json:"agent_id"`
	CallerUserID        string         `json:"caller_user_id"`
	LarkUserID          string         `json:"lark_user_id"`
	ChatID              string         `json:"chat_id"`
	ToolName            string         `json:"tool_name"`
	RequiredScope       string         `json:"required_scope"`
	Arguments           map[string]any `json:"arguments"`
	RequestedAt         time.Time      `json:"requested_at"`
	MaxLimit            int            `json:"max_limit"`
	MaxTimeRangeMinutes int            `json:"max_time_range_minutes"`
}

type Decision struct {
	Allowed             bool   `json:"allowed"`
	MaskingLevel        string `json:"masking_level"`
	MaxLimit            int    `json:"max_limit"`
	MaxTimeRangeMinutes int    `json:"max_time_range_minutes"`
	Reason              string `json:"reason,omitempty"`
}

type Agent struct {
	AgentID           string
	AllowedScopes     map[string]bool
	AllowedTools      map[string]bool
	AllowedLarkGroups map[string]bool
	Status            string
	RateLimitQPS      int
}

type StaticEngine struct {
	agents                     map[string]Agent
	defaultMaxLimit            int
	defaultMaxTimeRangeMinutes int
}

func NewStaticEngine(agents []Agent) *StaticEngine {
	index := make(map[string]Agent, len(agents))
	for _, agent := range agents {
		index[agent.AgentID] = agent
	}
	return &StaticEngine{
		agents:                     index,
		defaultMaxLimit:            100,
		defaultMaxTimeRangeMinutes: 30,
	}
}

func (e *StaticEngine) Authorize(ctx context.Context, req Request) (Decision, error) {
	_ = ctx
	agent, ok := e.agents[req.AgentID]
	if !ok {
		return deny("unknown agent %q", req.AgentID), nil
	}
	if agent.Status != "" && agent.Status != "enabled" {
		return deny("agent disabled"), nil
	}
	if req.RequiredScope == "" {
		return deny("tool has no required scope"), nil
	}
	if !agent.AllowedScopes[req.RequiredScope] {
		return deny("scope %q is not allowed", req.RequiredScope), nil
	}
	if len(agent.AllowedTools) > 0 && !agent.AllowedTools[req.ToolName] {
		return deny("tool %q is not allowed", req.ToolName), nil
	}
	if len(agent.AllowedLarkGroups) > 0 {
		if req.ChatID == "" {
			return deny("chat_id is required for this agent"), nil
		}
		if !agent.AllowedLarkGroups[req.ChatID] {
			return deny("chat %q is not allowed", req.ChatID), nil
		}
	}

	maxLimit := firstPositive(req.MaxLimit, e.defaultMaxLimit)
	maxRange := firstPositive(req.MaxTimeRangeMinutes, e.defaultMaxTimeRangeMinutes)
	return Decision{
		Allowed:             true,
		MaskingLevel:        "standard",
		MaxLimit:            maxLimit,
		MaxTimeRangeMinutes: maxRange,
	}, nil
}

func DefaultAgents() []Agent {
	return DefaultAgentsFor(DefaultAgentID)
}

func DefaultAgentsFor(agentID string) []Agent {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		agentID = DefaultAgentID
	}
	return []Agent{
		{
			AgentID: agentID,
			AllowedScopes: Set(
				"case:read",
				"case:write",
				"tool:list",
				"kline:read",
				"asset:read",
				"logs:read_summary",
				"deploy:read",
				"cache:read",
				"similar_case:read",
				"health_food:user:read",
				"health_food:ai_quota:read",
				"health_food:meal:read",
				"health_food:recommendation:read",
			),
			AllowedTools: Set(
				"search_logs_by_service",
				"get_recent_deployments",
				"get_similar_cases",
				"get_internal_kline",
				"get_external_kline_compare",
				"get_kline_cache_status",
				"get_market_source_status",
				"get_asset_snapshot",
				"get_asset_events",
				"get_user_recent_errors",
				"get_health_food_user_profile",
				"get_health_food_ai_quota",
				"get_health_food_meal_records",
				"get_health_food_recommendation_status",
			),
			Status:       "enabled",
			RateLimitQPS: 5,
		},
	}
}

func Set(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func deny(format string, args ...any) Decision {
	return Decision{Allowed: false, MaskingLevel: "standard", Reason: fmt.Sprintf(format, args...)}
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
