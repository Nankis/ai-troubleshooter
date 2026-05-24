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
	if strings.TrimSpace(cfg.Gateway.AgentConfigError) != "" {
		errs = append(errs, cfg.Gateway.AgentConfigError)
	}
	errs = append(errs, validateGatewayAgents(cfg.Gateway.Agents)...)
	if cfg.Gateway.AuthEnabled && len(cfg.Gateway.BearerTokens) == 0 {
		errs = append(errs, "GATEWAY_BEARER_TOKENS or gateway agent bearer_token_env is required when gateway auth is enabled")
	}
	if cfg.Gateway.AuthEnabled {
		errs = append(errs, cfg.Gateway.AgentTokenErrors...)
	}
	return validationError(errs)
}

func validateGatewayAgents(agents []GatewayAgentConfig) []string {
	errs := []string{}
	seen := map[string]bool{}
	for idx, agent := range agents {
		agentID := strings.TrimSpace(agent.AgentID)
		if agentID == "" {
			errs = append(errs, fmt.Sprintf("gateway agents[%d].agent_id is required", idx))
			continue
		}
		if seen[agentID] {
			errs = append(errs, fmt.Sprintf("duplicate gateway agent_id %q", agentID))
		}
		seen[agentID] = true
		if len(agent.AllowedScopes) == 0 {
			errs = append(errs, fmt.Sprintf("gateway agent %q must define allowed_scopes", agentID))
		}
	}
	return errs
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

func (cfg Config) ValidateForLLM() error {
	errs := []string{}
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	switch provider {
	case "", "local", "local_rules", "rules":
		return nil
	case "openai", "openai_compatible", "gpt", "claude", "llm_gateway", "qwen", "dashscope", "deepseek", "moonshot":
		if strings.TrimSpace(cfg.LLM.BaseURL) == "" {
			errs = append(errs, "LLM_BASE_URL or AI_MODEL_PROFILE is required for real LLM provider")
		}
		if strings.TrimSpace(cfg.LLM.APIKey) == "" {
			errs = append(errs, "LLM_API_KEY or profile-specific API key is required for real LLM provider")
		}
		if strings.TrimSpace(cfg.LLM.Model) == "" {
			errs = append(errs, "LLM_MODEL or AI_MODEL_PROFILE is required for real LLM provider")
		}
	default:
		errs = append(errs, fmt.Sprintf("unsupported LLM_PROVIDER %q", cfg.LLM.Provider))
	}
	return validationError(errs)
}

func (cfg Config) ValidateForDevServer() error {
	return combineValidation(
		cfg.ValidateForGateway(),
		cfg.ValidateForControlAPI(),
		cfg.ValidateForLarkBot(),
		cfg.ValidateForLLM(),
	)
}

func (cfg Config) ValidateForBaselineOrchestrator() error {
	return combineValidation(
		cfg.ValidateForControlAPI(),
		cfg.ValidateForGateway(),
		cfg.ValidateForLLM(),
	)
}

func (cfg Config) ValidateForWorker() error {
	return combineValidation(cfg.ValidateForGateway(), cfg.ValidateForLLM())
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
