package vision

import (
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/config"
)

func NewFromConfig(cfg config.VisionConfig) Client {
	return NewFromConfigs(cfg, config.LLMConfig{})
}

func NewFromConfigs(cfg config.VisionConfig, llmCfg config.LLMConfig) Client {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch provider {
	case "", "off", "disabled", "none":
		return nil
	case "same_as_llm", "llm", "main_llm":
		return newFromLLMConfig(cfg, llmCfg)
	case "qwen", "dashscope", "deepseek", "moonshot", "openai", "qwen_vl", "qwen_openai_compatible", "openai_compatible":
		return NewOpenAICompatibleClient(OpenAICompatibleOptions{
			Provider: cfg.Provider,
			BaseURL:  cfg.BaseURL,
			APIKey:   cfg.APIKey,
			Model:    cfg.Model,
			Timeout:  time.Duration(cfg.TimeoutSeconds) * time.Second,
		})
	default:
		return NewLocalClient()
	}
}

func newFromLLMConfig(visionCfg config.VisionConfig, llmCfg config.LLMConfig) Client {
	switch strings.ToLower(strings.TrimSpace(llmCfg.Provider)) {
	case "openai", "openai_compatible", "gpt", "claude", "llm_gateway", "qwen", "dashscope", "deepseek", "moonshot":
		timeoutSeconds := visionCfg.TimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = llmCfg.TimeoutSeconds
		}
		return NewOpenAICompatibleClient(OpenAICompatibleOptions{
			Provider: "same_as_llm:" + strings.TrimSpace(llmCfg.Provider),
			BaseURL:  fallback(visionCfg.BaseURL, llmCfg.BaseURL),
			APIKey:   fallback(visionCfg.APIKey, llmCfg.APIKey),
			Model:    fallback(visionCfg.Model, llmCfg.Model),
			Timeout:  time.Duration(timeoutSeconds) * time.Second,
		})
	default:
		return NewLocalClient()
	}
}
