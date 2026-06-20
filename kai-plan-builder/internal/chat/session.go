package chat

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/kaiplatform/plan-builder/internal/llm"
)

type Session struct {
	ID       string
	Spec     string
	Messages []llm.Message
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	counter  atomic.Int64
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*Session),
	}
}

func (s *Store) Create() *Session {
	id := fmt.Sprintf("conv-%d", s.counter.Add(1))
	sess := &Session{
		ID:       id,
		Spec:     "",
		Messages: []llm.Message{},
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess
}

func (s *Store) Get(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}
