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

type HealthFoodQuery struct {
	UserID             string
	UID                string
	StartTime          time.Time
	EndTime            time.Time
	AtTime             time.Time
	RecommendationDate string
	TraceID            string
	Limit              int
}

func (q HealthFoodQuery) EffectiveUserID() string {
	if q.UserID != "" {
		return q.UserID
	}
	return q.UID
}

type HealthFoodUserProfile struct {
	UID               string         `json:"uid"`
	Registered        bool           `json:"registered"`
	MembershipLevel   int            `json:"membership_level"`
	HealthGoalSummary string         `json:"health_goal_summary"`
	LatestDevice      map[string]any `json:"latest_device,omitempty"`
	UpdatedAt         time.Time      `json:"updated_at"`
	Source            string         `json:"source"`
	Version           string         `json:"version"`
}

type HealthFoodAIQuota struct {
	UID             string    `json:"uid"`
	MembershipLevel int       `json:"membership_level"`
	AvailableTokens string    `json:"available_tokens"`
	DailyChatCount  int       `json:"daily_chat_count"`
	LimitChat       int       `json:"limit_chat"`
	LastResetDate   time.Time `json:"last_reset_date"`
	Abnormal        bool      `json:"abnormal"`
	Reason          string    `json:"reason"`
	DataUpdatedAt   time.Time `json:"data_updated_at"`
}

type HealthFoodMealRecords struct {
	UID                 string           `json:"uid"`
	MealCount           int              `json:"meal_count"`
	MissingMealIDs      []string         `json:"missing_meal_ids"`
	MealDataFingerprint string           `json:"meal_data_fingerprint"`
	Meals               []map[string]any `json:"meals"`
	DataUpdatedAt       time.Time        `json:"data_updated_at"`
}

type HealthFoodRecommendationStatus struct {
	UID                 string     `json:"uid"`
	RecommendDate       string     `json:"recommend_date"`
	HasRecommendation   bool       `json:"has_recommendation"`
	JobStatus           string     `json:"job_status"`
	MealCount           int        `json:"meal_count"`
	MealDataFingerprint string     `json:"meal_data_fingerprint"`
	GeneratedAt         *time.Time `json:"generated_at,omitempty"`
	FailureReason       string     `json:"failure_reason,omitempty"`
	SourceMealIDs       []string   `json:"source_meal_ids"`
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

type HealthFoodConnector interface {
	UserProfile(ctx context.Context, q HealthFoodQuery) (HealthFoodUserProfile, error)
	AIQuota(ctx context.Context, q HealthFoodQuery) (HealthFoodAIQuota, error)
	MealRecords(ctx context.Context, q HealthFoodQuery) (HealthFoodMealRecords, error)
	RecommendationStatus(ctx context.Context, q HealthFoodQuery) (HealthFoodRecommendationStatus, error)
}
