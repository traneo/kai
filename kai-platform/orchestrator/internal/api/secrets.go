package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kaiplatform.com/orchestrator/internal/api/coordinator"
)

type MemorySecretStore struct {
	mu   sync.RWMutex
	data map[string]secretEntry
}

type secretEntry struct {
	value       string
	description string
	createdAt   time.Time
	updatedAt   time.Time
}

func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{
		data: make(map[string]secretEntry),
	}
}

func (s *MemorySecretStore) List(_ context.Context) ([]coordinator.SecretMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]coordinator.SecretMeta, 0, len(s.data))
	for name, entry := range s.data {
		result = append(result, coordinator.SecretMeta{
			Name:        name,
			Description: entry.description,
			CreatedAt:   entry.createdAt,
			UpdatedAt:   entry.updatedAt,
		})
	}
	return result, nil
}

func (s *MemorySecretStore) GetValue(_ context.Context, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[name]
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}
	return entry.value, nil
}

func (s *MemorySecretStore) Set(_ context.Context, name, value, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	existing, ok := s.data[name]
	if ok {
		s.data[name] = secretEntry{
			value:       value,
			description: coalesce(description, existing.description),
			createdAt:   existing.createdAt,
			updatedAt:   now,
		}
	} else {
		s.data[name] = secretEntry{
			value:       value,
			description: description,
			createdAt:   now,
			updatedAt:   now,
		}
	}
	return nil
}

func (s *MemorySecretStore) Delete(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[name]; !ok {
		return fmt.Errorf("secret %q not found", name)
	}
	delete(s.data, name)
	return nil
}

func coalesce(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}
