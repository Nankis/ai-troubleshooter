package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	dangerWords = []string{"delete", "remove", "drop", "truncate", "insert", "update", "upsert", "create", "write", "execute", "exec", "run", "shell", "command", "deploy", "restart", "approve", "refund", "pay", "transfer", "send", "grant", "revoke", "disable", "enable"}
	readWords   = []string{"get", "list", "search", "query", "read", "find", "describe", "profile", "status", "quota", "logs", "kline", "snapshot", "events", "compare", "similar", "metadata", "summary"}
)

func Import(ctx context.Context, store Store, req ImportRequest) (ImportResult, error) {
	if store == nil {
		return ImportResult{}, fmt.Errorf("capability store is required")
	}
	raw := strings.TrimSpace(req.RawConfig)
	if raw == "" {
		return ImportResult{}, fmt.Errorf("raw_config is required")
	}
	data, err := parseConfig(raw)
	if err != nil {
		return ImportResult{}, err
	}
	switch {
	case data["mcpServers"] != nil:
		return importClaudeMCP(ctx, store, req, data)
	case data["routes"] != nil:
		return importMCPRoutes(ctx, store, req, data)
	case data["capabilities"] != nil:
		return importHTTPManifest(ctx, store, req, data)
	default:
		return ImportResult{}, fmt.Errorf("unsupported config: expected mcpServers, routes, or capabilities")
	}
}

func importHTTPManifest(ctx context.Context, store Store, req ImportRequest, data map[string]any) (ImportResult, error) {
	serviceMap, _ := data["service"].(map[string]any)
	serviceName := firstNonEmpty(req.ServiceName, str(serviceMap["service_name"]), str(data["service_name"]))
	if serviceName == "" {
		return ImportResult{}, fmt.Errorf("service_name is required")
	}
	baseURL := firstNonEmpty(req.BaseURL, str(serviceMap["base_url"]), str(data["base_url"]))
	if baseURL != "" {
		if err := validateBaseURL(baseURL); err != nil {
			return ImportResult{}, err
		}
	}
	secretRef := firstNonEmpty(req.SecretRef, str(nestedMapString(serviceMap, "auth", "secret_ref")), str(nestedMapString(serviceMap, "auth", "token_env")))
	service, err := store.UpsertBusinessService(ctx, BusinessService{
		ServiceName:     serviceName,
		OwnerTeam:       str(serviceMap["owner_team"]),
		Environment:     firstNonEmpty(str(serviceMap["environment"]), "local"),
		BaseURL:         baseURL,
		HealthCheckPath: str(nestedMapString(serviceMap, "health_check", "path")),
		AuthType:        firstNonEmpty(str(nestedMapString(serviceMap, "auth", "type")), "bearer"),
		SecretRef:       secretRef,
		ServiceStatus:   "enabled",
	})
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{Services: []BusinessService{service}}
	for _, rawCapability := range list(data["capabilities"]) {
		item, ok := rawCapability.(map[string]any)
		if !ok {
			continue
		}
		toolName := normalizeToolName(str(item["tool_name"]))
		if toolName == "" {
			result.Warnings = append(result.Warnings, "skip capability without tool_name")
			continue
		}
		method := strings.ToUpper(firstNonEmpty(str(item["method"]), "POST"))
		path := normalizePath(str(item["path"]))
		inputSchema := schemaFromParams(stringsList(item["required_params"]), stringsList(item["optional_params"]))
		scope := firstNonEmpty(str(item["scope"]), scopeFromToolName(toolName))
		safetyStatus, reasons := assessSafety(toolName, str(item["description"]), method, path, scope)
		if !strings.Contains(path, "/readonly/") {
			safetyStatus = SafetyRejected
			reasons = append(reasons, "readonly http path must be under /readonly/")
		}
		status := StatusDraft
		approval := "pending"
		if safetyStatus == SafetyRejected {
			status = StatusRejected
			approval = "rejected"
		}
		capability, err := store.UpsertToolCapability(ctx, ToolCapability{
			ToolName:            toolName,
			Description:         str(item["description"]),
			ServiceName:         service.ServiceName,
			SourceType:          SourceHTTPAdapter,
			InputSchemaJSON:     mustJSON(inputSchema),
			OutputSchemaJSON:    mustJSON(map[string]any{"type": "object"}),
			RequiredScope:       scope,
			BackendHandler:      "dynamic_http." + toolName,
			ReadonlyBaseURL:     service.BaseURL,
			ReadonlyPath:        path,
			HTTPMethod:          method,
			SecretRef:           service.SecretRef,
			RequiredParamsJSON:  mustJSON(stringsList(item["required_params"])),
			OptionalParamsJSON:  mustJSON(stringsList(item["optional_params"])),
			MaxTimeRangeMinutes: intFromAny(item["max_time_range_minutes"]),
			MaxLimit:            intFromAny(item["max_limit"]),
			TimeoutMS:           intDefaultAny(item["timeout_ms"], 5000),
			SensitivityLevel:    firstNonEmpty(str(item["sensitivity_level"]), "normal"),
			SafetyStatus:        safetyStatus,
			SafetyReasonsJSON:   mustJSON(reasons),
			ApprovalStatus:      approval,
			ValidationStatus:    "not_run",
			ToolStatus:          status,
			CreatedBy:           req.CreatedBy,
		})
		if err != nil {
			return result, err
		}
		result.Capabilities = append(result.Capabilities, capability)
	}
	return result, nil
}

func importMCPRoutes(ctx context.Context, store Store, req ImportRequest, data map[string]any) (ImportResult, error) {
	serverMap, _ := data["server"].(map[string]any)
	serviceName := firstNonEmpty(req.ServiceName, str(data["service_name"]), sourceServiceName(data), "mcp-service")
	baseURL := firstNonEmpty(req.BaseURL, str(data["base_url"]), str(serverMap["base_url"]))
	if baseURL != "" {
		if err := validateBaseURL(baseURL); err != nil {
			return ImportResult{}, err
		}
	}
	service, err := store.UpsertBusinessService(ctx, BusinessService{
		ServiceName:   serviceName,
		Environment:   "local",
		BaseURL:       baseURL,
		AuthType:      "bearer",
		SecretRef:     req.SecretRef,
		ServiceStatus: "enabled",
	})
	if err != nil {
		return ImportResult{}, err
	}
	mcpServer, err := store.CreateMCPServer(ctx, MCPServer{
		ServerName:        firstNonEmpty(str(data["server_name"]), serviceName+"-mcp"),
		ServiceName:       service.ServiceName,
		TransportType:     firstNonEmpty(str(serverMap["transport"]), "stdio"),
		EndpointURL:       str(serverMap["url"]),
		CommandJSON:       mustJSON(serverMap["command"]),
		ArgsJSON:          mustJSON(serverMap["args"]),
		EnvJSON:           mustJSON(serverMap["env"]),
		ProtocolVersion:   str(serverMap["protocol_version"]),
		RequestTimeoutSec: intDefaultAny(serverMap["request_timeout_seconds"], 5),
		SecretRef:         req.SecretRef,
		ServerStatus:      "pending_discovery",
	})
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{Services: []BusinessService{service}, MCPServers: []MCPServer{mcpServer}}
	for _, rawRoute := range list(data["routes"]) {
		route, ok := rawRoute.(map[string]any)
		if !ok {
			continue
		}
		mcpToolName := str(route["tool_name"])
		toolName := normalizeToolName(firstNonEmpty(str(route["gateway_tool_name"]), gatewayToolName(mcpToolName, str(route["path"]))))
		method := "POST"
		path := normalizePath(str(route["path"]))
		desc := firstNonEmpty(str(route["description"]), fmt.Sprintf("MCP readonly route %s", mcpToolName))
		required := stringsList(route["required_params"])
		inputSchema := schemaFromParams(required, []string{})
		safetyStatus, reasons := assessSafety(toolName, desc, method, path, str(route["scope"]))
		if !strings.Contains(path, "/readonly/") {
			safetyStatus = SafetyRejected
			reasons = append(reasons, "mcp route path must be under /readonly/")
		}
		status := StatusDraft
		approval := "pending"
		if safetyStatus == SafetyRejected {
			status = StatusRejected
			approval = "rejected"
		}
		capability, err := store.UpsertToolCapability(ctx, ToolCapability{
			ToolName:           toolName,
			Description:        desc,
			ServiceName:        service.ServiceName,
			SourceType:         SourceMCP,
			InputSchemaJSON:    mustJSON(inputSchema),
			OutputSchemaJSON:   mustJSON(map[string]any{"type": "object"}),
			RequiredScope:      firstNonEmpty(str(route["scope"]), scopeFromToolName(toolName)),
			BackendHandler:     "dynamic_mcp." + toolName,
			ReadonlyBaseURL:    service.BaseURL,
			ReadonlyPath:       path,
			HTTPMethod:         method,
			SecretRef:          service.SecretRef,
			MCPServerID:        mcpServer.ID,
			MCPToolName:        mcpToolName,
			ParamMapJSON:       mustJSON(route["param_map"]),
			FixedParamsJSON:    mustJSON(route["fixed_params"]),
			RequiredParamsJSON: mustJSON(required),
			TimeoutMS:          intDefaultAny(route["timeout_ms"], 5000),
			SensitivityLevel:   firstNonEmpty(str(route["sensitivity_level"]), "normal"),
			SafetyStatus:       safetyStatus,
			SafetyReasonsJSON:  mustJSON(reasons),
			ApprovalStatus:     approval,
			ValidationStatus:   "not_run",
			ToolStatus:         status,
			CreatedBy:          req.CreatedBy,
		})
		if err != nil {
			return result, err
		}
		result.Capabilities = append(result.Capabilities, capability)
	}
	return result, nil
}

func importClaudeMCP(ctx context.Context, store Store, req ImportRequest, data map[string]any) (ImportResult, error) {
	servers, _ := data["mcpServers"].(map[string]any)
	if len(servers) == 0 {
		return ImportResult{}, fmt.Errorf("mcpServers is empty")
	}
	result := ImportResult{}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rawServer, _ := servers[name].(map[string]any)
		serviceName := firstNonEmpty(req.ServiceName, sanitizeServiceName(name))
		service, err := store.UpsertBusinessService(ctx, BusinessService{
			ServiceName:   serviceName,
			Environment:   "local",
			BaseURL:       req.BaseURL,
			AuthType:      "bearer",
			SecretRef:     req.SecretRef,
			ServiceStatus: "draft",
		})
		if err != nil {
			return result, err
		}
		server, err := store.CreateMCPServer(ctx, MCPServer{
			ServerName:        name,
			ServiceName:       service.ServiceName,
			TransportType:     firstNonEmpty(str(rawServer["transport"]), "stdio"),
			EndpointURL:       str(rawServer["url"]),
			CommandJSON:       mustJSON(rawServer["command"]),
			ArgsJSON:          mustJSON(rawServer["args"]),
			EnvJSON:           mustJSON(rawServer["env"]),
			ProtocolVersion:   str(rawServer["protocol_version"]),
			RequestTimeoutSec: 5,
			SecretRef:         req.SecretRef,
			ServerStatus:      "pending_discovery",
		})
		if err != nil {
			return result, err
		}
		run, _ := store.CreateValidationRun(ctx, ValidationRun{
			MCPServerID:        server.ID,
			RunType:            "mcp_config_import",
			RunStatus:          "pending_discovery",
			InputSnapshotJSON:  mustJSON(map[string]any{"server_name": name, "transport": server.TransportType}),
			OutputSnapshotJSON: mustJSON(map[string]any{"message": "mcpServers import does not execute arbitrary command; add readonly routes before publishing tools"}),
			CreatedBy:          req.CreatedBy,
		})
		result.Services = append(result.Services, service)
		result.MCPServers = append(result.MCPServers, server)
		result.ValidationRuns = append(result.ValidationRuns, run)
	}
	result.Warnings = append(result.Warnings, "mcpServers imported as pending discovery; no tool is published until readonly routes are allowlisted")
	return result, nil
}

func assessSafety(toolName, description, method, path, scope string) (string, []string) {
	signal := strings.ToLower(strings.Join([]string{toolName, description, path, scope}, " "))
	reasons := []string{}
	if method != "" && method != "GET" && method != "POST" {
		return SafetyRejected, []string{"method " + method + " is not allowed for readonly capability"}
	}
	for _, word := range dangerWords {
		if regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(word) + `([^a-z0-9]|$)`).MatchString(signal) {
			return SafetyRejected, []string{"dangerous action keyword: " + word}
		}
	}
	if strings.Contains(signal, "sql") && (strings.Contains(signal, "execute") || strings.Contains(signal, "script") || !strings.Contains(signal, "readonly")) {
		return SafetyRejected, []string{"sql capability must be named readonly query and cannot expose arbitrary execution"}
	}
	for _, word := range readWords {
		if strings.Contains(signal, word) {
			reasons = append(reasons, "readonly signal: "+word)
			return SafetyReadonlyCandidate, reasons
		}
	}
	return SafetyNeedsReview, []string{"no clear readonly signal; manual review required"}
}

func schemaFromParams(required []string, optional []string) map[string]any {
	props := map[string]any{}
	for _, name := range append(append([]string{}, required...), optional...) {
		if name == "" {
			continue
		}
		props[name] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "required": required, "properties": props}
}

func validateBaseURL(value string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("invalid base_url %q: %w", value, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("base_url must use http or https")
	}
	return nil
}

func gatewayToolName(mcpToolName string, path string) string {
	name := normalizeToolName(mcpToolName)
	if name == "" {
		name = normalizeToolName(strings.Trim(path, "/"))
	}
	if strings.HasPrefix(name, "get_") || strings.HasPrefix(name, "search_") || strings.HasPrefix(name, "list_") || strings.HasPrefix(name, "query_") || strings.HasPrefix(name, "read_") {
		return name
	}
	if strings.Contains(name, "log") && strings.Contains(name, "search") {
		return "search_" + name
	}
	return "get_" + name
}

func scopeFromToolName(name string) string {
	service := "dynamic"
	if parts := strings.Split(name, "_"); len(parts) > 1 {
		service = parts[1]
	}
	return service + ":read"
}

func normalizeToolName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	return value
}

func sanitizeServiceName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "mcp-service"
	}
	return value
}

func sourceServiceName(data map[string]any) string {
	for _, raw := range list(data["routes"]) {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		source := str(item["source"])
		if source == "" {
			continue
		}
		return sanitizeServiceName(strings.Split(source, "/")[0])
	}
	return ""
}

func list(value any) []any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	return items
}

func stringsList(value any) []string {
	items := []string{}
	for _, item := range list(value) {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" {
			items = append(items, text)
		}
	}
	return items
}

func str(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func nestedMapString(parent map[string]any, key string, child string) any {
	raw, _ := parent[key].(map[string]any)
	if raw == nil {
		return nil
	}
	return raw[child]
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var out int
		_, _ = fmt.Sscanf(v, "%d", &out)
		return out
	default:
		return 0
	}
}

func intDefaultAny(value any, def int) int {
	if got := intFromAny(value); got > 0 {
		return got
	}
	return def
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mustJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseConfig(raw string) (map[string]any, error) {
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		return data, nil
	}
	if err := yaml.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("raw_config must be JSON or YAML: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("raw_config is empty")
	}
	return data, nil
}

func normalizePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}
