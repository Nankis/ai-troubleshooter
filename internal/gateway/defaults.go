package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/audit"
	"github.com/Nankis/ai-troubleshooter/internal/config"
	"github.com/Nankis/ai-troubleshooter/internal/connectors"
	"github.com/Nankis/ai-troubleshooter/internal/policy"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
)

func NewDefault(timeout time.Duration) *Gateway {
	registry := tool.NewRegistry()
	RegisterDefaultTools(registry, connectors.MockKlineConnector{}, connectors.MockAssetConnector{}, connectors.MockOpsConnector{}, connectors.MockHealthFoodConnector{})
	return New(registry, policy.NewStaticEngine(policy.DefaultAgents()), audit.NewMemorySink(), timeout)
}

func NewFromConfig(cfg config.Config) (*Gateway, error) {
	return NewFromConfigWithAudit(cfg, audit.NewMemorySink())
}

func NewFromConfigWithAudit(cfg config.Config, auditSink audit.Sink) (*Gateway, error) {
	if err := cfg.ValidateForGateway(); err != nil {
		return nil, err
	}
	if auditSink == nil {
		auditSink = audit.NewMemorySink()
	}
	timeout := time.Duration(cfg.Limits.DefaultToolTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	registry := tool.NewRegistry()
	kline, asset, ops, healthFood, err := buildConnectors(cfg)
	if err != nil {
		return nil, err
	}
	RegisterDefaultTools(registry, kline, asset, ops, healthFood)
	return New(registry, policy.NewStaticEngine(policyAgentsFromConfig(cfg.Gateway.AgentID, cfg.Gateway.Agents)), auditSink, timeout).WithSecurity(SecurityConfig{
		AuthEnabled:                   cfg.Gateway.AuthEnabled,
		BearerTokens:                  cfg.Gateway.BearerTokens,
		AllowUnauthenticatedListTools: cfg.Gateway.AllowUnauthenticatedListTools,
		AgentQPS:                      cfg.Gateway.AgentQPS,
		UserQPS:                       cfg.Gateway.UserQPS,
		ToolQPS:                       cfg.Gateway.ToolQPS,
	}), nil
}

func policyAgentsFromConfig(defaultAgentID string, configs []config.GatewayAgentConfig) []policy.Agent {
	if len(configs) == 0 {
		return policy.DefaultAgentsFor(defaultAgentID)
	}
	agents := make([]policy.Agent, 0, len(configs))
	for _, item := range configs {
		agents = append(agents, policy.Agent{
			AgentID:           strings.TrimSpace(item.AgentID),
			AllowedScopes:     policy.Set(item.AllowedScopes...),
			AllowedTools:      policy.Set(item.AllowedTools...),
			AllowedLarkGroups: policy.Set(item.AllowedChatIDs...),
			Status:            firstNonEmpty(strings.TrimSpace(item.Status), "enabled"),
			RateLimitQPS:      item.RateLimitQPS,
		})
	}
	return agents
}

func buildConnectors(cfg config.Config) (connectors.KlineConnector, connectors.AssetConnector, connectors.OpsConnector, connectors.HealthFoodConnector, error) {
	if strings.EqualFold(cfg.Connectors.Mode, "http") {
		timeout := time.Duration(cfg.Connectors.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		kline, err := connectors.NewHTTPKlineConnector(connectors.HTTPConfig{
			BaseURL: cfg.Connectors.MarketBaseURL,
			APIKey:  cfg.Connectors.APIKey,
			Timeout: timeout,
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("market readonly connector: %w", err)
		}
		asset, err := connectors.NewHTTPAssetConnector(connectors.HTTPConfig{
			BaseURL: cfg.Connectors.AssetBaseURL,
			APIKey:  cfg.Connectors.APIKey,
			Timeout: timeout,
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("asset readonly connector: %w", err)
		}
		ops, err := connectors.NewHTTPOpsConnector(connectors.HTTPConfig{
			BaseURL: cfg.Connectors.OpsBaseURL,
			APIKey:  cfg.Connectors.APIKey,
			Timeout: timeout,
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("ops readonly connector: %w", err)
		}
		healthFoodBaseURL := firstNonEmpty(cfg.Connectors.HealthFoodBaseURL, cfg.Connectors.OpsBaseURL, cfg.Connectors.AssetBaseURL, cfg.Connectors.MarketBaseURL)
		healthFood, err := connectors.NewHTTPHealthFoodConnector(connectors.HTTPConfig{
			BaseURL: healthFoodBaseURL,
			APIKey:  cfg.Connectors.APIKey,
			Timeout: timeout,
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("health-food readonly connector: %w", err)
		}
		return kline, asset, ops, healthFood, nil
	}
	return connectors.MockKlineConnector{}, connectors.MockAssetConnector{}, connectors.MockOpsConnector{}, connectors.MockHealthFoodConnector{}, nil
}

func RegisterDefaultTools(reg *tool.Registry, kline connectors.KlineConnector, asset connectors.AssetConnector, ops connectors.OpsConnector, healthFood connectors.HealthFoodConnector) {
	mustRegister(reg, tool.Spec{
		Name:                "search_logs_by_service",
		Description:         "按服务、时间、关键词查询日志摘要。",
		InputSchema:         schema("service_name", "start_time", "end_time", "level", "keyword", "trace_id", "limit"),
		RequiredScope:       "logs:read_summary",
		BackendHandler:      "log_connector.search",
		MaxTimeRangeMinutes: 30,
		MaxLimit:            100,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		start, end := timeWindow(req.Arguments, 30*time.Minute)
		limit := intDefault(req.Arguments, "limit", 50)
		result, err := ops.SearchLogs(ctx, connectors.LogQuery{
			ServiceName: stringArg(req.Arguments, "service_name"),
			StartTime:   start,
			EndTime:     end,
			Level:       stringArg(req.Arguments, "level"),
			Keyword:     stringArg(req.Arguments, "keyword"),
			TraceID:     stringArg(req.Arguments, "trace_id"),
			Limit:       limit,
		})
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("found %d log samples for %s", result.Total, result.ServiceName),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_recent_deployments",
		Description:         "查询某服务在指定时间窗内的发布、配置、灰度信息。",
		InputSchema:         schema("service_name", "start_time", "end_time"),
		RequiredScope:       "deploy:read",
		BackendHandler:      "deploy_connector.recent",
		MaxTimeRangeMinutes: 120,
		MaxLimit:            50,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		start, end := timeWindow(req.Arguments, 120*time.Minute)
		serviceName := requiredString(req.Arguments, "service_name")
		if serviceName == "" {
			return tool.InvocationResponse{}, fmt.Errorf("service_name is required")
		}
		result, err := ops.RecentDeployments(ctx, serviceName, start, end)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("found %d deployment records for %s", len(result.Items), serviceName),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:           "get_similar_cases",
		Description:    "查询历史相似 case。",
		InputSchema:    schema("issue_domain", "issue_type", "text", "entities", "limit"),
		RequiredScope:  "similar_case:read",
		BackendHandler: "case_repository.similar",
		MaxLimit:       20,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		limit := intDefault(req.Arguments, "limit", 5)
		entities, _ := req.Arguments["entities"].(map[string]any)
		result, err := ops.SimilarCases(ctx, stringArg(req.Arguments, "issue_domain"), stringArg(req.Arguments, "issue_type"), stringArg(req.Arguments, "text"), entities, limit)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("found %d similar cases", len(result.Items)),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_internal_kline",
		Description:         "查询我方 K线结果。",
		InputSchema:         schema("symbol", "interval", "start_time", "end_time"),
		RequiredScope:       "kline:read",
		BackendHandler:      "market_connector.internal_kline",
		MaxTimeRangeMinutes: 30,
		MaxLimit:            500,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := klineQuery(req.Arguments)
		if q.Symbol == "" || q.Interval == "" {
			return tool.InvocationResponse{}, fmt.Errorf("symbol and interval are required")
		}
		result, err := kline.InternalKline(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("%s %s internal kline returned %d candles", q.Symbol, q.Interval, len(result.Candles)),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_external_kline_compare",
		Description:         "对比外部交易所 K线。",
		InputSchema:         schema("symbol", "interval", "start_time", "end_time", "exchange"),
		RequiredScope:       "kline:read",
		BackendHandler:      "external_exchange_connector.kline_compare",
		MaxTimeRangeMinutes: 30,
		MaxLimit:            500,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := klineQuery(req.Arguments)
		if q.Symbol == "" || q.Interval == "" {
			return tool.InvocationResponse{}, fmt.Errorf("symbol and interval are required")
		}
		result, err := kline.ExternalKlineCompare(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: result.ConsistencyNote,
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_kline_cache_status",
		Description:         "查询 K线缓存状态。",
		InputSchema:         schema("symbol", "interval", "time_bucket"),
		RequiredScope:       "cache:read",
		BackendHandler:      "redis_readonly_connector.kline_cache_status",
		MaxTimeRangeMinutes: 30,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := klineQuery(req.Arguments)
		if q.Symbol == "" || q.Interval == "" {
			return tool.InvocationResponse{}, fmt.Errorf("symbol and interval are required")
		}
		result, err := kline.KlineCacheStatus(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("cache exists=%t ttl=%d", result.Exists, result.TTLSeconds),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_market_source_status",
		Description:         "查询行情源状态。",
		InputSchema:         schema("symbol", "start_time", "end_time"),
		RequiredScope:       "kline:read",
		BackendHandler:      "market_connector.source_status",
		MaxTimeRangeMinutes: 30,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := klineQuery(req.Arguments)
		if q.Symbol == "" {
			return tool.InvocationResponse{}, fmt.Errorf("symbol is required")
		}
		result, err := kline.MarketSourceStatus(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("found %d delay events", len(result.DelayEvents)),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_asset_snapshot",
		Description:         "查询用户资产快照。",
		InputSchema:         schema("user_id", "account_id", "asset_symbol", "at_time"),
		RequiredScope:       "asset:read",
		BackendHandler:      "asset_connector.snapshot",
		MaxTimeRangeMinutes: 120,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := assetQuery(req.Arguments)
		if q.UserID == "" && q.AccountID == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or account_id is required")
		}
		if q.AssetSymbol == "" {
			return tool.InvocationResponse{}, fmt.Errorf("asset_symbol is required")
		}
		result, err := asset.AssetSnapshot(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("%s snapshot total=%s", q.AssetSymbol, result.TotalBalance),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_asset_events",
		Description:         "查询用户资产变更事件流。",
		InputSchema:         schema("user_id", "account_id", "asset_symbol", "start_time", "end_time", "event_types"),
		RequiredScope:       "asset:read",
		BackendHandler:      "asset_connector.events",
		MaxTimeRangeMinutes: 120,
		MaxLimit:            100,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := assetQuery(req.Arguments)
		if q.UserID == "" && q.AccountID == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or account_id is required")
		}
		if q.AssetSymbol == "" {
			return tool.InvocationResponse{}, fmt.Errorf("asset_symbol is required")
		}
		result, err := asset.AssetEvents(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("found %d asset events", len(result.Events)),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_user_recent_errors",
		Description:         "查询用户近期在相关服务中的错误日志摘要。",
		InputSchema:         schema("user_id", "account_id", "start_time", "end_time", "service_names"),
		RequiredScope:       "logs:read_summary",
		BackendHandler:      "log_connector.user_recent_errors",
		MaxTimeRangeMinutes: 120,
		MaxLimit:            100,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := assetQuery(req.Arguments)
		if q.UserID == "" && q.AccountID == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or account_id is required")
		}
		result, err := asset.UserRecentErrors(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("found %d user error samples", len(result.Errors)),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_health_food_user_profile",
		Description:         "查询 health-food 用户资料、会员等级和最近设备摘要。",
		InputSchema:         schema("user_id", "uid", "at_time"),
		RequiredScope:       "health_food:user:read",
		BackendHandler:      "health_food_connector.user_profile",
		MaxTimeRangeMinutes: 120,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := healthFoodQuery(req.Arguments)
		if q.EffectiveUserID() == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or uid is required")
		}
		result, err := healthFood.UserProfile(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("health-food user %s registered=%t membership_level=%d", result.UID, result.Registered, result.MembershipLevel),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_health_food_ai_quota",
		Description:         "查询 health-food 用户 AI token / 对话次数配额状态。",
		InputSchema:         schema("user_id", "uid", "at_time"),
		RequiredScope:       "health_food:ai_quota:read",
		BackendHandler:      "health_food_connector.ai_quota",
		MaxTimeRangeMinutes: 120,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := healthFoodQuery(req.Arguments)
		if q.EffectiveUserID() == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or uid is required")
		}
		result, err := healthFood.AIQuota(ctx, q)
		status := "normal"
		if result.Abnormal {
			status = "abnormal"
		}
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("health-food ai quota %s: tokens=%s daily_chat=%d/%d reason=%s", status, result.AvailableTokens, result.DailyChatCount, result.LimitChat, result.Reason),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_health_food_meal_records",
		Description:         "查询 health-food 指定用户时间窗内的餐食记录和数据指纹。",
		InputSchema:         schema("user_id", "uid", "start_time", "end_time", "limit"),
		RequiredScope:       "health_food:meal:read",
		BackendHandler:      "health_food_connector.meal_records",
		MaxTimeRangeMinutes: 24 * 60,
		MaxLimit:            100,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := healthFoodQuery(req.Arguments)
		if q.EffectiveUserID() == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or uid is required")
		}
		result, err := healthFood.MealRecords(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("health-food returned %d meal records, missing=%d fingerprint=%s", result.MealCount, len(result.MissingMealIDs), result.MealDataFingerprint),
		}, err
	})

	mustRegister(reg, tool.Spec{
		Name:                "get_health_food_recommendation_status",
		Description:         "查询 health-food 每日推荐生成状态、输入餐食和失败原因。",
		InputSchema:         schema("user_id", "uid", "recommendation_date", "start_time", "end_time"),
		RequiredScope:       "health_food:recommendation:read",
		BackendHandler:      "health_food_connector.recommendation_status",
		MaxTimeRangeMinutes: 24 * 60,
	}, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		q := healthFoodQuery(req.Arguments)
		if q.EffectiveUserID() == "" {
			return tool.InvocationResponse{}, fmt.Errorf("user_id or uid is required")
		}
		result, err := healthFood.RecommendationStatus(ctx, q)
		return tool.InvocationResponse{
			Status:  "success",
			Data:    result,
			Summary: fmt.Sprintf("health-food recommendation date=%s exists=%t job_status=%s reason=%s", result.RecommendDate, result.HasRecommendation, result.JobStatus, result.FailureReason),
		}, err
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mustRegister(reg *tool.Registry, spec tool.Spec, handler tool.HandlerFunc) {
	if err := reg.Register(spec, handler); err != nil {
		panic(err)
	}
}

func schema(fields ...string) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, field := range fields {
		properties[field] = map[string]any{"type": "string"}
		switch field {
		case "limit":
			properties[field] = map[string]any{"type": "integer"}
		case "entities":
			properties[field] = map[string]any{"type": "object"}
		}
		if field == "symbol" || field == "interval" || field == "asset_symbol" || field == "service_name" {
			required = append(required, field)
		}
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func klineQuery(args map[string]any) connectors.KlineQuery {
	start, end := timeWindow(args, 30*time.Minute)
	bucket := timeDefault(args, "time_bucket", start)
	return connectors.KlineQuery{
		Symbol:     strings.ToUpper(stringArg(args, "symbol")),
		Interval:   stringArg(args, "interval"),
		StartTime:  start,
		EndTime:    end,
		Exchange:   strings.ToLower(stringDefault(args, "exchange", "binance")),
		TimeBucket: bucket,
	}
}

func assetQuery(args map[string]any) connectors.AssetQuery {
	start, end := timeWindow(args, 120*time.Minute)
	return connectors.AssetQuery{
		UserID:      stringArg(args, "user_id"),
		AccountID:   stringArg(args, "account_id"),
		AssetSymbol: strings.ToUpper(stringArg(args, "asset_symbol")),
		StartTime:   start,
		EndTime:     end,
		AtTime:      timeDefault(args, "at_time", end),
		EventTypes:  stringSliceArg(args, "event_types"),
	}
}

func healthFoodQuery(args map[string]any) connectors.HealthFoodQuery {
	start, end := timeWindow(args, 24*time.Hour)
	uid := stringArg(args, "uid")
	userID := stringArg(args, "user_id")
	atTime := timeDefault(args, "at_time", end)
	recommendationDate := stringArg(args, "recommendation_date")
	if recommendationDate == "" && !atTime.IsZero() {
		recommendationDate = atTime.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
	}
	return connectors.HealthFoodQuery{
		UserID:             userID,
		UID:                uid,
		StartTime:          start,
		EndTime:            end,
		AtTime:             atTime,
		RecommendationDate: recommendationDate,
		TraceID:            stringArg(args, "trace_id"),
		Limit:              intDefault(args, "limit", 50),
	}
}

func timeWindow(args map[string]any, def time.Duration) (time.Time, time.Time) {
	now := time.Now()
	start := timeDefault(args, "start_time", now.Add(-def))
	end := timeDefault(args, "end_time", now)
	return start, end
}

func stringArg(args map[string]any, key string) string {
	raw, ok := args[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return fmt.Sprint(v)
	}
}

func requiredString(args map[string]any, key string) string {
	return stringArg(args, key)
}

func stringDefault(args map[string]any, key string, def string) string {
	if v := stringArg(args, key); v != "" {
		return v
	}
	return def
}

func intDefault(args map[string]any, key string, def int) int {
	if v, ok := intArg(args, key); ok {
		return v
	}
	return def
}

func timeDefault(args map[string]any, key string, def time.Time) time.Time {
	v, ok, err := timeArg(args, key)
	if err == nil && ok {
		return v
	}
	return def
}

func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		parts := strings.Split(v, ",")
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
