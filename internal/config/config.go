package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Connectors ConnectorConfig
	Gateway    GatewayConfig
	ControlAPI ControlAPIConfig
	Limits     LimitsConfig
}

type ServerConfig struct {
	Env      string
	HTTPPort int
}

type DatabaseConfig struct {
	Driver string
	DSN    string
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

type LimitsConfig struct {
	DefaultToolTimeoutSeconds int
}

func LoadFromEnv() Config {
	gatewayAgents, gatewayAgentConfigErr := loadGatewayAgentsFromEnv()
	gatewayAgentTokens, gatewayAgentTokenErrors := gatewayAgentTokenMap(gatewayAgents)
	bearerTokens := mergeTokenMaps(envTokenMap("GATEWAY_BEARER_TOKENS"), gatewayAgentTokens)
	return Config{
		Server: ServerConfig{
			Env:      env("APP_ENV", "dev"),
			HTTPPort: envInt("HTTP_PORT", 8080),
		},
		Database: DatabaseConfig{
			Driver: env("DB_DRIVER", "mysql"),
			DSN:    env("DB_DSN", ""),
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
		Limits: LimitsConfig{
			DefaultToolTimeoutSeconds: envInt("DEFAULT_TOOL_TIMEOUT_SECONDS", 5),
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
