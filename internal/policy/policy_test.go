package policy

import (
	"context"
	"testing"
	"time"
)

func TestStaticEngineDefaultDeny(t *testing.T) {
	engine := NewStaticEngine(DefaultAgents())
	decision, err := engine.Authorize(context.Background(), Request{
		AgentID:       "unknown",
		ToolName:      "get_asset_snapshot",
		RequiredScope: "asset:read",
		RequestedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed {
		t.Fatal("unknown agent should be denied")
	}
}

func TestStaticEngineAllowsRegisteredScope(t *testing.T) {
	engine := NewStaticEngine(DefaultAgents())
	decision, err := engine.Authorize(context.Background(), Request{
		AgentID:       "business-troubleshooter-v1",
		ToolName:      "get_asset_snapshot",
		RequiredScope: "asset:read",
		RequestedAt:   time.Now(),
		MaxLimit:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed {
		t.Fatalf("expected allow, got %q", decision.Reason)
	}
	if decision.MaxLimit != 10 {
		t.Fatalf("expected max limit 10, got %d", decision.MaxLimit)
	}
}
