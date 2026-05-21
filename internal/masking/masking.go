package masking

import (
	"encoding/json"
	"reflect"
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
	if value == nil {
		return nil
	}
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
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface:
			if rv.IsNil() {
				return nil
			}
			return MaskValue(rv.Elem().Interface())
		case reflect.Slice, reflect.Array:
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = MaskValue(rv.Index(i).Interface())
			}
			return out
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String {
				return value
			}
			out := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				key := iter.Key().String()
				if sensitiveKeys[strings.ToLower(key)] {
					out[key] = "[REDACTED]"
					continue
				}
				out[key] = MaskValue(iter.Value().Interface())
			}
			return out
		case reflect.Struct:
			b, err := json.Marshal(value)
			if err != nil {
				return value
			}
			var decoded any
			if err := json.Unmarshal(b, &decoded); err != nil {
				return value
			}
			return MaskValue(decoded)
		}
		return value
	}
}
