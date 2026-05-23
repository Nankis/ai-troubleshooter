package caseflow

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound        = errors.New("case not found")
	ErrVersionConflict = errors.New("case version conflict")
)

type CreateCaseInput struct {
	UID            string
	Source         string
	ChatID         string
	ThreadID       string
	MessageID      string
	ReporterUserID string
	OriginalText   string
	OCRText        string
	Timezone       string
}

type Store interface {
	CreateCase(ctx context.Context, input CreateCaseInput) (*Case, error)
	GetCase(ctx context.Context, id int64) (*Case, error)
	FindCaseByNo(ctx context.Context, caseNo string) (*Case, error)
	FindCaseByMessageID(ctx context.Context, source string, messageID string) (*Case, error)
	ListRecentCases(ctx context.Context, limit int) ([]Case, error)
	UpdateCase(ctx context.Context, id int64, expectedVersion int64, update func(*Case) error) (*Case, error)
	AddEntities(ctx context.Context, caseID int64, entities []Entity) error
	ListEntities(ctx context.Context, caseID int64) ([]Entity, error)
	AddMessage(ctx context.Context, msg Message) (Message, error)
	ListMessages(ctx context.Context, caseID int64) ([]Message, error)
	CreateInvestigation(ctx context.Context, inv Investigation) (Investigation, error)
	FinishInvestigation(ctx context.Context, id int64, status string, summary string, confidence *float64) (Investigation, error)
	AddAIDecisionLog(ctx context.Context, log AIDecisionLog) (AIDecisionLog, error)
	ListAIDecisionLogs(ctx context.Context, caseID int64, limit int) ([]AIDecisionLog, error)
	UpsertRootCause(ctx context.Context, rootCause RootCause) (RootCause, error)
	GetRootCause(ctx context.Context, caseID int64) (RootCause, error)
	AddCaseFeedback(ctx context.Context, feedback CaseFeedback) (CaseFeedback, error)
	ListCaseFeedback(ctx context.Context, caseID int64) ([]CaseFeedback, error)
	UpsertKnowledgeItem(ctx context.Context, item KnowledgeItem) (KnowledgeItem, error)
	GetKnowledgeItem(ctx context.Context, id int64) (KnowledgeItem, error)
	FindKnowledgeItem(ctx context.Context, issueDomain string, issueType string, rootCauseCategory string) (KnowledgeItem, error)
	ListKnowledgeItems(ctx context.Context, filter KnowledgeFilter) ([]KnowledgeItem, error)
	DeleteKnowledgeItem(ctx context.Context, id int64) error
	CreateKnowledgeEvolutionRun(ctx context.Context, run KnowledgeEvolutionRun) (KnowledgeEvolutionRun, error)
	ListKnowledgeEvolutionRuns(ctx context.Context, caseID int64) ([]KnowledgeEvolutionRun, error)
}

type InMemoryStore struct {
	mu                          sync.RWMutex
	nextCaseID                  int64
	nextEntityID                int64
	nextMessageID               int64
	nextInvestigationID         int64
	nextAIDecisionLogID         int64
	nextRootCauseID             int64
	nextFeedbackID              int64
	nextKnowledgeItemID         int64
	nextKnowledgeEvolutionRunID int64
	cases                       map[int64]*Case
	casesByNo                   map[string]int64
	casesBySourceMessageID      map[string]int64
	entities                    map[int64][]Entity
	messages                    map[int64][]Message
	investigations              map[int64]*Investigation
	aiDecisionLogs              map[int64][]AIDecisionLog
	rootCauses                  map[int64]*RootCause
	feedback                    map[int64][]CaseFeedback
	knowledgeItems              map[int64]*KnowledgeItem
	knowledgeIndex              map[string]int64
	evolutionRuns               map[int64][]KnowledgeEvolutionRun
}

type KnowledgeFilter struct {
	IssueDomain       string
	IssueType         string
	RootCauseCategory string
	Status            string
	Limit             int
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		cases:                  map[int64]*Case{},
		casesByNo:              map[string]int64{},
		casesBySourceMessageID: map[string]int64{},
		entities:               map[int64][]Entity{},
		messages:               map[int64][]Message{},
		investigations:         map[int64]*Investigation{},
		aiDecisionLogs:         map[int64][]AIDecisionLog{},
		rootCauses:             map[int64]*RootCause{},
		feedback:               map[int64][]CaseFeedback{},
		knowledgeItems:         map[int64]*KnowledgeItem{},
		knowledgeIndex:         map[string]int64{},
		evolutionRuns:          map[int64][]KnowledgeEvolutionRun{},
	}
}

func (s *InMemoryStore) CreateCase(ctx context.Context, input CreateCaseInput) (*Case, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	source := fallback(input.Source, "lark")
	if key := sourceMessageKey(source, input.MessageID); key != "" {
		if id, ok := s.casesBySourceMessageID[key]; ok {
			return cloneCase(s.cases[id]), nil
		}
	}

	s.nextCaseID++
	now := time.Now()
	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	c := &Case{
		ID:             s.nextCaseID,
		CaseNo:         fmt.Sprintf("case_%s_%06d", now.Format("20060102"), s.nextCaseID),
		UID:            fallback(input.UID, input.ReporterUserID),
		Source:         source,
		ChatID:         input.ChatID,
		ThreadID:       input.ThreadID,
		MessageID:      input.MessageID,
		ReporterUserID: input.ReporterUserID,
		OriginalText:   input.OriginalText,
		OCRText:        input.OCRText,
		Status:         StatusNew,
		Priority:       "normal",
		Timezone:       tz,
		CreatedAt:      now,
		UpdatedAt:      now,
		Version:        0,
	}
	s.cases[c.ID] = cloneCase(c)
	s.casesByNo[c.CaseNo] = c.ID
	if key := sourceMessageKey(c.Source, c.MessageID); key != "" {
		s.casesBySourceMessageID[key] = c.ID
	}
	return cloneCase(c), nil
}

func (s *InMemoryStore) GetCase(ctx context.Context, id int64) (*Case, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.cases[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneCase(c), nil
}

func (s *InMemoryStore) FindCaseByNo(ctx context.Context, caseNo string) (*Case, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.casesByNo[caseNo]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneCase(s.cases[id]), nil
}

func (s *InMemoryStore) FindCaseByMessageID(ctx context.Context, source string, messageID string) (*Case, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := sourceMessageKey(fallback(source, "lark"), messageID)
	if key == "" {
		return nil, ErrNotFound
	}
	id, ok := s.casesBySourceMessageID[key]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneCase(s.cases[id]), nil
}

func (s *InMemoryStore) ListRecentCases(ctx context.Context, limit int) ([]Case, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	items := make([]Case, 0, len(s.cases))
	for _, c := range s.cases {
		items = append(items, *cloneCase(c))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *InMemoryStore) UpdateCase(ctx context.Context, id int64, expectedVersion int64, update func(*Case) error) (*Case, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.cases[id]
	if !ok {
		return nil, ErrNotFound
	}
	if current.Version != expectedVersion {
		return nil, ErrVersionConflict
	}
	next := cloneCase(current)
	if err := update(next); err != nil {
		return nil, err
	}
	next.Version++
	next.UpdatedAt = time.Now()
	s.cases[id] = cloneCase(next)
	return cloneCase(next), nil
}

func (s *InMemoryStore) AddEntities(ctx context.Context, caseID int64, entities []Entity) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[caseID]; !ok {
		return ErrNotFound
	}
	now := time.Now()
	for _, entity := range entities {
		if entity.Value == "" || entity.Type == "" {
			continue
		}
		if hasEntity(s.entities[caseID], entity.Type, entity.Value) {
			continue
		}
		s.nextEntityID++
		entity.ID = s.nextEntityID
		entity.CaseID = caseID
		entity.CreatedAt = now
		if entity.Source == "" {
			entity.Source = "llm"
		}
		s.entities[caseID] = append(s.entities[caseID], entity)
	}
	return nil
}

func (s *InMemoryStore) ListEntities(ctx context.Context, caseID int64) ([]Entity, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Entity(nil), s.entities[caseID]...)
	return out, nil
}

func (s *InMemoryStore) AddMessage(ctx context.Context, msg Message) (Message, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[msg.CaseID]; !ok {
		return Message{}, ErrNotFound
	}
	s.nextMessageID++
	msg.ID = s.nextMessageID
	msg.CreatedAt = time.Now()
	if msg.ContentType == "" {
		msg.ContentType = "text"
	}
	s.messages[msg.CaseID] = append(s.messages[msg.CaseID], msg)
	return msg, nil
}

func (s *InMemoryStore) ListMessages(ctx context.Context, caseID int64) ([]Message, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Message(nil), s.messages[caseID]...)
	return out, nil
}

func (s *InMemoryStore) CreateInvestigation(ctx context.Context, inv Investigation) (Investigation, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[inv.CaseID]; !ok {
		return Investigation{}, ErrNotFound
	}
	s.nextInvestigationID++
	now := time.Now()
	inv.ID = s.nextInvestigationID
	inv.InvestigationNo = fmt.Sprintf("inv_%s_%06d", now.Format("20060102"), inv.ID)
	inv.StartedAt = now
	inv.CreatedAt = now
	inv.UpdatedAt = now
	if inv.Status == "" {
		inv.Status = "running"
	}
	s.investigations[inv.ID] = cloneInvestigation(&inv)
	return inv, nil
}

func (s *InMemoryStore) FinishInvestigation(ctx context.Context, id int64, status string, summary string, confidence *float64) (Investigation, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.investigations[id]
	if !ok {
		return Investigation{}, ErrNotFound
	}
	now := time.Now()
	inv.Status = status
	inv.FinalSummary = summary
	inv.Confidence = confidence
	inv.FinishedAt = &now
	inv.UpdatedAt = now
	return *cloneInvestigation(inv), nil
}

func (s *InMemoryStore) AddAIDecisionLog(ctx context.Context, log AIDecisionLog) (AIDecisionLog, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[log.CaseID]; !ok {
		return AIDecisionLog{}, ErrNotFound
	}
	s.nextAIDecisionLogID++
	log.ID = s.nextAIDecisionLogID
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	if log.Status == "" {
		log.Status = "success"
	}
	s.aiDecisionLogs[log.CaseID] = append(s.aiDecisionLogs[log.CaseID], log)
	return log, nil
}

func (s *InMemoryStore) ListAIDecisionLogs(ctx context.Context, caseID int64, limit int) ([]AIDecisionLog, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	items := s.aiDecisionLogs[caseID]
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return append([]AIDecisionLog(nil), items...), nil
}

func (s *InMemoryStore) UpsertRootCause(ctx context.Context, rootCause RootCause) (RootCause, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[rootCause.CaseID]; !ok {
		return RootCause{}, ErrNotFound
	}
	now := time.Now()
	if rootCause.ConfirmedAt.IsZero() {
		rootCause.ConfirmedAt = now
	}
	if existing, ok := s.rootCauses[rootCause.CaseID]; ok {
		rootCause.ID = existing.ID
		rootCause.CreatedAt = existing.CreatedAt
		rootCause.UpdatedAt = now
		s.rootCauses[rootCause.CaseID] = cloneRootCause(&rootCause)
		return rootCause, nil
	}
	s.nextRootCauseID++
	rootCause.ID = s.nextRootCauseID
	rootCause.CreatedAt = now
	rootCause.UpdatedAt = now
	s.rootCauses[rootCause.CaseID] = cloneRootCause(&rootCause)
	return rootCause, nil
}

func (s *InMemoryStore) GetRootCause(ctx context.Context, caseID int64) (RootCause, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	rootCause, ok := s.rootCauses[caseID]
	if !ok {
		return RootCause{}, ErrNotFound
	}
	return *cloneRootCause(rootCause), nil
}

func (s *InMemoryStore) AddCaseFeedback(ctx context.Context, feedback CaseFeedback) (CaseFeedback, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[feedback.CaseID]; !ok {
		return CaseFeedback{}, ErrNotFound
	}
	s.nextFeedbackID++
	feedback.ID = s.nextFeedbackID
	feedback.CreatedAt = time.Now()
	s.feedback[feedback.CaseID] = append(s.feedback[feedback.CaseID], feedback)
	return feedback, nil
}

func (s *InMemoryStore) ListCaseFeedback(ctx context.Context, caseID int64) ([]CaseFeedback, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]CaseFeedback(nil), s.feedback[caseID]...), nil
}

func (s *InMemoryStore) UpsertKnowledgeItem(ctx context.Context, item KnowledgeItem) (KnowledgeItem, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	key := knowledgeKey(item.IssueDomain, item.IssueType, item.LastRootCauseCategory)
	if item.ID == 0 {
		if id, ok := s.knowledgeIndex[key]; ok {
			item.ID = id
		}
	}
	if item.ID != 0 {
		existing, ok := s.knowledgeItems[item.ID]
		if !ok {
			return KnowledgeItem{}, ErrNotFound
		}
		item.CreatedAt = existing.CreatedAt
		item.UpdatedAt = now
		if item.LastEvolvedAt.IsZero() {
			item.LastEvolvedAt = now
		}
		if item.Status == "" {
			item.Status = existing.Status
		}
		s.knowledgeItems[item.ID] = cloneKnowledgeItem(&item)
		s.knowledgeIndex[key] = item.ID
		return item, nil
	}
	s.nextKnowledgeItemID++
	item.ID = s.nextKnowledgeItemID
	if item.Status == "" {
		item.Status = "active"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if item.LastEvolvedAt.IsZero() {
		item.LastEvolvedAt = now
	}
	s.knowledgeItems[item.ID] = cloneKnowledgeItem(&item)
	s.knowledgeIndex[key] = item.ID
	return item, nil
}

func (s *InMemoryStore) GetKnowledgeItem(ctx context.Context, id int64) (KnowledgeItem, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.knowledgeItems[id]
	if !ok {
		return KnowledgeItem{}, ErrNotFound
	}
	return *cloneKnowledgeItem(item), nil
}

func (s *InMemoryStore) FindKnowledgeItem(ctx context.Context, issueDomain string, issueType string, rootCauseCategory string) (KnowledgeItem, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.knowledgeIndex[knowledgeKey(issueDomain, issueType, rootCauseCategory)]
	if !ok {
		return KnowledgeItem{}, ErrNotFound
	}
	return *cloneKnowledgeItem(s.knowledgeItems[id]), nil
}

func (s *InMemoryStore) ListKnowledgeItems(ctx context.Context, filter KnowledgeFilter) ([]KnowledgeItem, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	out := []KnowledgeItem{}
	for _, item := range s.knowledgeItems {
		if filter.Status == "" && item.Status == "deleted" {
			continue
		}
		if filter.IssueDomain != "" && item.IssueDomain != filter.IssueDomain {
			continue
		}
		if filter.IssueType != "" && item.IssueType != filter.IssueType {
			continue
		}
		if filter.RootCauseCategory != "" && item.LastRootCauseCategory != filter.RootCauseCategory {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		out = append(out, *cloneKnowledgeItem(item))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *InMemoryStore) DeleteKnowledgeItem(ctx context.Context, id int64) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.knowledgeItems[id]
	if !ok {
		return ErrNotFound
	}
	next := *item
	next.Status = "deleted"
	next.UpdatedAt = time.Now()
	s.knowledgeItems[id] = &next
	return nil
}

func (s *InMemoryStore) CreateKnowledgeEvolutionRun(ctx context.Context, run KnowledgeEvolutionRun) (KnowledgeEvolutionRun, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[run.CaseID]; !ok {
		return KnowledgeEvolutionRun{}, ErrNotFound
	}
	s.nextKnowledgeEvolutionRunID++
	now := time.Now()
	run.ID = s.nextKnowledgeEvolutionRunID
	run.RunNo = fmt.Sprintf("ke_%s_%06d", now.Format("20060102"), run.ID)
	run.CreatedAt = now
	s.evolutionRuns[run.CaseID] = append(s.evolutionRuns[run.CaseID], run)
	return run, nil
}

func (s *InMemoryStore) ListKnowledgeEvolutionRuns(ctx context.Context, caseID int64) ([]KnowledgeEvolutionRun, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]KnowledgeEvolutionRun(nil), s.evolutionRuns[caseID]...), nil
}

func cloneCase(c *Case) *Case {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

func cloneInvestigation(inv *Investigation) *Investigation {
	if inv == nil {
		return nil
	}
	cp := *inv
	return &cp
}

func cloneRootCause(rootCause *RootCause) *RootCause {
	if rootCause == nil {
		return nil
	}
	cp := *rootCause
	return &cp
}

func cloneKnowledgeItem(item *KnowledgeItem) *KnowledgeItem {
	if item == nil {
		return nil
	}
	cp := *item
	return &cp
}

func knowledgeKey(issueDomain string, issueType string, rootCauseCategory string) string {
	return issueDomain + "|" + issueType + "|" + rootCauseCategory
}

func sourceMessageKey(source string, messageID string) string {
	if source == "" || messageID == "" {
		return ""
	}
	return source + "|" + messageID
}

func hasEntity(entities []Entity, typ string, value string) bool {
	for _, entity := range entities {
		if entity.Type == typ && entity.Value == value {
			return true
		}
	}
	return false
}

func fallback(v string, def string) string {
	if v != "" {
		return v
	}
	return def
}
