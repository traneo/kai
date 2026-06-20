package store

import (
	"context"
	"strings"
	"sync"
	"time"

	"kaiplatform.com/observability/internal/models"
)

type MemoryStore struct {
	mu      sync.RWMutex
	entries []models.LogEntry
	cap     int
}

func NewMemoryStore(cap int) *MemoryStore {
	if cap <= 0 {
		cap = 50000
	}
	return &MemoryStore{cap: cap}
}

func (s *MemoryStore) Append(_ context.Context, entries []models.LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entries...)
	if len(s.entries) > s.cap {
		trim := len(s.entries) - s.cap
		s.entries = s.entries[trim:]
	}
	return nil
}

func (s *MemoryStore) Query(_ context.Context, filter models.QueryFilter) ([]models.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var result []models.LogEntry
	for i := len(s.entries) - 1; i >= 0; i-- {
		e := s.entries[i]
		if filter.Service != "" && e.Service != filter.Service {
			continue
		}
		if filter.Level != "" && e.Level != filter.Level {
			continue
		}
		if !filter.From.IsZero() && time.UnixMilli(e.Timestamp).Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && time.UnixMilli(e.Timestamp).After(filter.To) {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, e)
	}

	if offset >= len(result) {
		return nil, nil
	}
	result = result[offset:]
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *MemoryStore) GetByID(_ context.Context, id string) (*models.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) Close() error { return nil }
