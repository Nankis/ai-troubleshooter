package config

import (
	"strings"
	"testing"
)

func TestValidateForGatewayFailsClosedInProd(t *testing.T) {
	cfg := Config{Server: ServerConfig{Env: "prod"}}
	err := cfg.ValidateForGateway()
	if err == nil || !strings.Contains(err.Error(), "GATEWAY_AUTH_ENABLED") {
		t.Fatalf("expected gateway auth error, got %v", err)
	}
}

func TestValidateForGatewayRequiresTokensWhenEnabled(t *testing.T) {
	cfg := Config{
		Server:  ServerConfig{Env: "dev"},
		Gateway: GatewayConfig{AuthEnabled: true},
	}
	err := cfg.ValidateForGateway()
	if err == nil || !strings.Contains(err.Error(), "bearer_token_env") {
		t.Fatalf("expected gateway token error, got %v", err)
	}
}

func TestLoadFromEnvSupportsGatewayAgentConfigJSON(t *testing.T) {
	t.Setenv("GATEWAY_AGENT_CONFIG_JSON", `{
	  "agents": [{
	    "agent_id": "health-agent",
	    "status": "enabled",
	    "bearer_token_env": "HEALTH_AGENT_TOKEN",
	    "allowed_scopes": ["health_food:user:read"],
	    "allowed_tools": ["get_health_food_user_profile"],
	    "allowed_chat_ids": ["oc_allowed"]
	  }]
	}`)
	t.Setenv("HEALTH_AGENT_TOKEN", "test-token")
	t.Setenv("GATEWAY_AUTH_ENABLED", "true")

	cfg := LoadFromEnv()
	if err := cfg.ValidateForGateway(); err != nil {
		t.Fatalf("expected valid gateway config, got %v", err)
	}
	if len(cfg.Gateway.Agents) != 1 || cfg.Gateway.Agents[0].AgentID != "health-agent" {
		t.Fatalf("unexpected agents: %+v", cfg.Gateway.Agents)
	}
	if cfg.Gateway.BearerTokens["test-token"] != "health-agent" {
		t.Fatalf("expected bearer token to map to health-agent, got %+v", cfg.Gateway.BearerTokens)
	}
}

func TestValidateForGatewayRejectsInvalidAgentConfig(t *testing.T) {
	cfg := Config{
		Gateway: GatewayConfig{
			Agents: []GatewayAgentConfig{{AgentID: "bad-agent"}},
		},
	}
	err := cfg.ValidateForGateway()
	if err == nil || !strings.Contains(err.Error(), "allowed_scopes") {
		t.Fatalf("expected allowed scopes validation error, got %v", err)
	}
}

func TestLoadFromEnvReportsMissingGatewayAgentTokenWhenAuthEnabled(t *testing.T) {
	t.Setenv("GATEWAY_AUTH_ENABLED", "true")
	t.Setenv("GATEWAY_AGENT_CONFIG_JSON", `{"agents":[{"agent_id":"health-agent","bearer_token_env":"MISSING_HEALTH_AGENT_TOKEN","allowed_scopes":["health_food:user:read"]}]}`)

	cfg := LoadFromEnv()
	err := cfg.ValidateForGateway()
	if err == nil || !strings.Contains(err.Error(), "MISSING_HEALTH_AGENT_TOKEN") {
		t.Fatalf("expected missing token env error, got %v", err)
	}
}

func TestValidateForControlAPIRequiresTokenWhenEnabled(t *testing.T) {
	cfg := Config{
		ControlAPI: ControlAPIConfig{AuthEnabled: true},
	}
	err := cfg.ValidateForControlAPI()
	if err == nil || !strings.Contains(err.Error(), "CONTROL_API_BEARER_TOKENS") {
		t.Fatalf("expected control api token error, got %v", err)
	}
}

func TestLoadFromEnvIgnoresModelSpecificEnvironment(t *testing.T) {
	t.Setenv("AI_MODEL_PROFILE", "qwen")
	t.Setenv("LLM_PROVIDER", "openai_compatible")
	t.Setenv("VISION_PROVIDER", "qwen_openai_compatible")
	t.Setenv("LARK_PLATFORM", "feishu")

	cfg := LoadFromEnv()
	if cfg.Server.HTTPPort == 0 || cfg.Connectors.TimeoutSeconds == 0 {
		t.Fatalf("expected gateway config to load normally, got %+v", cfg)
	}
}
