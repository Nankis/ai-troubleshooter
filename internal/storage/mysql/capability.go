package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/capability"
)

func (s *Store) UpsertBusinessService(ctx context.Context, service capability.BusinessService) (capability.BusinessService, error) {
	now := time.Now()
	if strings.TrimSpace(service.ServiceStatus) == "" {
		service.ServiceStatus = "draft"
	}
	if strings.TrimSpace(service.Environment) == "" {
		service.Environment = "local"
	}
	if strings.TrimSpace(service.AuthType) == "" {
		service.AuthType = "bearer"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tb_troubleshoot_business_service
(service_name, owner_team, environment, base_url, health_check_path, auth_type, secret_ref, service_status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
owner_team = VALUES(owner_team),
environment = VALUES(environment),
base_url = VALUES(base_url),
health_check_path = VALUES(health_check_path),
auth_type = VALUES(auth_type),
secret_ref = VALUES(secret_ref),
service_status = VALUES(service_status),
update_time = VALUES(update_time)`,
		service.ServiceName, nullableString(service.OwnerTeam), service.Environment, nullableString(service.BaseURL),
		nullableString(service.HealthCheckPath), service.AuthType, nullableString(service.SecretRef), service.ServiceStatus, now, now)
	if err != nil {
		return capability.BusinessService{}, err
	}
	return s.getBusinessServiceByName(ctx, service.ServiceName)
}

func (s *Store) CreateMCPServer(ctx context.Context, server capability.MCPServer) (capability.MCPServer, error) {
	now := time.Now()
	if strings.TrimSpace(server.TransportType) == "" {
		server.TransportType = "stdio"
	}
	if server.RequestTimeoutSec <= 0 {
		server.RequestTimeoutSec = 5
	}
	if strings.TrimSpace(server.ServerStatus) == "" {
		server.ServerStatus = "pending_discovery"
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO tb_troubleshoot_mcp_server
(server_name, service_name, transport_type, endpoint_url, command_json, args_json, env_json, protocol_version, request_timeout_seconds, secret_ref, server_status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		server.ServerName, server.ServiceName, server.TransportType, nullableString(server.EndpointURL), nullableString(server.CommandJSON),
		nullableString(server.ArgsJSON), nullableString(server.EnvJSON), nullableString(server.ProtocolVersion), server.RequestTimeoutSec,
		nullableString(server.SecretRef), server.ServerStatus, now, now)
	if err != nil {
		return capability.MCPServer{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return capability.MCPServer{}, err
	}
	return s.getMCPServer(ctx, id)
}

func (s *Store) UpsertToolCapability(ctx context.Context, item capability.ToolCapability) (capability.ToolCapability, error) {
	item = normalizeToolCapability(item)
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tb_troubleshoot_tool_registry
(tool_name, description, service_name, source_type, input_schema_json, output_schema_json, required_scope, backend_handler,
 readonly_base_url, readonly_path, http_method, secret_ref, mcp_server_id, mcp_tool_name, param_map_json, fixed_params_json,
 required_params_json, optional_params_json, max_time_range_minutes, max_limit, timeout_ms, sensitivity_level, safety_status,
 safety_reasons_json, approval_status, validation_status, tool_status, created_by, published_by, published_at, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
description = VALUES(description),
service_name = VALUES(service_name),
source_type = VALUES(source_type),
input_schema_json = VALUES(input_schema_json),
output_schema_json = VALUES(output_schema_json),
required_scope = VALUES(required_scope),
backend_handler = VALUES(backend_handler),
readonly_base_url = VALUES(readonly_base_url),
readonly_path = VALUES(readonly_path),
http_method = VALUES(http_method),
secret_ref = VALUES(secret_ref),
mcp_server_id = VALUES(mcp_server_id),
mcp_tool_name = VALUES(mcp_tool_name),
param_map_json = VALUES(param_map_json),
fixed_params_json = VALUES(fixed_params_json),
required_params_json = VALUES(required_params_json),
optional_params_json = VALUES(optional_params_json),
max_time_range_minutes = VALUES(max_time_range_minutes),
max_limit = VALUES(max_limit),
timeout_ms = VALUES(timeout_ms),
sensitivity_level = VALUES(sensitivity_level),
safety_status = VALUES(safety_status),
safety_reasons_json = VALUES(safety_reasons_json),
approval_status = VALUES(approval_status),
validation_status = VALUES(validation_status),
tool_status = VALUES(tool_status),
created_by = VALUES(created_by),
published_by = VALUES(published_by),
published_at = VALUES(published_at),
update_time = VALUES(update_time)`,
		item.ToolName, item.Description, nullableString(item.ServiceName), item.SourceType, item.InputSchemaJSON, nullableString(item.OutputSchemaJSON),
		item.RequiredScope, item.BackendHandler, nullableString(item.ReadonlyBaseURL), nullableString(item.ReadonlyPath), item.HTTPMethod,
		nullableString(item.SecretRef), nullableInt64(item.MCPServerID), nullableString(item.MCPToolName), nullableString(item.ParamMapJSON),
		nullableString(item.FixedParamsJSON), nullableString(item.RequiredParamsJSON), nullableString(item.OptionalParamsJSON),
		nullableInt(item.MaxTimeRangeMinutes), nullableInt(item.MaxLimit), nullableInt(item.TimeoutMS), item.SensitivityLevel,
		item.SafetyStatus, nullableString(item.SafetyReasonsJSON), item.ApprovalStatus, item.ValidationStatus, item.ToolStatus,
		nullableString(item.CreatedBy), nullableString(item.PublishedBy), nullableTime(item.PublishedAt), now, now)
	if err != nil {
		return capability.ToolCapability{}, err
	}
	return s.getToolCapabilityByName(ctx, item.ToolName)
}

func (s *Store) GetToolCapability(ctx context.Context, id int64) (capability.ToolCapability, error) {
	row := s.db.QueryRowContext(ctx, capabilitySelect()+` WHERE id = ? AND status = 1`, id)
	return scanCapability(row)
}

func (s *Store) ListToolCapabilities(ctx context.Context, filter capability.ToolFilter) ([]capability.ToolCapability, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := capabilitySelect() + ` WHERE status = 1`
	args := []any{}
	if strings.TrimSpace(filter.Status) != "" {
		query += ` AND tool_status = ?`
		args = append(args, filter.Status)
	}
	if strings.TrimSpace(filter.SourceType) != "" {
		query += ` AND source_type = ?`
		args = append(args, filter.SourceType)
	}
	query += ` ORDER BY service_name, tool_name LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []capability.ToolCapability{}
	for rows.Next() {
		item, err := scanCapability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpdateToolCapabilityStatus(ctx context.Context, id int64, status string, publishedBy string) (capability.ToolCapability, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = capability.StatusDraft
	}
	now := time.Now()
	if status == capability.StatusEnabled {
		_, err := s.db.ExecContext(ctx, `
UPDATE tb_troubleshoot_tool_registry
SET tool_status = ?, approval_status = 'approved', published_by = ?, published_at = ?, update_time = ?
WHERE id = ? AND status = 1`,
			status, nullableString(publishedBy), now, now, id)
		if err != nil {
			return capability.ToolCapability{}, err
		}
		return s.GetToolCapability(ctx, id)
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE tb_troubleshoot_tool_registry
SET tool_status = ?, update_time = ?
WHERE id = ? AND status = 1`, status, now, id)
	if err != nil {
		return capability.ToolCapability{}, err
	}
	return s.GetToolCapability(ctx, id)
}

func (s *Store) CreateValidationRun(ctx context.Context, run capability.ValidationRun) (capability.ValidationRun, error) {
	now := time.Now()
	if strings.TrimSpace(run.RunStatus) == "" {
		run.RunStatus = "pending"
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO tb_troubleshoot_tool_validation_run
(tool_id, mcp_server_id, run_type, run_status, input_snapshot_json, output_snapshot_json, error_message, created_by, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableInt64(run.ToolID), nullableInt64(run.MCPServerID), run.RunType, run.RunStatus, nullableString(run.InputSnapshotJSON),
		nullableString(run.OutputSnapshotJSON), nullableString(run.ErrorMessage), nullableString(run.CreatedBy), now, now)
	if err != nil {
		return capability.ValidationRun{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return capability.ValidationRun{}, err
	}
	run.ID = id
	run.CreatedAt = now
	return run, nil
}

func (s *Store) getBusinessServiceByName(ctx context.Context, serviceName string) (capability.BusinessService, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, service_name, owner_team, environment, base_url, health_check_path, auth_type, secret_ref, service_status, create_time, update_time
FROM tb_troubleshoot_business_service WHERE service_name = ? AND status = 1`, serviceName)
	var item capability.BusinessService
	var ownerTeam, baseURL, healthCheckPath, authType, secretRef sql.NullString
	if err := row.Scan(&item.ID, &item.ServiceName, &ownerTeam, &item.Environment, &baseURL, &healthCheckPath, &authType, &secretRef,
		&item.ServiceStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return capability.BusinessService{}, normalizeCapabilityNotFound(err)
	}
	item.OwnerTeam = nullStringValue(ownerTeam)
	item.BaseURL = nullStringValue(baseURL)
	item.HealthCheckPath = nullStringValue(healthCheckPath)
	item.AuthType = nullStringValue(authType)
	item.SecretRef = nullStringValue(secretRef)
	return item, nil
}

func (s *Store) getMCPServer(ctx context.Context, id int64) (capability.MCPServer, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, server_name, service_name, transport_type, endpoint_url, command_json, args_json, env_json, protocol_version,
       request_timeout_seconds, secret_ref, server_status, create_time, update_time
FROM tb_troubleshoot_mcp_server WHERE id = ? AND status = 1`, id)
	var item capability.MCPServer
	var endpointURL, commandJSON, argsJSON, envJSON, protocolVersion, secretRef sql.NullString
	if err := row.Scan(&item.ID, &item.ServerName, &item.ServiceName, &item.TransportType, &endpointURL, &commandJSON, &argsJSON,
		&envJSON, &protocolVersion, &item.RequestTimeoutSec, &secretRef, &item.ServerStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return capability.MCPServer{}, normalizeCapabilityNotFound(err)
	}
	item.EndpointURL = nullStringValue(endpointURL)
	item.CommandJSON = nullStringValue(commandJSON)
	item.ArgsJSON = nullStringValue(argsJSON)
	item.EnvJSON = nullStringValue(envJSON)
	item.ProtocolVersion = nullStringValue(protocolVersion)
	item.SecretRef = nullStringValue(secretRef)
	return item, nil
}

func (s *Store) getToolCapabilityByName(ctx context.Context, name string) (capability.ToolCapability, error) {
	row := s.db.QueryRowContext(ctx, capabilitySelect()+` WHERE tool_name = ? AND status = 1`, name)
	return scanCapability(row)
}

func capabilitySelect() string {
	return `SELECT id, tool_name, description, service_name, source_type, input_schema_json, output_schema_json, required_scope,
backend_handler, readonly_base_url, readonly_path, http_method, secret_ref, mcp_server_id, mcp_tool_name, param_map_json, fixed_params_json,
required_params_json, optional_params_json, max_time_range_minutes, max_limit, timeout_ms, sensitivity_level, safety_status,
safety_reasons_json, approval_status, validation_status, tool_status, created_by, published_by, published_at, create_time, update_time
FROM tb_troubleshoot_tool_registry`
}

func scanCapability(row scanner) (capability.ToolCapability, error) {
	var item capability.ToolCapability
	var serviceName, outputSchema, readonlyBaseURL, readonlyPath, secretRef, mcpToolName, paramMap, fixedParams sql.NullString
	var requiredParams, optionalParams, safetyReasons, createdBy, publishedBy sql.NullString
	var mcpServerID, maxTimeRange, maxLimit, timeoutMS sql.NullInt64
	var publishedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.ToolName, &item.Description, &serviceName, &item.SourceType, &item.InputSchemaJSON,
		&outputSchema, &item.RequiredScope, &item.BackendHandler, &readonlyBaseURL, &readonlyPath, &item.HTTPMethod, &secretRef,
		&mcpServerID, &mcpToolName, &paramMap, &fixedParams, &requiredParams, &optionalParams, &maxTimeRange, &maxLimit, &timeoutMS,
		&item.SensitivityLevel, &item.SafetyStatus, &safetyReasons, &item.ApprovalStatus, &item.ValidationStatus, &item.ToolStatus,
		&createdBy, &publishedBy, &publishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return capability.ToolCapability{}, normalizeCapabilityNotFound(err)
	}
	item.ServiceName = nullStringValue(serviceName)
	item.OutputSchemaJSON = nullStringValue(outputSchema)
	item.ReadonlyBaseURL = nullStringValue(readonlyBaseURL)
	item.ReadonlyPath = nullStringValue(readonlyPath)
	item.SecretRef = nullStringValue(secretRef)
	if mcpServerID.Valid {
		item.MCPServerID = mcpServerID.Int64
	}
	item.MCPToolName = nullStringValue(mcpToolName)
	item.ParamMapJSON = nullStringValue(paramMap)
	item.FixedParamsJSON = nullStringValue(fixedParams)
	item.RequiredParamsJSON = nullStringValue(requiredParams)
	item.OptionalParamsJSON = nullStringValue(optionalParams)
	if maxTimeRange.Valid {
		item.MaxTimeRangeMinutes = int(maxTimeRange.Int64)
	}
	if maxLimit.Valid {
		item.MaxLimit = int(maxLimit.Int64)
	}
	if timeoutMS.Valid {
		item.TimeoutMS = int(timeoutMS.Int64)
	}
	item.SafetyReasonsJSON = nullStringValue(safetyReasons)
	item.CreatedBy = nullStringValue(createdBy)
	item.PublishedBy = nullStringValue(publishedBy)
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	return item, nil
}

func normalizeToolCapability(item capability.ToolCapability) capability.ToolCapability {
	if strings.TrimSpace(item.Description) == "" {
		item.Description = item.ToolName
	}
	if strings.TrimSpace(item.SourceType) == "" {
		item.SourceType = capability.SourceHTTPAdapter
	}
	if strings.TrimSpace(item.InputSchemaJSON) == "" {
		item.InputSchemaJSON = `{"type":"object"}`
	}
	if strings.TrimSpace(item.OutputSchemaJSON) == "" {
		item.OutputSchemaJSON = `{"type":"object"}`
	}
	if strings.TrimSpace(item.HTTPMethod) == "" {
		item.HTTPMethod = "POST"
	}
	if strings.TrimSpace(item.SensitivityLevel) == "" {
		item.SensitivityLevel = "normal"
	}
	if strings.TrimSpace(item.SafetyStatus) == "" {
		item.SafetyStatus = capability.SafetyNeedsReview
	}
	if strings.TrimSpace(item.ApprovalStatus) == "" {
		item.ApprovalStatus = "pending"
	}
	if strings.TrimSpace(item.ValidationStatus) == "" {
		item.ValidationStatus = "not_run"
	}
	if strings.TrimSpace(item.ToolStatus) == "" {
		item.ToolStatus = capability.StatusDraft
	}
	if item.TimeoutMS <= 0 {
		item.TimeoutMS = 5000
	}
	return item
}

func normalizeCapabilityNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return capability.ErrNotFound
	}
	return err
}
