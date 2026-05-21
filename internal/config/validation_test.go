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
	if err == nil || !strings.Contains(err.Error(), "GATEWAY_BEARER_TOKENS") {
		t.Fatalf("expected gateway token error, got %v", err)
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
