package capability

import (
	"context"
	"strings"
	"testing"
)

func TestImportHTTPManifestCreatesReadonlyDraft(t *testing.T) {
	store := NewMemoryStore()
	result, err := Import(context.Background(), store, ImportRequest{
		CreatedBy: "test",
		RawConfig: `{
		  "service": {"service_name": "health-food", "base_url": "http://127.0.0.1:19081", "auth": {"token_env": "CONNECTOR_API_KEY"}},
		  "capabilities": [{
		    "tool_name": "get_health_food_ai_quota",
		    "description": "查询用户 AI token 配额状态",
		    "scope": "health_food:ai_quota:read",
		    "method": "POST",
		    "path": "/v1/readonly/health-food/ai/quota",
		    "required_params": ["uid"]
		  }]
		}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Capabilities) != 1 {
		t.Fatalf("expected one capability, got %+v", result)
	}
	item := result.Capabilities[0]
	if item.ToolStatus != StatusDraft || item.SafetyStatus != SafetyReadonlyCandidate || item.RequiredScope != "health_food:ai_quota:read" {
		t.Fatalf("unexpected capability: %+v", item)
	}
	if item.SecretRef != "CONNECTOR_API_KEY" {
		t.Fatalf("expected secret ref from manifest, got %q", item.SecretRef)
	}
}

func TestImportHTTPManifestRejectsDangerousCapability(t *testing.T) {
	store := NewMemoryStore()
	result, err := Import(context.Background(), store, ImportRequest{
		RawConfig: `{
		  "service": {"service_name": "ops", "base_url": "http://127.0.0.1:19081"},
		  "capabilities": [{
		    "tool_name": "delete_user_cache",
		    "description": "删除用户缓存",
		    "method": "POST",
		    "path": "/v1/write/cache/delete",
		    "required_params": ["uid"]
		  }]
		}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	item := result.Capabilities[0]
	if item.ToolStatus != StatusRejected || item.SafetyStatus != SafetyRejected {
		t.Fatalf("expected rejected capability, got %+v", item)
	}
	if !strings.Contains(item.SafetyReasonsJSON, "dangerous action keyword") {
		t.Fatalf("expected safety reason, got %s", item.SafetyReasonsJSON)
	}
}

func TestImportClaudeMCPDoesNotPublishTools(t *testing.T) {
	store := NewMemoryStore()
	result, err := Import(context.Background(), store, ImportRequest{
		RawConfig: `{
		  "mcpServers": {
		    "dms": {"command": "uvx", "args": ["alibabacloud-dms-mcp-server@latest"]}
		  }
		}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MCPServers) != 1 || len(result.ValidationRuns) != 1 {
		t.Fatalf("expected server and validation run, got %+v", result)
	}
	if len(result.Capabilities) != 0 {
		t.Fatalf("mcpServers import must not publish tools, got %+v", result.Capabilities)
	}
}

func TestImportMCPRoutesRequiresReadonlyPath(t *testing.T) {
	store := NewMemoryStore()
	result, err := Import(context.Background(), store, ImportRequest{
		BaseURL: "http://127.0.0.1:19085",
		RawConfig: `{
		  "service_name": "dms",
		  "server": {"command": ["uvx", "dms"]},
		  "routes": [
		    {"path": "/v1/readonly/db/tables/list", "tool_name": "listTables", "required_params": ["database_id"]},
		    {"path": "/v1/write/db/sql/execute", "tool_name": "executeSQL", "required_params": ["sql"]}
		  ]
		}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Capabilities) != 2 {
		t.Fatalf("expected two imported capabilities, got %+v", result.Capabilities)
	}
	if result.Capabilities[0].SafetyStatus != SafetyReadonlyCandidate {
		t.Fatalf("expected readonly first capability, got %+v", result.Capabilities[0])
	}
	if result.Capabilities[1].SafetyStatus != SafetyRejected || result.Capabilities[1].ToolStatus != StatusRejected {
		t.Fatalf("expected rejected write route, got %+v", result.Capabilities[1])
	}
}

func TestImportYAMLManifest(t *testing.T) {
	store := NewMemoryStore()
	result, err := Import(context.Background(), store, ImportRequest{
		RawConfig: `
service:
  service_name: health-food
  base_url: http://127.0.0.1:19081
capabilities:
  - tool_name: get_health_food_user_profile
    description: 查询用户资料
    path: /v1/readonly/health-food/user/profile
    required_params: [uid]
`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Capabilities) != 1 || result.Capabilities[0].RequiredScope == "" {
		t.Fatalf("expected yaml capability with default scope, got %+v", result.Capabilities)
	}
}
