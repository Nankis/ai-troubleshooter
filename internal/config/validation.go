package config

import (
	"fmt"
	"strings"

	"github.com/Nankis/ai-troubleshooter/internal/chatplatform"
)

func (cfg Config) IsProd() bool {
	return strings.EqualFold(strings.TrimSpace(cfg.Server.Env), "prod")
}

func (cfg Config) ValidateForGateway() error {
	errs := []string{}
	if cfg.IsProd() && !cfg.Gateway.AuthEnabled {
		errs = append(errs, "GATEWAY_AUTH_ENABLED must be true in prod")
	}
	if cfg.Gateway.AuthEnabled && len(cfg.Gateway.BearerTokens) == 0 {
		errs = append(errs, "GATEWAY_BEARER_TOKENS is required when gateway auth is enabled")
	}
	return validationError(errs)
}

func (cfg Config) ValidateForControlAPI() error {
	errs := []string{}
	if cfg.IsProd() && !cfg.ControlAPI.AuthEnabled {
		errs = append(errs, "CONTROL_API_AUTH_ENABLED must be true in prod")
	}
	if cfg.ControlAPI.AuthEnabled && len(cfg.ControlAPI.BearerTokens) == 0 {
		errs = append(errs, "CONTROL_API_BEARER_TOKENS is required when control api auth is enabled")
	}
	return validationError(errs)
}

func (cfg Config) ValidateForLarkBot() error {
	errs := []string{}
	if strings.TrimSpace(cfg.Lark.Platform) != "" && !chatplatform.Supported(cfg.Lark.Platform) {
		errs = append(errs, "LARK_PLATFORM must be lark or feishu")
	}
	if cfg.IsProd() && strings.TrimSpace(cfg.Lark.VerificationToken) == "" {
		errs = append(errs, "LARK_VERIFICATION_TOKEN is required in prod")
	}
	if cfg.IsProd() && len(cfg.Lark.AllowedChatIDs) == 0 {
		errs = append(errs, "LARK_ALLOWED_CHAT_IDS is required in prod")
	}
	return validationError(errs)
}

func (cfg Config) ValidateForDevServer() error {
	return combineValidation(
		cfg.ValidateForGateway(),
		cfg.ValidateForControlAPI(),
		cfg.ValidateForLarkBot(),
	)
}

func (cfg Config) ValidateForOrchestrator() error {
	return combineValidation(
		cfg.ValidateForControlAPI(),
		cfg.ValidateForGateway(),
	)
}

func (cfg Config) ValidateForWorker() error {
	return cfg.ValidateForGateway()
}

func combineValidation(errs ...error) error {
	messages := []string{}
	for _, err := range errs {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	return validationError(messages)
}

func validationError(messages []string) error {
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("invalid config: %s", strings.Join(messages, "; "))
}
