package audit

import (
	"context"
	"log"
	"sync"
	"time"
)

type Record struct {
	ToolCallID       string    `json:"tool_call_id"`
	CaseID           string    `json:"case_id"`
	InvestigationID  string    `json:"investigation_id,omitempty"`
	AgentID          string    `json:"agent_id"`
	LarkUserID       string    `json:"lark_user_id,omitempty"`
	ToolName         string    `json:"tool_name"`
	RequiredScope    string    `json:"required_scope,omitempty"`
	ArgumentsSummary string    `json:"arguments_summary,omitempty"`
	PolicyDecision   string    `json:"policy_decision"`
	DenyReason       string    `json:"deny_reason,omitempty"`
	QueryID          string    `json:"query_id,omitempty"`
	ResultCount      int       `json:"result_count,omitempty"`
	LatencyMS        int64     `json:"latency_ms,omitempty"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type Sink interface {
	Record(ctx context.Context, record Record) error
}

type MemorySink struct {
	mu      sync.RWMutex
	records []Record
}

func NewMemorySink() *MemorySink {
	return &MemorySink{}
}

func (s *MemorySink) Record(ctx context.Context, record Record) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	s.records = append(s.records, record)
	log.Printf("tool_audit tool_call_id=%s tool=%s decision=%s latency_ms=%d", record.ToolCallID, record.ToolName, record.PolicyDecision, record.LatencyMS)
	return nil
}

func (s *MemorySink) Records() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Record(nil), s.records...)
}
