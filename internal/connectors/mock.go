package connectors

import (
	"context"
	"fmt"
	"time"
)

type MockKlineConnector struct{}

func (MockKlineConnector) InternalKline(ctx context.Context, q KlineQuery) (InternalKlineResult, error) {
	_ = ctx
	candles := mockCandles(q.StartTime, 3, 65000)
	now := time.Now()
	return InternalKlineResult{
		Candles:       candles,
		DataUpdatedAt: now.Add(-8 * time.Second),
		Source:        "market-service/mock",
		QueriedAt:     now,
	}, nil
}

func (MockKlineConnector) ExternalKlineCompare(ctx context.Context, q KlineQuery) (KlineCompareResult, error) {
	_ = ctx
	at := q.StartTime.Add(3 * time.Minute)
	return KlineCompareResult{
		InternalCandlesSummary: "3 candles, latest close 65012.40",
		ExternalCandlesSummary: fmt.Sprintf("3 candles from %s, latest close 65129.42", fallback(q.Exchange, "binance")),
		MaxPriceDiffRatio:      0.0018,
		MaxVolumeDiffRatio:     0.012,
		AbnormalPoints: []map[string]any{
			{
				"time":            at.Format(time.RFC3339),
				"field":           "high",
				"internal_value":  65120.12,
				"external_value":  65237.33,
				"diff_ratio":      0.0018,
				"evidence_source": "mock",
			},
		},
		ConsistencyNote: "mock shows a small high-price deviation around the reported minute",
	}, nil
}

func (MockKlineConnector) KlineCacheStatus(ctx context.Context, q KlineQuery) (CacheStatus, error) {
	_ = ctx
	now := time.Now()
	return CacheStatus{
		CacheKey:      fmt.Sprintf("kline:%s:%s:%s", q.Symbol, q.Interval, q.TimeBucket.Format("200601021504")),
		Exists:        true,
		GeneratedAt:   now.Add(-4 * time.Minute),
		TTLSeconds:    180,
		Version:       "mock-v1",
		DataUpdatedAt: now.Add(-3 * time.Minute),
	}, nil
}

func (MockKlineConnector) MarketSourceStatus(ctx context.Context, q KlineQuery) (MarketSourceStatus, error) {
	_ = ctx
	return MarketSourceStatus{
		SourceName: "market-source/mock",
		DelayEvents: []map[string]any{
			{
				"start_time": q.StartTime.Add(2 * time.Minute).Format(time.RFC3339),
				"end_time":   q.StartTime.Add(4 * time.Minute).Format(time.RFC3339),
				"delay_ms":   1800,
			},
		},
		ReconnectEvents: []map[string]any{},
		DataGapEvents:   []map[string]any{},
	}, nil
}

type MockAssetConnector struct{}

func (MockAssetConnector) AssetSnapshot(ctx context.Context, q AssetQuery) (AssetSnapshot, error) {
	_ = ctx
	now := time.Now()
	return AssetSnapshot{
		AvailableBalance: "1024.12000000",
		FrozenBalance:    "18.50000000",
		TotalBalance:     "1042.62000000",
		UpdatedAt:        now.Add(-10 * time.Second),
		Source:           "asset-service/mock",
		Version:          "mock-v1",
	}, nil
}

func (MockAssetConnector) AssetEvents(ctx context.Context, q AssetQuery) (AssetEventsResult, error) {
	_ = ctx
	start := q.StartTime
	if start.IsZero() {
		start = time.Now().Add(-30 * time.Minute)
	}
	return AssetEventsResult{
		Events: []AssetEvent{
			{
				EventID:      "evt_mock_001",
				EventType:    "trade_freeze",
				Delta:        "-18.50000000",
				BalanceAfter: "1024.12000000",
				OccurredAt:   start.Add(5 * time.Minute),
				ReferenceID:  "order_mock_001",
			},
			{
				EventID:      "evt_mock_002",
				EventType:    "trade_unfreeze",
				Delta:        "18.50000000",
				BalanceAfter: "1042.62000000",
				OccurredAt:   start.Add(8 * time.Minute),
				ReferenceID:  "order_mock_001",
			},
		},
		BalanceBefore: "1042.62000000",
		BalanceAfter:  "1042.62000000",
		DataUpdatedAt: time.Now().Add(-12 * time.Second),
	}, nil
}

func (MockAssetConnector) UserRecentErrors(ctx context.Context, q AssetQuery) (UserErrorsResult, error) {
	_ = ctx
	return UserErrorsResult{
		ServiceNames: []string{"asset-service", "order-service"},
		Errors: []map[string]any{
			{
				"time":    time.Now().Add(-12 * time.Minute).Format(time.RFC3339),
				"service": "asset-service",
				"level":   "warn",
				"message": "mock balance view refresh lag detected",
			},
		},
	}, nil
}

type MockOpsConnector struct{}

func (MockOpsConnector) SearchLogs(ctx context.Context, q LogQuery) (LogSearchResult, error) {
	_ = ctx
	limit := q.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 3 {
		limit = 3
	}
	samples := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		samples = append(samples, map[string]any{
			"time":    q.StartTime.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
			"level":   fallback(q.Level, "warn"),
			"service": fallback(q.ServiceName, "unknown-service"),
			"message": fmt.Sprintf("mock log sample %d keyword=%s trace_id=%s", i+1, q.Keyword, q.TraceID),
		})
	}
	return LogSearchResult{
		ServiceName: q.ServiceName,
		Total:       len(samples),
		Samples:     samples,
	}, nil
}

func (MockOpsConnector) RecentDeployments(ctx context.Context, serviceName string, startTime time.Time, endTime time.Time) (DeploymentResult, error) {
	_ = ctx
	return DeploymentResult{
		ServiceName: serviceName,
		Items: []map[string]any{
			{
				"time":        startTime.Add(10 * time.Minute).Format(time.RFC3339),
				"version":     "mock-20260521.1",
				"operator":    "release-bot",
				"change_note": "mock canary config update",
			},
		},
	}, nil
}

func (MockOpsConnector) SimilarCases(ctx context.Context, issueDomain string, issueType string, text string, entities map[string]any, limit int) (SimilarCaseResult, error) {
	_ = ctx
	if limit <= 0 || limit > 5 {
		limit = 3
	}
	items := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, map[string]any{
			"case_no":      fmt.Sprintf("case_mock_%03d", i+1),
			"issue_domain": issueDomain,
			"issue_type":   issueType,
			"summary":      fmt.Sprintf("similar mock case for %q", text),
			"score":        0.92 - float64(i)*0.08,
		})
	}
	return SimilarCaseResult{Items: items}, nil
}

func mockCandles(start time.Time, count int, base float64) []Candle {
	if start.IsZero() {
		start = time.Now().Add(-time.Duration(count) * time.Minute)
	}
	out := make([]Candle, 0, count)
	for i := 0; i < count; i++ {
		open := base + float64(i)*8.2
		out = append(out, Candle{
			OpenTime: start.Add(time.Duration(i) * time.Minute),
			Open:     open,
			High:     open + 24.5,
			Low:      open - 18.1,
			Close:    open + 12.4,
			Volume:   120.5 + float64(i)*3.3,
		})
	}
	return out
}

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}
