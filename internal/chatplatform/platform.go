package chatplatform

import "strings"

const (
	PlatformLark   = "lark"
	PlatformFeishu = "feishu"

	LarkOpenAPIBaseURL   = "https://open.larksuite.com"
	FeishuOpenAPIBaseURL = "https://open.feishu.cn"
)

func Normalize(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "", PlatformLark, "larksuite", "global", "intl", "international":
		return PlatformLark
	case PlatformFeishu, "feishu_cn", "cn", "china":
		return PlatformFeishu
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func Supported(platform string) bool {
	switch Normalize(platform) {
	case PlatformLark, PlatformFeishu:
		return true
	default:
		return false
	}
}

func DefaultOpenAPIBaseURL(platform string) string {
	switch Normalize(platform) {
	case PlatformFeishu:
		return FeishuOpenAPIBaseURL
	default:
		return LarkOpenAPIBaseURL
	}
}
