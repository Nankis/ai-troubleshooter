package caseflow

import "time"

type Status string

const (
	StatusNew                   Status = "NEW"
	StatusNeedMoreInfo          Status = "NEED_MORE_INFO"
	StatusWaitingUserReply      Status = "WAITING_USER_REPLY"
	StatusReadyToInvestigate    Status = "READY_TO_INVESTIGATE"
	StatusInvestigating         Status = "INVESTIGATING"
	StatusWaitingToolResult     Status = "WAITING_TOOL_RESULT"
	StatusNeedHumanConfirmation Status = "NEED_HUMAN_CONFIRMATION"
	StatusDone                  Status = "DONE"
	StatusFailed                Status = "FAILED"
	StatusCancelled             Status = "CANCELLED"
)

const (
	DomainKline = "kline"
	DomainAsset = "asset"
)

type Case struct {
	ID             int64      `json:"id"`
	CaseNo         string     `json:"case_no"`
	Source         string     `json:"source"`
	ChatID         string     `json:"chat_id,omitempty"`
	ThreadID       string     `json:"thread_id,omitempty"`
	MessageID      string     `json:"message_id,omitempty"`
	ReporterUserID string     `json:"reporter_user_id,omitempty"`
	OriginalText   string     `json:"original_text,omitempty"`
	OCRText        string     `json:"ocr_text,omitempty"`
	IssueDomain    string     `json:"issue_domain,omitempty"`
	IssueType      string     `json:"issue_type,omitempty"`
	Status         Status     `json:"status"`
	Priority       string     `json:"priority"`
	Timezone       string     `json:"timezone"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	Version        int64      `json:"version"`
}

type Entity struct {
	ID         int64     `json:"id"`
	CaseID     int64     `json:"case_id"`
	Type       string    `json:"entity_type"`
	Value      string    `json:"entity_value"`
	Source     string    `json:"source"`
	Confidence *float64  `json:"confidence,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Message struct {
	ID            int64     `json:"id"`
	CaseID        int64     `json:"case_id"`
	Role          string    `json:"role"`
	LarkMessageID string    `json:"lark_message_id,omitempty"`
	Content       string    `json:"content"`
	ContentType   string    `json:"content_type"`
	CreatedAt     time.Time `json:"created_at"`
}

type Investigation struct {
	ID                int64      `json:"id"`
	InvestigationNo   string     `json:"investigation_no"`
	CaseID            int64      `json:"case_id"`
	AgentID           string     `json:"agent_id"`
	AgentVersion      string     `json:"agent_version,omitempty"`
	ModelProvider     string     `json:"model_provider,omitempty"`
	ModelName         string     `json:"model_name,omitempty"`
	Status            string     `json:"status"`
	InitialHypothesis string     `json:"initial_hypothesis,omitempty"`
	FinalSummary      string     `json:"final_summary,omitempty"`
	Confidence        *float64   `json:"confidence,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ProcessResult struct {
	CaseID        int64    `json:"case_id"`
	CaseNo        string   `json:"case_no"`
	Status        Status   `json:"status"`
	Reply         string   `json:"reply"`
	ToolCallIDs   []string `json:"tool_call_ids,omitempty"`
	MissingFields []string `json:"missing_fields,omitempty"`
}
