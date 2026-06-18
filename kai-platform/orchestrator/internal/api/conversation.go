package api

import (
	"sort"
	"sync"

	"kaiplatform.com/orchestrator/internal/api/coordinator"
)

type MemoryConversationStore struct {
	mu  sync.RWMutex
	buf []*coordinator.ConversationEntry
	seq int64
}

func NewMemoryConversationStore() *MemoryConversationStore {
	return &MemoryConversationStore{
		buf: make([]*coordinator.ConversationEntry, 0, 1024),
	}
}

func (s *MemoryConversationStore) Append(entry *coordinator.ConversationEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	if entry.Sequence == 0 {
		entry.Sequence = s.seq
	}
	s.buf = append(s.buf, entry)
	if len(s.buf) > 50000 {
		s.buf = s.buf[len(s.buf)-25000:]
	}
}

func (s *MemoryConversationStore) List(runID, stepID string, limit int) []*coordinator.ConversationEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*coordinator.ConversationEntry
	for i := len(s.buf) - 1; i >= 0; i-- {
		entry := s.buf[i]
		if entry.RunID != runID {
			continue
		}
		if stepID != "" && entry.StepID != stepID {
			continue
		}
		result = append(result, entry)
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Sequence < result[j].Sequence
	})
	return result
}

func (s *MemoryConversationStore) bufLen() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.buf)
}

func (s *MemoryConversationStore) Close() {}
