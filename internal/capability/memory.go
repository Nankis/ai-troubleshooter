package capability

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu             sync.RWMutex
	nextServiceID  int64
	nextServerID   int64
	nextToolID     int64
	nextRunID      int64
	services       map[string]BusinessService
	servers        map[int64]MCPServer
	tools          map[int64]ToolCapability
	toolsByName    map[string]int64
	validationRuns map[int64]ValidationRun
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		services:       map[string]BusinessService{},
		servers:        map[int64]MCPServer{},
		tools:          map[int64]ToolCapability{},
		toolsByName:    map[string]int64{},
		validationRuns: map[int64]ValidationRun{},
	}
}

func (s *MemoryStore) UpsertBusinessService(ctx context.Context, service BusinessService) (BusinessService, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if existing, ok := s.services[service.ServiceName]; ok {
		service.ID = existing.ID
		service.CreatedAt = existing.CreatedAt
		service.UpdatedAt = now
		s.services[service.ServiceName] = service
		return service, nil
	}
	s.nextServiceID++
	service.ID = s.nextServiceID
	service.CreatedAt = now
	service.UpdatedAt = now
	s.services[service.ServiceName] = service
	return service, nil
}

func (s *MemoryStore) CreateMCPServer(ctx context.Context, server MCPServer) (MCPServer, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextServerID++
	now := time.Now()
	server.ID = s.nextServerID
	server.CreatedAt = now
	server.UpdatedAt = now
	s.servers[server.ID] = server
	return server, nil
}

func (s *MemoryStore) UpsertToolCapability(ctx context.Context, item ToolCapability) (ToolCapability, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if id, ok := s.toolsByName[item.ToolName]; ok {
		item.ID = id
		item.CreatedAt = s.tools[id].CreatedAt
		item.UpdatedAt = now
		s.tools[id] = item
		return item, nil
	}
	s.nextToolID++
	item.ID = s.nextToolID
	item.CreatedAt = now
	item.UpdatedAt = now
	s.tools[item.ID] = item
	s.toolsByName[item.ToolName] = item.ID
	return item, nil
}

func (s *MemoryStore) GetToolCapability(ctx context.Context, id int64) (ToolCapability, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.tools[id]
	if !ok {
		return ToolCapability{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) ListToolCapabilities(ctx context.Context, filter ToolFilter) ([]ToolCapability, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []ToolCapability{}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	for _, item := range s.tools {
		if filter.Status != "" && item.ToolStatus != filter.Status {
			continue
		}
		if filter.SourceType != "" && item.SourceType != filter.SourceType {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ServiceName != out[j].ServiceName {
			return out[i].ServiceName < out[j].ServiceName
		}
		return out[i].ToolName < out[j].ToolName
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) UpdateToolCapabilityStatus(ctx context.Context, id int64, status string, publishedBy string) (ToolCapability, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.tools[id]
	if !ok {
		return ToolCapability{}, ErrNotFound
	}
	item.ToolStatus = status
	item.PublishedBy = publishedBy
	if status == StatusEnabled {
		now := time.Now()
		item.PublishedAt = &now
		item.ApprovalStatus = "approved"
	}
	item.UpdatedAt = time.Now()
	s.tools[id] = item
	return item, nil
}

func (s *MemoryStore) CreateValidationRun(ctx context.Context, run ValidationRun) (ValidationRun, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextRunID++
	run.ID = s.nextRunID
	run.CreatedAt = time.Now()
	s.validationRuns[run.ID] = run
	return run, nil
}
