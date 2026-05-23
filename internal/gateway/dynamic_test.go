package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nankis/ai-troubleshooter/internal/audit"
	"github.com/Nankis/ai-troubleshooter/internal/capability"
	"github.com/Nankis/ai-troubleshooter/internal/policy"
	"github.com/Nankis/ai-troubleshooter/internal/tool"
)

func TestRegisterDynamicCapabilityInvokesReadonlyAdapter(t *testing.T) {
	var gotAuth string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/readonly/demo/status" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id":      "req_test",
			"source":          "demo-service",
			"queried_at":      time.Now().Format(time.RFC3339),
			"data_updated_at": time.Now().Format(time.RFC3339),
			"version":         "test-v1",
			"data":            map[string]any{"uid": "u1", "ok": true},
			"warnings":        []string{"sample"},
		})
	}))
	defer server.Close()
	t.Setenv("DEMO_CONNECTOR_TOKEN", "secret-token")

	reg := tool.NewRegistry()
	err := RegisterDynamicCapability(reg, capability.ToolCapability{
		ToolName:           "get_demo_status",
		Description:        "查询 demo 状态",
		ServiceName:        "demo-service",
		SourceType:         capability.SourceHTTPAdapter,
		InputSchemaJSON:    `{"type":"object","required":["uid"],"properties":{"uid":{"type":"string"}}}`,
		OutputSchemaJSON:   `{"type":"object"}`,
		RequiredScope:      "dynamic:read",
		BackendHandler:     "dynamic_http.get_demo_status",
		ReadonlyBaseURL:    server.URL,
		ReadonlyPath:       "/v1/readonly/demo/status",
		HTTPMethod:         "POST",
		SecretRef:          "DEMO_CONNECTOR_TOKEN",
		RequiredParamsJSON: `["uid"]`,
		TimeoutMS:          1000,
		SensitivityLevel:   "normal",
		SafetyStatus:       capability.SafetyReadonlyCandidate,
		ToolStatus:         capability.StatusEnabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	gw := New(reg, policy.NewStaticEngine(policy.DefaultAgents()), audit.NewMemorySink(), time.Second)
	resp, err := gw.Invoke(context.Background(), tool.InvocationRequest{
		CaseID:   "case_test",
		AgentID:  policy.DefaultAgentID,
		ChatID:   "web-local",
		ToolName: "get_demo_status",
		Arguments: map[string]any{
			"uid": "u1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "success" || resp.Summary == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("expected bearer token, got %q", gotAuth)
	}
	params, _ := gotPayload["params"].(map[string]any)
	if params["uid"] != "u1" {
		t.Fatalf("expected uid to be forwarded, got %+v", gotPayload)
	}
}

func TestRegisterDynamicCapabilityRequiresReadonlySafety(t *testing.T) {
	err := RegisterDynamicCapability(tool.NewRegistry(), capability.ToolCapability{
		ToolName:        "delete_demo",
		RequiredScope:   "dynamic:read",
		ReadonlyBaseURL: "http://127.0.0.1:1",
		ReadonlyPath:    "/v1/write/demo",
		SafetyStatus:    capability.SafetyRejected,
		ToolStatus:      capability.StatusEnabled,
	})
	if err == nil {
		t.Fatal("expected rejected safety error")
	}
}
