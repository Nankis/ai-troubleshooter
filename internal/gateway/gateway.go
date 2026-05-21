package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ginseng/ai-troubleshooter/internal/audit"
	"github.com/ginseng/ai-troubleshooter/internal/masking"
	"github.com/ginseng/ai-troubleshooter/internal/policy"
	"github.com/ginseng/ai-troubleshooter/internal/tool"
)

var (
	ErrDenied           = errors.New("tool invocation denied")
	ErrInvalidArguments = errors.New("invalid tool arguments")
)

type Gateway struct {
	registry *tool.Registry
	policy   policy.Engine
	audit    audit.Sink
	timeout  time.Duration
}

func New(registry *tool.Registry, policyEngine policy.Engine, auditSink audit.Sink, timeout time.Duration) *Gateway {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Gateway{
		registry: registry,
		policy:   policyEngine,
		audit:    auditSink,
		timeout:  timeout,
	}
}

func (g *Gateway) Registry() *tool.Registry {
	return g.registry
}

func (g *Gateway) LocalClient() LocalClient {
	return LocalClient{gateway: g}
}

type LocalClient struct {
	gateway *Gateway
}

func (c LocalClient) Invoke(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
	return c.gateway.Invoke(ctx, req)
}

func (g *Gateway) Invoke(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
	start := time.Now()
	toolCallID := newID("tc")
	resp := tool.InvocationResponse{
		ToolCallID: toolCallID,
		Status:     "failed",
		Warnings:   []string{},
	}

	spec, ok := g.registry.Get(req.ToolName)
	if !ok {
		resp.Summary = "tool not found"
		return resp, tool.ErrToolNotFound
	}
	if spec.Status != "" && spec.Status != "enabled" {
		resp.Summary = "tool disabled"
		return resp, tool.ErrToolDisabled
	}

	decision, err := g.policy.Authorize(ctx, policy.Request{
		CaseID:              req.CaseID,
		AgentID:             req.AgentID,
		LarkUserID:          req.LarkUserID,
		ChatID:              req.ChatID,
		ToolName:            spec.Name,
		RequiredScope:       spec.RequiredScope,
		Arguments:           req.Arguments,
		RequestedAt:         start,
		MaxLimit:            spec.MaxLimit,
		MaxTimeRangeMinutes: spec.MaxTimeRangeMinutes,
	})
	if err != nil {
		resp.Summary = err.Error()
		g.recordAudit(ctx, req, spec, toolCallID, "error", "", "", 0, start, err)
		return resp, err
	}
	if !decision.Allowed {
		resp.Status = "denied"
		resp.Summary = decision.Reason
		g.recordAudit(ctx, req, spec, toolCallID, "denied", decision.Reason, "", 0, start, nil)
		return resp, ErrDenied
	}
	if err := validateBoundary(req.Arguments, decision); err != nil {
		resp.Status = "denied"
		resp.Summary = err.Error()
		g.recordAudit(ctx, req, spec, toolCallID, "denied", err.Error(), "", 0, start, err)
		return resp, ErrInvalidArguments
	}

	handler, ok := g.registry.Handler(spec.Name)
	if !ok {
		resp.Summary = "tool handler not registered"
		g.recordAudit(ctx, req, spec, toolCallID, "error", "", "", 0, start, tool.ErrToolNotFound)
		return resp, tool.ErrToolNotFound
	}

	callCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()
	out, err := handler(callCtx, req)
	out.ToolCallID = toolCallID
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	if out.Status == "" {
		out.Status = "success"
	}
	if out.QueryID == "" {
		out.QueryID = newID("query")
	}
	out.Data = maskAny(out.Data)
	if err != nil {
		out.Status = "failed"
		out.Summary = err.Error()
		g.recordAudit(ctx, req, spec, toolCallID, "allowed", "", out.QueryID, 0, start, err)
		return out, err
	}
	g.recordAudit(ctx, req, spec, toolCallID, "allowed", "", out.QueryID, resultCount(out.Data), start, nil)
	return out, nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/tools":
		writeJSON(w, http.StatusOK, map[string]any{"tools": g.registry.List()})
		return
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/tools/") && strings.HasSuffix(r.URL.Path, "/invoke"):
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/tools/"), "/invoke")
		name = strings.Trim(name, "/")
		var req tool.InvocationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		req.ToolName = name
		if req.Arguments == nil {
			req.Arguments = map[string]any{}
		}
		resp, err := g.Invoke(r.Context(), req)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrDenied) {
				status = http.StatusForbidden
			}
			if errors.Is(err, ErrInvalidArguments) || errors.Is(err, tool.ErrToolNotFound) || errors.Is(err, tool.ErrToolDisabled) {
				status = http.StatusBadRequest
			}
			writeJSON(w, status, resp)
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
}

func validateBoundary(args map[string]any, decision policy.Decision) error {
	if decision.MaxLimit > 0 {
		limit, ok := intArg(args, "limit")
		if ok && limit > decision.MaxLimit {
			return fmt.Errorf("limit %d exceeds max %d", limit, decision.MaxLimit)
		}
	}
	start, hasStart, err := timeArg(args, "start_time")
	if err != nil {
		return err
	}
	end, hasEnd, err := timeArg(args, "end_time")
	if err != nil {
		return err
	}
	if hasStart && hasEnd {
		if end.Before(start) {
			return fmt.Errorf("end_time must be after start_time")
		}
		if decision.MaxTimeRangeMinutes > 0 && end.Sub(start) > time.Duration(decision.MaxTimeRangeMinutes)*time.Minute {
			return fmt.Errorf("time range exceeds max %d minutes", decision.MaxTimeRangeMinutes)
		}
	}
	return nil
}

func (g *Gateway) recordAudit(ctx context.Context, req tool.InvocationRequest, spec tool.Spec, toolCallID string, decision string, denyReason string, queryID string, resultCount int, start time.Time, err error) {
	if g.audit == nil {
		return
	}
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	_ = g.audit.Record(ctx, audit.Record{
		ToolCallID:       toolCallID,
		CaseID:           req.CaseID,
		AgentID:          req.AgentID,
		LarkUserID:       req.LarkUserID,
		ToolName:         spec.Name,
		RequiredScope:    spec.RequiredScope,
		ArgumentsSummary: argumentsSummary(req.Arguments),
		PolicyDecision:   decision,
		DenyReason:       denyReason,
		QueryID:          queryID,
		ResultCount:      resultCount,
		LatencyMS:        time.Since(start).Milliseconds(),
		ErrorMessage:     errorMessage,
		CreatedAt:        time.Now(),
	})
}

func argumentsSummary(args map[string]any) string {
	masked := masking.MaskValue(args)
	b, err := json.Marshal(masked)
	if err != nil {
		return "{}"
	}
	if len(b) > 512 {
		return string(b[:512]) + "..."
	}
	return string(b)
}

func maskAny(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return masking.MaskValue(v)
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return masking.MaskValue(v)
	}
	return masking.MaskValue(decoded)
}

func resultCount(data any) int {
	switch v := data.(type) {
	case []any:
		return len(v)
	case map[string]any:
		for _, key := range []string{"items", "events", "candles", "samples", "errors"} {
			if list, ok := v[key].([]any); ok {
				return len(list)
			}
		}
	}
	return 1
}

func intArg(args map[string]any, key string) (int, bool) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		i, err := strconv.Atoi(v.String())
		return i, err == nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		return i, err == nil
	default:
		return 0, false
	}
}

func timeArg(args map[string]any, key string) (time.Time, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return time.Time{}, false, nil
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return time.Time{}, false, fmt.Errorf("%s must be RFC3339 string", key)
	}
	t, err := parseTime(value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%s parse failed: %w", key, err)
	}
	return t, true, nil
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
