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
	DomainKline      = "kline"
	DomainAsset      = "asset"
	DomainHealthFood = "health_food"
)

type Case struct {
	ID             int64      `json:"id"`
	CaseNo         string     `json:"case_no"`
	Title          string     `json:"title,omitempty"`
	UID            string     `json:"uid,omitempty"`
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
	ID                int64     `json:"id"`
	CaseID            int64     `json:"case_id"`
	Role              string    `json:"role"`
	PlatformMessageID string    `json:"platform_message_id,omitempty"`
	Content           string    `json:"content"`
	ContentType       string    `json:"content_type"`
	CreatedAt         time.Time `json:"created_at"`
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

type AIDecisionLog struct {
	ID                 int64     `json:"id"`
	CaseID             int64     `json:"case_id"`
	InvestigationID    int64     `json:"investigation_id,omitempty"`
	AgentID            string    `json:"agent_id"`
	DecisionType       string    `json:"decision_type"`
	Reason             string    `json:"reason,omitempty"`
	InputSnapshotJSON  string    `json:"input_snapshot_json,omitempty"`
	OutputSnapshotJSON string    `json:"output_snapshot_json,omitempty"`
	SelectedToolsJSON  string    `json:"selected_tools_json,omitempty"`
	Status             string    `json:"status"`
	LatencyMS          int64     `json:"latency_ms"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type RootCause struct {
	ID                     int64     `json:"id"`
	CaseID                 int64     `json:"case_id"`
	AIPredictedReason      string    `json:"ai_predicted_reason,omitempty"`
	HumanConfirmedReason   string    `json:"human_confirmed_reason"`
	RootCauseCategory      string    `json:"root_cause_category"`
	OwnerService           string    `json:"owner_service,omitempty"`
	OwnerTeam              string    `json:"owner_team,omitempty"`
	IsCacheIssue           bool      `json:"is_cache_issue"`
	IsDataSyncIssue        bool      `json:"is_data_sync_issue"`
	IsExternalSourceIssue  bool      `json:"is_external_source_issue"`
	IsFrontendDisplayIssue bool      `json:"is_frontend_display_issue"`
	IsUserMisunderstanding bool      `json:"is_user_misunderstanding"`
	FixAction              string    `json:"fix_action,omitempty"`
	PreventionAction       string    `json:"prevention_action,omitempty"`
	ConfirmedBy            string    `json:"confirmed_by,omitempty"`
	ConfirmedAt            time.Time `json:"confirmed_at"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type CaseFeedback struct {
	ID                    int64     `json:"id"`
	CaseID                int64     `json:"case_id"`
	Rating                int       `json:"rating"`
	AIUseful              bool      `json:"ai_useful"`
	WrongConclusion       bool      `json:"wrong_conclusion"`
	MissingKeyInformation string    `json:"missing_key_information,omitempty"`
	MissingToolsJSON      string    `json:"missing_tools_json,omitempty"`
	Comment               string    `json:"comment,omitempty"`
	CreatedBy             string    `json:"created_by,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type KnowledgeItem struct {
	ID                    int64     `json:"id"`
	Title                 string    `json:"title"`
	IssueDomain           string    `json:"issue_domain"`
	IssueType             string    `json:"issue_type,omitempty"`
	TypicalDescription    string    `json:"typical_description,omitempty"`
	TypicalOCRFeatures    string    `json:"typical_ocr_features,omitempty"`
	RequiredFieldsJSON    string    `json:"required_fields_json,omitempty"`
	RecommendedStepsJSON  string    `json:"recommended_steps_json,omitempty"`
	CommonCausesJSON      string    `json:"common_causes_json,omitempty"`
	UsefulToolsJSON       string    `json:"useful_tools_json,omitempty"`
	SuccessCaseIDsJSON    string    `json:"success_case_ids_json,omitempty"`
	FailureCaseIDsJSON    string    `json:"failure_case_ids_json,omitempty"`
	Confidence            float64   `json:"confidence"`
	Status                string    `json:"status"`
	ObservedCaseCount     int       `json:"observed_case_count"`
	LastRootCauseCategory string    `json:"last_root_cause_category,omitempty"`
	LastConfirmedReason   string    `json:"last_confirmed_reason,omitempty"`
	LastEvolvedAt         time.Time `json:"last_evolved_at"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type KnowledgeEvolutionRun struct {
	ID                   int64     `json:"id"`
	RunNo                string    `json:"run_no"`
	CaseID               int64     `json:"case_id"`
	KnowledgeItemID      int64     `json:"knowledge_item_id"`
	TriggerType          string    `json:"trigger_type"`
	InputSnapshotJSON    string    `json:"input_snapshot_json"`
	OutputSummary        string    `json:"output_summary"`
	Decision             string    `json:"decision"`
	CreatedKnowledgeItem bool      `json:"created_knowledge_item"`
	UpdatedKnowledgeItem bool      `json:"updated_knowledge_item"`
	ErrorMessage         string    `json:"error_message,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

type ProcessResult struct {
	CaseID        int64    `json:"case_id"`
	CaseNo        string   `json:"case_no"`
	Status        Status   `json:"status"`
	Reply         string   `json:"reply"`
	ToolCallIDs   []string `json:"tool_call_ids,omitempty"`
	MissingFields []string `json:"missing_fields,omitempty"`
}
