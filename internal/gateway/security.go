package gateway

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/ratelimit"
)

var (
	ErrUnauthenticated = errors.New("gateway authentication required")
	ErrAgentMismatch   = errors.New("authenticated agent does not match request agent")
	ErrRateLimited     = errors.New("gateway rate limit exceeded")
)

type SecurityConfig struct {
	AuthEnabled bool
	// BearerTokens maps token values to agent ids.
	BearerTokens                  map[string]string
	AllowUnauthenticatedListTools bool
	AgentQPS                      int
	UserQPS                       int
	ToolQPS                       int
}

type authenticatedAgentKey struct{}

type rateLimiters struct {
	agent *ratelimit.FixedWindow
	user  *ratelimit.FixedWindow
	tool  *ratelimit.FixedWindow
}

func newRateLimiters(cfg SecurityConfig) rateLimiters {
	return rateLimiters{
		agent: ratelimit.NewFixedWindow(cfg.AgentQPS, time.Second),
		user:  ratelimit.NewFixedWindow(cfg.UserQPS, time.Second),
		tool:  ratelimit.NewFixedWindow(cfg.ToolQPS, time.Second),
	}
}

func contextWithAuthenticatedAgent(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, authenticatedAgentKey{}, agentID)
}

func authenticatedAgentFromContext(ctx context.Context) string {
	v, _ := ctx.Value(authenticatedAgentKey{}).(string)
	return v
}

func bearerToken(r *http.Request) string {
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

func authenticateBearer(tokens map[string]string, token string) (string, bool) {
	if token == "" {
		return "", false
	}
	for allowedToken, agentID := range tokens {
		allowedHash := sha256.Sum256([]byte(allowedToken))
		tokenHash := sha256.Sum256([]byte(token))
		if subtle.ConstantTimeCompare(allowedHash[:], tokenHash[:]) == 1 {
			return agentID, true
		}
	}
	return "", false
}
