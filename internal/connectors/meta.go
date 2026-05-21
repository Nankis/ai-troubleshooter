package connectors

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type RequestMeta struct {
	RequestID    string
	CaseID       string
	AgentID      string
	CallerUserID string
	ToolName     string
	Timeout      time.Duration
}

type requestMetaKey struct{}

func ContextWithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	if meta.RequestID == "" {
		meta.RequestID = newRequestID()
	}
	return context.WithValue(ctx, requestMetaKey{}, meta)
}

func RequestMetaFromContext(ctx context.Context) RequestMeta {
	meta, _ := ctx.Value(requestMetaKey{}).(RequestMeta)
	if meta.RequestID == "" {
		meta.RequestID = newRequestID()
	}
	return meta
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return "req_" + hex.EncodeToString(b[:])
}
