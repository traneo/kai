package api

import (
	"sync"

	"kaiplatform.com/observability/internal/models"
)

type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[chan models.LogEntry]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[chan models.LogEntry]struct{}),
	}
}

func (h *SSEHub) Subscribe() chan models.LogEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan models.LogEntry, 256)
	h.subscribers[ch] = struct{}{}
	return ch
}

func (h *SSEHub) Unsubscribe(ch chan models.LogEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, ch)
}

func (h *SSEHub) Publish(entry models.LogEntry) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- entry:
		default:
		}
	}
}
