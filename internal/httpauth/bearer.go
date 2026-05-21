package httpauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type Config struct {
	AuthEnabled  bool
	BearerTokens []string
}

func Require(config Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !config.AuthEnabled {
			next.ServeHTTP(w, r)
			return
		}
		if !TokenAllowed(config.BearerTokens, BearerToken(r)) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "bearer authentication required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireFunc(config Config, next http.HandlerFunc) http.Handler {
	return Require(config, next)
}

func BearerToken(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func TokenAllowed(tokens []string, token string) bool {
	if token == "" {
		return false
	}
	tokenHash := sha256.Sum256([]byte(token))
	for _, allowed := range tokens {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		allowedHash := sha256.Sum256([]byte(allowed))
		if subtle.ConstantTimeCompare(allowedHash[:], tokenHash[:]) == 1 {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
