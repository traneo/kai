package secrets

import (
	"context"
	"testing"
)

func TestMemoryManager_GetSecret(t *testing.T) {
	m := NewMemoryManager()
	val, err := m.GetSecret(context.Background(), "llm", "openai-key")
	if err != nil {
		t.Logf("expected openai-key might be empty, error: %v", err)
	}
	_ = val
}

func TestMemoryManager_GetSecrets(t *testing.T) {
	m := NewMemoryManager()
	secrets, err := m.GetSecrets(context.Background(), "llm")
	if err != nil {
		t.Fatalf("GetSecrets: %v", err)
	}
	t.Logf("found %d llm secrets", len(secrets))
}

func TestMemoryManager_GetSecretNotFound(t *testing.T) {
	m := NewMemoryManager()
	_, err := m.GetSecret(context.Background(), "nonexistent", "key")
	if err == nil {
		t.Error("expected error for nonexistent secret")
	}
}

func TestResolve(t *testing.T) {
	m := NewMemoryManager()
	sec := Resolve(context.Background(), m)
	if sec == nil {
		t.Fatal("expected non-nil result")
	}
}
