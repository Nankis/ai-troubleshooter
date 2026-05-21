package llm

import (
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/config"
)

func NewFromConfig(cfg config.LLMConfig) LLMClient {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "local", "local_rules", "rules":
		return NewRuleBasedClient()
	case "openai_compatible", "gpt", "claude", "llm_gateway":
		return NewOpenAICompatibleClient(OpenAICompatibleOptions{
			Provider: cfg.Provider,
			BaseURL:  cfg.BaseURL,
			APIKey:   cfg.APIKey,
			Model:    cfg.Model,
			Timeout:  time.Duration(cfg.TimeoutSeconds) * time.Second,
		})
	default:
		return NewRuleBasedClient()
	}
}
