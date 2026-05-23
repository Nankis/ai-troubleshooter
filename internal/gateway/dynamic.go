package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/capability"
	"github.com/Nankis/ai-troubleshooter/internal/connectors"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
)

type CapabilityReloader struct {
	mu         sync.Mutex
	registry   *tool.Registry
	store      capability.Store
	registered map[string]bool
}

func NewCapabilityReloader(registry *tool.Registry, store capability.Store) *CapabilityReloader {
	return &CapabilityReloader{
		registry:   registry,
		store:      store,
		registered: map[string]bool{},
	}
}

func (r *CapabilityReloader) Reload(ctx context.Context) error {
	if r == nil || r.registry == nil || r.store == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.store.ListToolCapabilities(ctx, capability.ToolFilter{Status: capability.StatusEnabled, Limit: 500})
	if err != nil {
		return err
	}
	nextRegistered := map[string]bool{}
	for _, item := range items {
		if item.SourceType != capability.SourceHTTPAdapter && item.SourceType != capability.SourceMCP {
			continue
		}
		if err := RegisterDynamicCapability(r.registry, item); err != nil {
			return fmt.Errorf("register dynamic capability %s: %w", item.ToolName, err)
		}
		nextRegistered[item.ToolName] = true
	}
	for name := range r.registered {
		if !nextRegistered[name] {
			r.registry.Unregister(name)
		}
	}
	r.registered = nextRegistered
	return nil
}

func RegisterDynamicCapability(reg *tool.Registry, item capability.ToolCapability) error {
	if reg == nil {
		return fmt.Errorf("registry is required")
	}
	if item.ToolStatus != capability.StatusEnabled {
		return fmt.Errorf("tool is not enabled")
	}
	if item.SafetyStatus != capability.SafetyReadonlyCandidate {
		return fmt.Errorf("tool safety status is %s", item.SafetyStatus)
	}
	if strings.TrimSpace(item.ReadonlyBaseURL) == "" || strings.TrimSpace(item.ReadonlyPath) == "" {
		return fmt.Errorf("readonly_base_url and readonly_path are required")
	}
	inputSchema, err := jsonObject(item.InputSchemaJSON, map[string]any{"type": "object"})
	if err != nil {
		return fmt.Errorf("input schema: %w", err)
	}
	outputSchema, err := jsonObject(item.OutputSchemaJSON, map[string]any{"type": "object"})
	if err != nil {
		return fmt.Errorf("output schema: %w", err)
	}
	spec := tool.Spec{
		Name:                item.ToolName,
		Description:         item.Description,
		InputSchema:         inputSchema,
		OutputSchema:        outputSchema,
		RequiredScope:       item.RequiredScope,
		BackendHandler:      item.BackendHandler,
		MaxTimeRangeMinutes: item.MaxTimeRangeMinutes,
		MaxLimit:            item.MaxLimit,
		SensitivityLevel:    item.SensitivityLevel,
		Status:              capability.StatusEnabled,
	}
	if spec.BackendHandler == "" {
		spec.BackendHandler = item.SourceType + "." + item.ToolName
	}
	timeout := time.Duration(item.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	apiKey := ""
	if item.SecretRef != "" {
		apiKey = os.Getenv(item.SecretRef)
	}
	connector, err := connectors.NewGenericReadonlyConnector(connectors.HTTPConfig{
		BaseURL: item.ReadonlyBaseURL,
		APIKey:  apiKey,
		Timeout: timeout,
	})
	if err != nil {
		return err
	}
	requiredParams := jsonStringList(item.RequiredParamsJSON)
	return reg.Register(spec, func(ctx context.Context, req tool.InvocationRequest) (tool.InvocationResponse, error) {
		if err := requireDynamicParams(req.Arguments, requiredParams); err != nil {
			return tool.InvocationResponse{}, err
		}
		args := dynamicArgs(req.Arguments, item.ParamMapJSON, item.FixedParamsJSON)
		result, err := connector.Invoke(ctx, item.ReadonlyPath, args)
		return tool.InvocationResponse{
			Status:   "success",
			Data:     result.Data,
			Summary:  fmt.Sprintf("%s returned from %s", item.ToolName, firstNonEmpty(result.Source, item.ServiceName, item.ReadonlyBaseURL)),
			Warnings: result.Warnings,
		}, err
	})
}

func requireDynamicParams(args map[string]any, required []string) error {
	for _, name := range required {
		if strings.TrimSpace(name) == "" {
			continue
		}
		value, ok := args[name]
		if !ok || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func dynamicArgs(args map[string]any, paramMapJSON string, fixedParamsJSON string) map[string]any {
	out := map[string]any{}
	for key, value := range args {
		out[key] = value
	}
	paramMap := map[string]string{}
	if strings.TrimSpace(paramMapJSON) != "" {
		_ = json.Unmarshal([]byte(paramMapJSON), &paramMap)
	}
	for sourceName, targetName := range paramMap {
		if targetName == "" {
			continue
		}
		if value, ok := args[sourceName]; ok {
			out[targetName] = value
		}
	}
	fixed := map[string]any{}
	if strings.TrimSpace(fixedParamsJSON) != "" {
		_ = json.Unmarshal([]byte(fixedParamsJSON), &fixed)
	}
	for key, value := range fixed {
		out[key] = value
	}
	return out
}

func jsonObject(raw string, fallback map[string]any) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func jsonStringList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := []string{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
