package connectors

import (
	"context"
	"time"
)

type Candle struct {
	OpenTime time.Time `json:"open_time"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   float64   `json:"volume"`
}

type KlineQuery struct {
	Symbol     string
	Interval   string
	StartTime  time.Time
	EndTime    time.Time
	Exchange   string
	TimeBucket time.Time
}

type InternalKlineResult struct {
	Candles       []Candle  `json:"candles"`
	DataUpdatedAt time.Time `json:"data_updated_at"`
	Source        string    `json:"source"`
	QueriedAt     time.Time `json:"queried_at"`
}

type KlineCompareResult struct {
	InternalCandlesSummary string           `json:"internal_candles_summary"`
	ExternalCandlesSummary string           `json:"external_candles_summary"`
	MaxPriceDiffRatio      float64          `json:"max_price_diff_ratio"`
	MaxVolumeDiffRatio     float64          `json:"max_volume_diff_ratio"`
	AbnormalPoints         []map[string]any `json:"abnormal_points"`
	ConsistencyNote        string           `json:"consistency_note"`
}

type CacheStatus struct {
	CacheKey      string    `json:"cache_key"`
	Exists        bool      `json:"exists"`
	GeneratedAt   time.Time `json:"generated_at"`
	TTLSeconds    int       `json:"ttl"`
	Version       string    `json:"version"`
	DataUpdatedAt time.Time `json:"data_updated_at"`
}

type MarketSourceStatus struct {
	SourceName      string           `json:"source_name"`
	DelayEvents     []map[string]any `json:"delay_events"`
	ReconnectEvents []map[string]any `json:"reconnect_events"`
	DataGapEvents   []map[string]any `json:"data_gap_events"`
}

type AssetQuery struct {
	UserID      string
	AccountID   string
	AssetSymbol string
	StartTime   time.Time
	EndTime     time.Time
	AtTime      time.Time
	EventTypes  []string
}

type AssetSnapshot struct {
	AvailableBalance string    `json:"available_balance"`
	FrozenBalance    string    `json:"frozen_balance"`
	TotalBalance     string    `json:"total_balance"`
	UpdatedAt        time.Time `json:"updated_at"`
	Source           string    `json:"source"`
	Version          string    `json:"version"`
}

type AssetEvent struct {
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	Delta        string    `json:"delta"`
	BalanceAfter string    `json:"balance_after"`
	OccurredAt   time.Time `json:"occurred_at"`
	ReferenceID  string    `json:"reference_id"`
}

type AssetEventsResult struct {
	Events        []AssetEvent `json:"events"`
	BalanceBefore string       `json:"balance_before"`
	BalanceAfter  string       `json:"balance_after"`
	DataUpdatedAt time.Time    `json:"data_updated_at"`
}

type UserErrorsResult struct {
	ServiceNames []string         `json:"service_names"`
	Errors       []map[string]any `json:"errors"`
}

type LogQuery struct {
	ServiceName string
	StartTime   time.Time
	EndTime     time.Time
	Level       string
	Keyword     string
	TraceID     string
	Limit       int
}

type LogSearchResult struct {
	ServiceName string           `json:"service_name"`
	Total       int              `json:"total"`
	Samples     []map[string]any `json:"samples"`
}

type DeploymentResult struct {
	ServiceName string           `json:"service_name"`
	Items       []map[string]any `json:"items"`
}

type SimilarCaseResult struct {
	Items []map[string]any `json:"items"`
}

type KlineConnector interface {
	InternalKline(ctx context.Context, q KlineQuery) (InternalKlineResult, error)
	ExternalKlineCompare(ctx context.Context, q KlineQuery) (KlineCompareResult, error)
	KlineCacheStatus(ctx context.Context, q KlineQuery) (CacheStatus, error)
	MarketSourceStatus(ctx context.Context, q KlineQuery) (MarketSourceStatus, error)
}

type AssetConnector interface {
	AssetSnapshot(ctx context.Context, q AssetQuery) (AssetSnapshot, error)
	AssetEvents(ctx context.Context, q AssetQuery) (AssetEventsResult, error)
	UserRecentErrors(ctx context.Context, q AssetQuery) (UserErrorsResult, error)
}

type OpsConnector interface {
	SearchLogs(ctx context.Context, q LogQuery) (LogSearchResult, error)
	RecentDeployments(ctx context.Context, serviceName string, startTime time.Time, endTime time.Time) (DeploymentResult, error)
	SimilarCases(ctx context.Context, issueDomain string, issueType string, text string, entities map[string]any, limit int) (SimilarCaseResult, error)
}
