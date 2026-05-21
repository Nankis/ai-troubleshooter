package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/Nankis/ai-troubleshooter/internal/chatplatform"
)

type Config struct {
	Server      ServerConfig
	Lark        LarkConfig
	LLM         LLMConfig
	Vision      VisionConfig
	Database    DatabaseConfig
	Queue       QueueConfig
	Connectors  ConnectorConfig
	Gateway     GatewayConfig
	ControlAPI  ControlAPIConfig
	ToolGateway ToolGatewayConfig
	Limits      LimitsConfig
}

type ServerConfig struct {
	Env      string
	HTTPPort int
}

type LarkConfig struct {
	Platform          string
	AppID             string
	AppSecret         string
	APIBaseURL        string
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

type VisionConfig struct {
	Provider            string
	BaseURL             string
	APIKey              string
	Model               string
	TimeoutSeconds      int
	MaxImagesPerMessage int
	MaxImageBytes       int
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

type GatewayConfig struct {
	AuthEnabled                   bool
	BearerTokens                  map[string]string
	AllowUnauthenticatedListTools bool
	AgentQPS                      int
	UserQPS                       int
	ToolQPS                       int
}

type ControlAPIConfig struct {
	AuthEnabled  bool
	BearerTokens []string
}

type ToolGatewayConfig struct {
	Endpoint     string
	ClientID     string
	ClientSecret string
}

type LimitsConfig struct {
	MaxToolCallsPerCase        int
	MaxLLMCallsPerCase         int
	MaxToolFailuresPerCase     int
	MaxInvestigationSeconds    int
	DefaultLogTimeRangeMinutes int
	DefaultToolTimeoutSeconds  int
	WorkerConcurrency          int
}

func LoadFromEnv() Config {
	larkPlatform := chatplatform.Normalize(env("LARK_PLATFORM", chatplatform.PlatformLark))
	larkAPIBaseURL := env("LARK_API_BASE_URL", "")
	if larkAPIBaseURL == "" {
		larkAPIBaseURL = chatplatform.DefaultOpenAPIBaseURL(larkPlatform)
	}
	return Config{
		Server: ServerConfig{
			Env:      env("APP_ENV", "dev"),
			HTTPPort: envInt("HTTP_PORT", 8080),
		},
		Lark: LarkConfig{
			Platform:          larkPlatform,
			AppID:             env("LARK_APP_ID", ""),
			AppSecret:         env("LARK_APP_SECRET", ""),
			APIBaseURL:        larkAPIBaseURL,
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
		Vision: VisionConfig{
			Provider:            env("VISION_PROVIDER", "local_mock"),
			BaseURL:             env("VISION_BASE_URL", ""),
			APIKey:              env("VISION_API_KEY", ""),
			Model:               env("VISION_MODEL", "qwen3-vl-plus"),
			TimeoutSeconds:      envInt("VISION_TIMEOUT_SECONDS", 30),
			MaxImagesPerMessage: envInt("VISION_MAX_IMAGES_PER_MESSAGE", 3),
			MaxImageBytes:       envInt("VISION_MAX_IMAGE_BYTES", 10*1024*1024),
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
		Gateway: GatewayConfig{
			AuthEnabled:                   envBool("GATEWAY_AUTH_ENABLED", false),
			BearerTokens:                  envTokenMap("GATEWAY_BEARER_TOKENS"),
			AllowUnauthenticatedListTools: envBool("GATEWAY_ALLOW_UNAUTHENTICATED_LIST_TOOLS", false),
			AgentQPS:                      envInt("GATEWAY_AGENT_QPS", 20),
			UserQPS:                       envInt("GATEWAY_USER_QPS", 10),
			ToolQPS:                       envInt("GATEWAY_TOOL_QPS", 20),
		},
		ControlAPI: ControlAPIConfig{
			AuthEnabled:  envBool("CONTROL_API_AUTH_ENABLED", false),
			BearerTokens: envCSV("CONTROL_API_BEARER_TOKENS"),
		},
		ToolGateway: ToolGatewayConfig{
			Endpoint:     env("TOOL_GATEWAY_ENDPOINT", "http://localhost:8080"),
			ClientID:     env("TOOL_GATEWAY_CLIENT_ID", "dev-client"),
			ClientSecret: env("TOOL_GATEWAY_CLIENT_SECRET", ""),
		},
		Limits: LimitsConfig{
			MaxToolCallsPerCase:        envInt("MAX_TOOL_CALLS_PER_CASE", 10),
			MaxLLMCallsPerCase:         envInt("MAX_LLM_CALLS_PER_CASE", 8),
			MaxToolFailuresPerCase:     envInt("MAX_TOOL_FAILURES_PER_CASE", 3),
			MaxInvestigationSeconds:    envInt("MAX_INVESTIGATION_SECONDS", 120),
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

func envBool(key string, def bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return def
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
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

func envTokenMap(key string) map[string]string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pair := strings.SplitN(part, ":", 2)
		if len(pair) != 2 {
			continue
		}
		agentID := strings.TrimSpace(pair[0])
		token := strings.TrimSpace(pair[1])
		if agentID != "" && token != "" {
			out[token] = agentID
		}
	}
	return out
}
