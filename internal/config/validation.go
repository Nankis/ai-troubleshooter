package config

import (
	"fmt"
	"strings"
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

func validationError(messages []string) error {
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("invalid config: %s", strings.Join(messages, "; "))
}
