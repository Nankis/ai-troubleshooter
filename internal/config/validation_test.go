package config

import (
	"strings"
	"testing"
)

func TestLoadFromEnvDefaultsToLarkPlatform(t *testing.T) {
	t.Setenv("LARK_PLATFORM", "")
	t.Setenv("LARK_API_BASE_URL", "")

	cfg := LoadFromEnv()
	if cfg.Lark.Platform != "lark" {
		t.Fatalf("expected lark platform by default, got %q", cfg.Lark.Platform)
	}
	if cfg.Lark.APIBaseURL != "https://open.larksuite.com" {
		t.Fatalf("expected lark OpenAPI base URL, got %q", cfg.Lark.APIBaseURL)
	}
}

func TestLoadFromEnvSupportsFeishuPlatform(t *testing.T) {
	t.Setenv("LARK_PLATFORM", "feishu")
	t.Setenv("LARK_API_BASE_URL", "")

	cfg := LoadFromEnv()
	if cfg.Lark.Platform != "feishu" {
		t.Fatalf("expected feishu platform, got %q", cfg.Lark.Platform)
	}
	if cfg.Lark.APIBaseURL != "https://open.feishu.cn" {
		t.Fatalf("expected feishu OpenAPI base URL, got %q", cfg.Lark.APIBaseURL)
	}
}

func TestLoadFromEnvAllowsExplicitLarkAPIBaseURL(t *testing.T) {
	t.Setenv("LARK_PLATFORM", "lark")
	t.Setenv("LARK_API_BASE_URL", "https://proxy.example.internal/lark")

	cfg := LoadFromEnv()
	if cfg.Lark.APIBaseURL != "https://proxy.example.internal/lark" {
		t.Fatalf("expected explicit OpenAPI base URL, got %q", cfg.Lark.APIBaseURL)
	}
}

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

func TestValidateForLarkBotFailsClosedInProd(t *testing.T) {
	cfg := Config{Server: ServerConfig{Env: "prod"}}
	err := cfg.ValidateForLarkBot()
	if err == nil || !strings.Contains(err.Error(), "LARK_VERIFICATION_TOKEN") || !strings.Contains(err.Error(), "LARK_ALLOWED_CHAT_IDS") {
		t.Fatalf("expected lark prod errors, got %v", err)
	}
}

func TestValidateForLarkBotRejectsUnknownPlatform(t *testing.T) {
	cfg := Config{Lark: LarkConfig{Platform: "unknown"}}
	err := cfg.ValidateForLarkBot()
	if err == nil || !strings.Contains(err.Error(), "LARK_PLATFORM") {
		t.Fatalf("expected platform validation error, got %v", err)
	}
}
