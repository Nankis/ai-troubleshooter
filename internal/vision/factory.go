package vision

import (
	"strings"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/config"
)

func NewFromConfig(cfg config.VisionConfig) Client {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "off", "disabled", "none":
		return nil
	case "qwen", "qwen_vl", "qwen_openai_compatible", "openai_compatible":
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
