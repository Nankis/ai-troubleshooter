package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

type ImageInput struct {
	ImageKey  string
	MediaType string
	Data      []byte
}

type Analysis struct {
	ModelProvider string
	ModelName     string
	OCRText       string
	Summary       string
}

type Client interface {
	AnalyzeImages(ctx context.Context, userText string, images []ImageInput) (Analysis, error)
}

type LocalClient struct{}

func NewLocalClient() LocalClient {
	return LocalClient{}
}

func (LocalClient) AnalyzeImages(ctx context.Context, userText string, images []ImageInput) (Analysis, error) {
	_ = ctx
	parts := []string{}
	for _, image := range images {
		if text := printableText(image.Data); text != "" {
			parts = append(parts, fmt.Sprintf("image_key=%s 识别文本：%s", image.ImageKey, text))
			continue
		}
		parts = append(parts, fmt.Sprintf("image_key=%s 已下载，media_type=%s，等待真实视觉模型识别", image.ImageKey, fallback(image.MediaType, "unknown")))
	}
	if len(parts) == 0 {
		parts = append(parts, "未发现可识别图片")
	}
	return Analysis{
		ModelProvider: "local_mock",
		ModelName:     "local-vision-mock",
		OCRText:       strings.Join(parts, "\n"),
		Summary:       "本地 mock 视觉识别结果；生产建议配置 Qwen-VL。",
	}, nil
}

type OpenAICompatibleClient struct {
	provider string
	baseURL  string
	apiKey   string
	model    string
	client   *http.Client
}

type OpenAICompatibleOptions struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

func NewOpenAICompatibleClient(options OpenAICompatibleOptions) *OpenAICompatibleClient {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OpenAICompatibleClient{
		provider: fallback(options.Provider, "openai_compatible"),
		baseURL:  normalizeChatCompletionsURL(options.BaseURL),
		apiKey:   strings.TrimSpace(options.APIKey),
		model:    fallback(options.Model, "qwen3-vl-plus"),
		client:   &http.Client{Timeout: timeout},
	}
}

func (c *OpenAICompatibleClient) AnalyzeImages(ctx context.Context, userText string, images []ImageInput) (Analysis, error) {
	if c.baseURL == "" {
		return Analysis{}, fmt.Errorf("vision base url is required")
	}
	if c.apiKey == "" {
		return Analysis{}, fmt.Errorf("vision api key is required")
	}
	if len(images) == 0 {
		return Analysis{}, fmt.Errorf("at least one image is required")
	}
	content := []map[string]any{{
		"type": "text",
		"text": buildVisionPrompt(userText),
	}}
	for _, image := range images {
		mediaType := fallback(image.MediaType, http.DetectContentType(image.Data))
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url": "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(image.Data),
			},
		})
	}
	body := map[string]any{
		"model":       c.model,
		"temperature": 0.1,
		"max_tokens":  1200,
		"messages": []map[string]any{
			{"role": "system", "content": "你是生产故障排查截图识别助手。只根据图片和用户文本提取客观信息，不要编造。"},
			{"role": "user", "content": content},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Analysis{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return Analysis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return Analysis{}, err
	}
	defer resp.Body.Close()
	var out struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Analysis{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if out.Error != nil && out.Error.Message != "" {
			return Analysis{}, fmt.Errorf("vision api status=%d error=%s", resp.StatusCode, out.Error.Message)
		}
		return Analysis{}, fmt.Errorf("vision api status=%d", resp.StatusCode)
	}
	if out.Error != nil && out.Error.Message != "" {
		return Analysis{}, fmt.Errorf("vision api error=%s", out.Error.Message)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return Analysis{}, fmt.Errorf("vision api returned empty content")
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	return Analysis{ModelProvider: c.provider, ModelName: c.model, OCRText: text, Summary: text}, nil
}

func buildVisionPrompt(userText string) string {
	return "请识别排障工单截图。输出中文，包含：1. 截图中所有可读文字/OCR；2. 币对、周期、用户/账户、时间、错误码、页面状态等关键字段；3. 你能确定的客观现象；4. 不确定或看不清的内容。用户补充文本：\n" + userText
}

func normalizeChatCompletionsURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return strings.TrimRight(base, "/") + "/chat/completions"
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	return parsed.String()
}

func printableText(data []byte) string {
	value := strings.TrimSpace(string(data))
	if value == "" || len(value) > 4096 {
		return ""
	}
	printable := 0
	for _, r := range value {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printable++
		}
	}
	if printable < len([]rune(value))*8/10 {
		return ""
	}
	return value
}

func fallback(v string, def string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}
