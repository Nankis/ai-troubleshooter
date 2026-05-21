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
