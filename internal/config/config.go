package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server      ServerConfig
	Lark        LarkConfig
	LLM         LLMConfig
	Database    DatabaseConfig
	Queue       QueueConfig
	Connectors  ConnectorConfig
	ToolGateway ToolGatewayConfig
	Limits      LimitsConfig
}

type ServerConfig struct {
	Env      string
	HTTPPort int
}

type LarkConfig struct {
	AppID             string
	AppSecret         string
	VerificationToken string
	EncryptKey        string
	AllowedChatIDs    []string
}

type LLMConfig struct {
	Provider       string
	BaseURL        string
	APIKey         string
	Model          string
	TimeoutSeconds int
	MaxConcurrency int
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type QueueConfig struct {
	Type       string
	RedisAddr  string
	StreamName string
}

type ConnectorConfig struct {
	Mode           string
	APIKey         string
	TimeoutSeconds int
	MarketBaseURL  string
	AssetBaseURL   string
	OpsBaseURL     string
}

type ToolGatewayConfig struct {
	Endpoint     string
	ClientID     string
	ClientSecret string
}

type LimitsConfig struct {
	MaxToolCallsPerCase        int
	MaxLLMCallsPerCase         int
	DefaultLogTimeRangeMinutes int
	DefaultToolTimeoutSeconds  int
	WorkerConcurrency          int
}

func LoadFromEnv() Config {
	return Config{
		Server: ServerConfig{
			Env:      env("APP_ENV", "dev"),
			HTTPPort: envInt("HTTP_PORT", 8080),
		},
		Lark: LarkConfig{
			AppID:             env("LARK_APP_ID", ""),
			AppSecret:         env("LARK_APP_SECRET", ""),
			VerificationToken: env("LARK_VERIFICATION_TOKEN", ""),
			EncryptKey:        env("LARK_ENCRYPT_KEY", ""),
			AllowedChatIDs:    envCSV("LARK_ALLOWED_CHAT_IDS"),
		},
		LLM: LLMConfig{
			Provider:       env("LLM_PROVIDER", "local_rules"),
			BaseURL:        env("LLM_BASE_URL", ""),
			APIKey:         env("LLM_API_KEY", ""),
			Model:          env("LLM_MODEL", "rules-v1"),
			TimeoutSeconds: envInt("LLM_TIMEOUT_SECONDS", 30),
			MaxConcurrency: envInt("LLM_MAX_CONCURRENCY", 10),
		},
		Database: DatabaseConfig{
			Driver: env("DB_DRIVER", "mysql"),
			DSN:    env("DB_DSN", ""),
		},
		Queue: QueueConfig{
			Type:       env("QUEUE_TYPE", "memory"),
			RedisAddr:  env("REDIS_ADDR", ""),
			StreamName: env("QUEUE_STREAM_NAME", "case_events"),
		},
		Connectors: ConnectorConfig{
			Mode:           env("CONNECTOR_MODE", "mock"),
			APIKey:         env("CONNECTOR_API_KEY", ""),
			TimeoutSeconds: envInt("CONNECTOR_TIMEOUT_SECONDS", 5),
			MarketBaseURL:  env("MARKET_READONLY_BASE_URL", ""),
			AssetBaseURL:   env("ASSET_READONLY_BASE_URL", ""),
			OpsBaseURL:     env("OPS_READONLY_BASE_URL", ""),
		},
		ToolGateway: ToolGatewayConfig{
			Endpoint:     env("TOOL_GATEWAY_ENDPOINT", "http://localhost:8080"),
			ClientID:     env("TOOL_GATEWAY_CLIENT_ID", "dev-client"),
			ClientSecret: env("TOOL_GATEWAY_CLIENT_SECRET", ""),
		},
		Limits: LimitsConfig{
			MaxToolCallsPerCase:        envInt("MAX_TOOL_CALLS_PER_CASE", 10),
			MaxLLMCallsPerCase:         envInt("MAX_LLM_CALLS_PER_CASE", 8),
			DefaultLogTimeRangeMinutes: envInt("DEFAULT_LOG_TIME_RANGE_MINUTES", 30),
			DefaultToolTimeoutSeconds:  envInt("DEFAULT_TOOL_TIMEOUT_SECONDS", 5),
			WorkerConcurrency:          envInt("WORKER_CONCURRENCY", 4),
		},
	}
}

func env(key string, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func envCSV(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
