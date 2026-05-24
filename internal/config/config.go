package config

import (
	"encoding/json"
	"fmt"
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
	Provider          string
	Profile           string
	ConfigFile        string
	BaseURL           string
	APIKey            string
	Model             string
	TimeoutSeconds    int
	MaxConcurrency    int
	AllowRuleFallback bool
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
	Mode              string
	APIKey            string
	TimeoutSeconds    int
	MarketBaseURL     string
	AssetBaseURL      string
	OpsBaseURL        string
	HealthFoodBaseURL string
}

type GatewayConfig struct {
	AuthEnabled                   bool
	AgentID                       string
	BearerTokens                  map[string]string
	AllowUnauthenticatedListTools bool
	AgentQPS                      int
	UserQPS                       int
	ToolQPS                       int
	Agents                        []GatewayAgentConfig
	AgentConfigError              string
	AgentTokenErrors              []string
}

type GatewayAgentConfig struct {
	AgentID        string   `json:"agent_id"`
	Status         string   `json:"status"`
	BearerTokenEnv string   `json:"bearer_token_env"`
	AllowedScopes  []string `json:"allowed_scopes"`
	AllowedTools   []string `json:"allowed_tools"`
	AllowedChatIDs []string `json:"allowed_chat_ids"`
	RateLimitQPS   int      `json:"rate_limit_qps"`
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
	gatewayAgents, gatewayAgentConfigErr := loadGatewayAgentsFromEnv()
	gatewayAgentTokens, gatewayAgentTokenErrors := gatewayAgentTokenMap(gatewayAgents)
	bearerTokens := mergeTokenMaps(envTokenMap("GATEWAY_BEARER_TOKENS"), gatewayAgentTokens)
	llmCfg := LLMConfig{
		Provider:          "local_rules",
		Model:             "rules-v1",
		TimeoutSeconds:    envInt("LLM_TIMEOUT_SECONDS", 30),
		MaxConcurrency:    envInt("LLM_MAX_CONCURRENCY", 10),
		AllowRuleFallback: envBool("LLM_ALLOW_RULE_FALLBACK", false),
	}
	visionCfg := VisionConfig{
		Provider:            "same_as_llm",
		TimeoutSeconds:      envInt("VISION_TIMEOUT_SECONDS", 30),
		MaxImagesPerMessage: envInt("VISION_MAX_IMAGES_PER_MESSAGE", 3),
		MaxImageBytes:       envInt("VISION_MAX_IMAGE_BYTES", 10*1024*1024),
	}
	applyModelProfileFromEnv(&llmCfg, &visionCfg)
	overrideModelConfigFromEnv(&llmCfg, &visionCfg)
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
		LLM:    llmCfg,
		Vision: visionCfg,
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
			Mode:              env("CONNECTOR_MODE", "mock"),
			APIKey:            env("CONNECTOR_API_KEY", ""),
			TimeoutSeconds:    envInt("CONNECTOR_TIMEOUT_SECONDS", 5),
			MarketBaseURL:     env("MARKET_READONLY_BASE_URL", ""),
			AssetBaseURL:      env("ASSET_READONLY_BASE_URL", ""),
			OpsBaseURL:        env("OPS_READONLY_BASE_URL", ""),
			HealthFoodBaseURL: env("HEALTH_FOOD_READONLY_BASE_URL", ""),
		},
		Gateway: GatewayConfig{
			AuthEnabled:                   envBool("GATEWAY_AUTH_ENABLED", false),
			AgentID:                       env("GATEWAY_AGENT_ID", "business-troubleshooter-v1"),
			BearerTokens:                  bearerTokens,
			AllowUnauthenticatedListTools: envBool("GATEWAY_ALLOW_UNAUTHENTICATED_LIST_TOOLS", false),
			AgentQPS:                      envInt("GATEWAY_AGENT_QPS", 20),
			UserQPS:                       envInt("GATEWAY_USER_QPS", 10),
			ToolQPS:                       envInt("GATEWAY_TOOL_QPS", 20),
			Agents:                        gatewayAgents,
			AgentConfigError:              gatewayAgentConfigErr,
			AgentTokenErrors:              gatewayAgentTokenErrors,
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

func applyModelProfileFromEnv(llm *LLMConfig, vision *VisionConfig) {
	profile := firstEnv("AI_MODEL_PROFILE", "MODEL_PROFILE")
	configFile := firstEnv("AI_MODEL_CONFIG_FILE", "MODEL_CONFIG_FILE")
	if profile == "" {
		return
	}
	normalized := strings.ToLower(strings.TrimSpace(profile))
	llm.Profile = normalized
	llm.ConfigFile = configFile
	switch normalized {
	case "local", "local_rules", "rules":
		llm.Provider = "local_rules"
		llm.Model = "rules-v1"
		vision.Provider = "same_as_llm"
		return
	case "qwen", "dashscope":
		llm.Provider = normalized
		llm.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		llm.Model = "qwen-plus"
		llm.APIKey = firstEnv("DASHSCOPE_API_KEY", "QWEN_API_KEY")
		vision.Provider = "same_as_llm"
	case "deepseek":
		llm.Provider = normalized
		llm.BaseURL = "https://api.deepseek.com"
		llm.Model = "deepseek-chat"
		llm.APIKey = firstEnv("DEEPSEEK_API_KEY")
		vision.Provider = "same_as_llm"
	case "moonshot":
		llm.Provider = normalized
		llm.BaseURL = "https://api.moonshot.cn/v1"
		llm.Model = "moonshot-v1-8k"
		llm.APIKey = firstEnv("MOONSHOT_API_KEY")
		vision.Provider = "same_as_llm"
	case "openai":
		llm.Provider = normalized
		llm.BaseURL = "https://api.openai.com/v1"
		llm.Model = "gpt-4.1-mini"
		llm.APIKey = firstEnv("OPENAI_API_KEY")
		vision.Provider = "same_as_llm"
	default:
		llm.Provider = "openai_compatible"
		vision.Provider = "same_as_llm"
	}
	if configFile != "" {
		if loaded, ok := loadModelProfileFile(configFile, normalized); ok {
			if loaded.BaseURL != "" {
				llm.BaseURL = loaded.BaseURL
			}
			if loaded.APIKey != "" {
				llm.APIKey = loaded.APIKey
			}
			if loaded.Model != "" {
				llm.Model = loaded.Model
			}
		}
	}
}

func overrideModelConfigFromEnv(llm *LLMConfig, vision *VisionConfig) {
	if value, ok := lookupEnv("LLM_PROVIDER"); ok {
		llm.Provider = value
	}
	if value, ok := lookupEnv("LLM_BASE_URL"); ok {
		llm.BaseURL = value
	}
	if value, ok := lookupEnv("LLM_API_KEY"); ok {
		llm.APIKey = value
	}
	if value, ok := lookupEnv("LLM_MODEL"); ok {
		llm.Model = value
	}
	if value, ok := lookupEnv("VISION_PROVIDER"); ok {
		vision.Provider = value
	}
	if value, ok := lookupEnv("VISION_BASE_URL"); ok {
		vision.BaseURL = value
	}
	if value, ok := lookupEnv("VISION_API_KEY"); ok {
		vision.APIKey = value
	}
	if value, ok := lookupEnv("VISION_MODEL"); ok {
		vision.Model = value
	}
}

type loadedModelProfile struct {
	BaseURL string
	APIKey  string
	Model   string
}

func loadModelProfileFile(path string, profile string) (loadedModelProfile, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return loadedModelProfile{}, false
	}
	lines := strings.Split(string(raw), "\n")
	start := -1
	baseIndent := 0
	for idx, line := range lines {
		trimmed := strings.TrimSpace(stripYAMLComment(line))
		if trimmed == profile+":" {
			start = idx
			baseIndent = indentOf(line)
			break
		}
	}
	if start < 0 {
		return loadedModelProfile{}, false
	}
	out := loadedModelProfile{}
	for _, line := range lines[start+1:] {
		if strings.TrimSpace(line) != "" && indentOf(line) <= baseIndent {
			break
		}
		key, value, ok := splitYAMLScalar(line)
		if !ok {
			continue
		}
		switch key {
		case "api-key":
			out.APIKey = resolvePropertyValue(value)
		case "env-api-key":
			if out.APIKey == "" {
				out.APIKey = resolvePropertyValue(value)
			}
		case "base-url-http", "base-url":
			if out.BaseURL == "" || key == "base-url-http" {
				out.BaseURL = resolvePropertyValue(value)
			}
		case "model":
			if out.Model == "" {
				out.Model = resolvePropertyValue(value)
			}
		}
	}
	return out, out.BaseURL != "" || out.APIKey != "" || out.Model != ""
}

func splitYAMLScalar(line string) (string, string, bool) {
	clean := strings.TrimSpace(stripYAMLComment(line))
	if clean == "" || !strings.Contains(clean, ":") {
		return "", "", false
	}
	parts := strings.SplitN(clean, ":", 2)
	key := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}

func stripYAMLComment(line string) string {
	if idx := strings.Index(line, " #"); idx >= 0 {
		return line[:idx]
	}
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return ""
	}
	return line
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func resolvePropertyValue(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		parts := strings.SplitN(inner, ":", 2)
		if envValue := strings.TrimSpace(os.Getenv(parts[0])); envValue != "" {
			return envValue
		}
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	return value
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value, ok := lookupEnv(key); ok {
			return value
		}
	}
	return ""
}

func lookupEnv(key string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value), strings.TrimSpace(value) != ""
	}
	return "", false
}

func loadGatewayAgentsFromEnv() ([]GatewayAgentConfig, string) {
	raw := strings.TrimSpace(os.Getenv("GATEWAY_AGENT_CONFIG_JSON"))
	if raw == "" {
		path := strings.TrimSpace(os.Getenv("GATEWAY_AGENT_CONFIG_FILE"))
		if path == "" {
			return nil, ""
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Sprintf("read GATEWAY_AGENT_CONFIG_FILE: %v", err)
		}
		raw = string(data)
	}
	if raw == "" {
		return nil, ""
	}
	agents, err := parseGatewayAgentConfig([]byte(raw))
	if err != nil {
		return nil, err.Error()
	}
	return agents, ""
}

func parseGatewayAgentConfig(raw []byte) ([]GatewayAgentConfig, error) {
	var wrapped struct {
		Agents []GatewayAgentConfig `json:"agents"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Agents) > 0 {
		return wrapped.Agents, nil
	}
	var direct []GatewayAgentConfig
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	return nil, fmt.Errorf("GATEWAY_AGENT_CONFIG_JSON must be {\"agents\":[...]} or an agent array")
}

func gatewayAgentTokenMap(agents []GatewayAgentConfig) (map[string]string, []string) {
	out := map[string]string{}
	errs := []string{}
	for _, agent := range agents {
		agentID := strings.TrimSpace(agent.AgentID)
		if agentID == "" || strings.TrimSpace(agent.BearerTokenEnv) == "" {
			continue
		}
		token := strings.TrimSpace(os.Getenv(strings.TrimSpace(agent.BearerTokenEnv)))
		if token == "" {
			errs = append(errs, fmt.Sprintf("%s is empty for gateway agent %s", agent.BearerTokenEnv, agentID))
			continue
		}
		out[token] = agentID
	}
	return out, errs
}

func mergeTokenMaps(values ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, value := range values {
		for token, agentID := range value {
			out[token] = agentID
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
