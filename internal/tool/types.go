package tool

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var (
	ErrToolNotFound = errors.New("tool not found")
	ErrToolDisabled = errors.New("tool disabled")
)

type Spec struct {
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	InputSchema         map[string]any `json:"input_schema_json"`
	OutputSchema        map[string]any `json:"output_schema_json,omitempty"`
	RequiredScope       string         `json:"required_scope"`
	BackendHandler      string         `json:"backend_handler"`
	MaxTimeRangeMinutes int            `json:"max_time_range_minutes,omitempty"`
	MaxLimit            int            `json:"max_limit,omitempty"`
	SensitivityLevel    string         `json:"sensitivity_level"`
	Status              string         `json:"status"`
}

type InvocationRequest struct {
	CaseID       string         `json:"case_id"`
	AgentID      string         `json:"agent_id"`
	CallerUserID string         `json:"caller_user_id,omitempty"`
	LarkUserID   string         `json:"lark_user_id,omitempty"`
	ChatID       string         `json:"chat_id"`
	ToolName     string         `json:"tool_name,omitempty"`
	Arguments    map[string]any `json:"arguments"`
}

type InvocationResponse struct {
	ToolCallID string   `json:"tool_call_id"`
	Status     string   `json:"status"`
	Data       any      `json:"data,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Warnings   []string `json:"warnings"`
	QueryID    string   `json:"query_id,omitempty"`
}

type HandlerFunc func(ctx context.Context, req InvocationRequest) (InvocationResponse, error)

type Registry struct {
	mu       sync.RWMutex
	specs    map[string]Spec
	handlers map[string]HandlerFunc
}

func NewRegistry() *Registry {
	return &Registry{
		specs:    map[string]Spec{},
		handlers: map[string]HandlerFunc{},
	}
}

func (r *Registry) Register(spec Spec, handler HandlerFunc) error {
	if spec.Name == "" {
		return errors.New("tool name is required")
	}
	if spec.RequiredScope == "" {
		return errors.New("required scope is required")
	}
	if spec.Status == "" {
		spec.Status = "enabled"
	}
	if spec.SensitivityLevel == "" {
		spec.SensitivityLevel = "normal"
	}
	if spec.BackendHandler == "" {
		spec.BackendHandler = spec.Name
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.specs[spec.Name] = spec
	r.handlers[spec.Name] = handler
	return nil
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.specs, name)
	delete(r.handlers, name)
}

func (r *Registry) Get(name string) (Spec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.specs[name]
	return spec, ok
}

func (r *Registry) Handler(name string) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[name]
	return handler, ok
}

func (r *Registry) List() []Spec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Spec, 0, len(r.specs))
	for _, spec := range r.specs {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
