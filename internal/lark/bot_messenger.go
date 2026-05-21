package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/chatplatform"
)

const defaultAPIBaseURL = chatplatform.LarkOpenAPIBaseURL

type BotMessengerOptions struct {
	AppID     string
	AppSecret string
	BaseURL   string
	Timeout   time.Duration
}

type BotMessenger struct {
	appID     string
	appSecret string
	baseURL   string
	client    *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func NewBotMessenger(options BotMessengerOptions) *BotMessenger {
	baseURL := strings.TrimRight(options.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &BotMessenger{
		appID:     strings.TrimSpace(options.AppID),
		appSecret: strings.TrimSpace(options.AppSecret),
		baseURL:   baseURL,
		client:    &http.Client{Timeout: timeout},
	}
}

func (m *BotMessenger) SendMessage(ctx context.Context, chatID string, threadID string, text string) error {
	if m.appID == "" || m.appSecret == "" {
		return fmt.Errorf("lark app id and app secret are required")
	}
	token, err := m.tenantAccessToken(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(threadID) != "" {
		if err := m.replyText(ctx, token, threadID, text); err == nil {
			return nil
		}
	}
	if strings.TrimSpace(chatID) == "" {
		return fmt.Errorf("chat id is required")
	}
	return m.sendChatText(ctx, token, chatID, text)
}

func (m *BotMessenger) DownloadImage(ctx context.Context, messageID string, imageKey string) (DownloadedImage, error) {
	if m.appID == "" || m.appSecret == "" {
		return DownloadedImage{}, fmt.Errorf("lark app id and app secret are required")
	}
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(imageKey) == "" {
		return DownloadedImage{}, fmt.Errorf("message id and image key are required")
	}
	token, err := m.tenantAccessToken(ctx)
	if err != nil {
		return DownloadedImage{}, err
	}
	path := "/open-apis/im/v1/messages/" + url.PathEscape(messageID) + "/resources/" + url.PathEscape(imageKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+path, nil)
	if err != nil {
		return DownloadedImage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := m.client.Do(req)
	if err != nil {
		return DownloadedImage{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return DownloadedImage{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DownloadedImage{}, fmt.Errorf("lark image download status=%d body=%s", resp.StatusCode, string(data))
	}
	mediaType := resp.Header.Get("Content-Type")
	if mediaType == "" {
		mediaType = http.DetectContentType(data)
	}
	return DownloadedImage{ImageKey: imageKey, MediaType: mediaType, Data: data}, nil
}

func (m *BotMessenger) tenantAccessToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	if m.cachedToken != "" && time.Now().Before(m.tokenExpiry) {
		token := m.cachedToken
		m.mu.Unlock()
		return token, nil
	}
	m.mu.Unlock()

	var out struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int64  `json:"expire"`
	}
	if err := m.postJSON(ctx, "/open-apis/auth/v3/tenant_access_token/internal", "", map[string]any{
		"app_id":     m.appID,
		"app_secret": m.appSecret,
	}, &out); err != nil {
		return "", err
	}
	if out.Code != 0 || out.TenantAccessToken == "" {
		return "", fmt.Errorf("lark tenant token failed code=%d msg=%s", out.Code, out.Msg)
	}
	expire := out.Expire
	if expire <= 0 {
		expire = 7200
	}
	m.mu.Lock()
	m.cachedToken = out.TenantAccessToken
	m.tokenExpiry = time.Now().Add(time.Duration(expire-60) * time.Second)
	m.mu.Unlock()
	return out.TenantAccessToken, nil
}

func (m *BotMessenger) replyText(ctx context.Context, token string, messageID string, text string) error {
	var out apiResponse
	return m.postJSON(ctx, "/open-apis/im/v1/messages/"+messageID+"/reply", token, textMessageBody(text), &out)
}

func (m *BotMessenger) sendChatText(ctx context.Context, token string, chatID string, text string) error {
	var out apiResponse
	body := textMessageBody(text)
	body["receive_id"] = chatID
	return m.postJSON(ctx, "/open-apis/im/v1/messages?receive_id_type=chat_id", token, body, &out)
}

func textMessageBody(text string) map[string]any {
	content, _ := json.Marshal(map[string]string{"text": text})
	return map[string]any{
		"msg_type": "text",
		"content":  string(content),
	}
}

type apiResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (m *BotMessenger) postJSON(ctx context.Context, path string, token string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("lark api status=%d", resp.StatusCode)
	}
	if apiOut, ok := out.(*apiResponse); ok && apiOut.Code != 0 {
		return fmt.Errorf("lark api failed code=%d msg=%s", apiOut.Code, apiOut.Msg)
	}
	return nil
}
