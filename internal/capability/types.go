package capability

import (
	"context"
	"time"
)

const (
	SourceHTTPAdapter = "http_adapter"
	SourceMCP         = "mcp"

	StatusDraft    = "draft"
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
	StatusRejected = "rejected"

	SafetyReadonlyCandidate = "readonly_candidate"
	SafetyNeedsReview       = "needs_review"
	SafetyRejected          = "rejected"
)

type BusinessService struct {
	ID              int64     `json:"id"`
	ServiceName     string    `json:"service_name"`
	OwnerTeam       string    `json:"owner_team,omitempty"`
	Environment     string    `json:"environment,omitempty"`
	BaseURL         string    `json:"base_url,omitempty"`
	HealthCheckPath string    `json:"health_check_path,omitempty"`
	AuthType        string    `json:"auth_type,omitempty"`
	SecretRef       string    `json:"secret_ref,omitempty"`
	ServiceStatus   string    `json:"service_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MCPServer struct {
	ID                int64     `json:"id"`
	ServerName        string    `json:"server_name"`
	ServiceName       string    `json:"service_name"`
	TransportType     string    `json:"transport_type"`
	EndpointURL       string    `json:"endpoint_url,omitempty"`
	CommandJSON       string    `json:"command_json,omitempty"`
	ArgsJSON          string    `json:"args_json,omitempty"`
	EnvJSON           string    `json:"env_json,omitempty"`
	ProtocolVersion   string    `json:"protocol_version,omitempty"`
	RequestTimeoutSec int       `json:"request_timeout_seconds"`
	SecretRef         string    `json:"secret_ref,omitempty"`
	ServerStatus      string    `json:"server_status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ToolCapability struct {
	ID                  int64      `json:"id"`
	ToolName            string     `json:"tool_name"`
	Description         string     `json:"description"`
	ServiceName         string     `json:"service_name"`
	SourceType          string     `json:"source_type"`
	InputSchemaJSON     string     `json:"input_schema_json"`
	OutputSchemaJSON    string     `json:"output_schema_json,omitempty"`
	RequiredScope       string     `json:"required_scope"`
	BackendHandler      string     `json:"backend_handler"`
	ReadonlyBaseURL     string     `json:"readonly_base_url,omitempty"`
	ReadonlyPath        string     `json:"readonly_path,omitempty"`
	HTTPMethod          string     `json:"http_method"`
	SecretRef           string     `json:"secret_ref,omitempty"`
	MCPServerID         int64      `json:"mcp_server_id,omitempty"`
	MCPToolName         string     `json:"mcp_tool_name,omitempty"`
	ParamMapJSON        string     `json:"param_map_json,omitempty"`
	FixedParamsJSON     string     `json:"fixed_params_json,omitempty"`
	RequiredParamsJSON  string     `json:"required_params_json,omitempty"`
	OptionalParamsJSON  string     `json:"optional_params_json,omitempty"`
	MaxTimeRangeMinutes int        `json:"max_time_range_minutes,omitempty"`
	MaxLimit            int        `json:"max_limit,omitempty"`
	TimeoutMS           int        `json:"timeout_ms,omitempty"`
	SensitivityLevel    string     `json:"sensitivity_level"`
	SafetyStatus        string     `json:"safety_status"`
	SafetyReasonsJSON   string     `json:"safety_reasons_json,omitempty"`
	ApprovalStatus      string     `json:"approval_status"`
	ValidationStatus    string     `json:"validation_status"`
	ToolStatus          string     `json:"tool_status"`
	CreatedBy           string     `json:"created_by,omitempty"`
	PublishedBy         string     `json:"published_by,omitempty"`
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type ValidationRun struct {
	ID                 int64     `json:"id"`
	ToolID             int64     `json:"tool_id,omitempty"`
	MCPServerID        int64     `json:"mcp_server_id,omitempty"`
	RunType            string    `json:"run_type"`
	RunStatus          string    `json:"run_status"`
	InputSnapshotJSON  string    `json:"input_snapshot_json,omitempty"`
	OutputSnapshotJSON string    `json:"output_snapshot_json,omitempty"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	CreatedBy          string    `json:"created_by,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type ToolFilter struct {
	Status     string
	SourceType string
	Limit      int
}

type Store interface {
	UpsertBusinessService(ctx context.Context, service BusinessService) (BusinessService, error)
	CreateMCPServer(ctx context.Context, server MCPServer) (MCPServer, error)
	UpsertToolCapability(ctx context.Context, item ToolCapability) (ToolCapability, error)
	GetToolCapability(ctx context.Context, id int64) (ToolCapability, error)
	ListToolCapabilities(ctx context.Context, filter ToolFilter) ([]ToolCapability, error)
	UpdateToolCapabilityStatus(ctx context.Context, id int64, status string, publishedBy string) (ToolCapability, error)
	CreateValidationRun(ctx context.Context, run ValidationRun) (ValidationRun, error)
}

type ImportRequest struct {
	Kind        string `json:"kind"`
	ServiceName string `json:"service_name"`
	BaseURL     string `json:"base_url"`
	SecretRef   string `json:"secret_ref"`
	CreatedBy   string `json:"created_by"`
	RawConfig   string `json:"raw_config"`
}

type ImportResult struct {
	Services       []BusinessService `json:"services"`
	MCPServers     []MCPServer       `json:"mcp_servers"`
	Capabilities   []ToolCapability  `json:"capabilities"`
	ValidationRuns []ValidationRun   `json:"validation_runs"`
	Warnings       []string          `json:"warnings"`
}
