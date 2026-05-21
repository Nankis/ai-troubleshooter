package masking

import (
	"regexp"
	"strings"
)

var (
	emailPattern = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phonePattern = regexp.MustCompile(`\b1[3-9]\d{9}\b`)
	tokenPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|access[_-]?key)["'=:\s]+[A-Za-z0-9._\-]{8,}`)
)

var sensitiveKeys = map[string]bool{
	"phone":       true,
	"mobile":      true,
	"email":       true,
	"id_card":     true,
	"token":       true,
	"secret":      true,
	"api_key":     true,
	"access_key":  true,
	"address":     true,
	"raw_payload": true,
}

func MaskString(input string) string {
	out := emailPattern.ReplaceAllStringFunc(input, func(v string) string {
		parts := strings.SplitN(v, "@", 2)
		if len(parts) != 2 {
			return "[email]"
		}
		local := parts[0]
		if len(local) <= 2 {
			return "**@" + parts[1]
		}
		return local[:1] + "***" + local[len(local)-1:] + "@" + parts[1]
	})
	out = phonePattern.ReplaceAllStringFunc(out, func(v string) string {
		if len(v) != 11 {
			return "[phone]"
		}
		return v[:3] + "****" + v[7:]
	})
	out = tokenPattern.ReplaceAllString(out, "$1=[REDACTED]")
	return out
}

func MaskValue(value any) any {
	switch v := value.(type) {
	case string:
		return MaskString(v)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			if sensitiveKeys[strings.ToLower(key)] {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = MaskValue(child)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = MaskValue(child)
		}
		return out
	default:
		return value
	}
}
