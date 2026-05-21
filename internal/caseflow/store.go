package caseflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrNotFound        = errors.New("case not found")
	ErrVersionConflict = errors.New("case version conflict")
)

type CreateCaseInput struct {
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
	UpdateCase(ctx context.Context, id int64, expectedVersion int64, update func(*Case) error) (*Case, error)
	AddEntities(ctx context.Context, caseID int64, entities []Entity) error
	ListEntities(ctx context.Context, caseID int64) ([]Entity, error)
	AddMessage(ctx context.Context, msg Message) (Message, error)
	ListMessages(ctx context.Context, caseID int64) ([]Message, error)
	CreateInvestigation(ctx context.Context, inv Investigation) (Investigation, error)
	FinishInvestigation(ctx context.Context, id int64, status string, summary string, confidence *float64) (Investigation, error)
}

type InMemoryStore struct {
	mu                  sync.RWMutex
	nextCaseID          int64
	nextEntityID        int64
	nextMessageID       int64
	nextInvestigationID int64
	cases               map[int64]*Case
	casesByNo           map[string]int64
	entities            map[int64][]Entity
	messages            map[int64][]Message
	investigations      map[int64]*Investigation
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		cases:          map[int64]*Case{},
		casesByNo:      map[string]int64{},
		entities:       map[int64][]Entity{},
		messages:       map[int64][]Message{},
		investigations: map[int64]*Investigation{},
	}
}

func (s *InMemoryStore) CreateCase(ctx context.Context, input CreateCaseInput) (*Case, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextCaseID++
	now := time.Now()
	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	c := &Case{
		ID:             s.nextCaseID,
		CaseNo:         fmt.Sprintf("case_%s_%06d", now.Format("20060102"), s.nextCaseID),
		Source:         fallback(input.Source, "lark"),
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
