package audit

import (
	"context"
	"testing"
)

func TestMemoryStore_Append(t *testing.T) {
	s := NewMemoryStore()
	err := s.Append(context.Background(), &Event{
		Type:    EventSystem,
		RunID:   "run-1",
		Message: "test event",
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 5; i++ {
		s.Append(context.Background(), &Event{Type: EventSystem, Message: "e"})
	}
	events, err := s.List(context.Background(), 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestMemoryStore_ListAll(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 5; i++ {
		s.Append(context.Background(), &Event{Type: EventSystem, Message: "e"})
	}
	events, err := s.List(context.Background(), 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}
}

func TestMemoryStore_Query(t *testing.T) {
	s := NewMemoryStore()
	s.Append(context.Background(), &Event{Type: EventPipelineCreated, RunID: "run-1"})
	s.Append(context.Background(), &Event{Type: EventPipelineCreated, RunID: "run-2"})
	s.Append(context.Background(), &Event{Type: EventStepStarted, RunID: "run-1"})

	events, err := s.Query(context.Background(), "run-1", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events for run-1, got %d", len(events))
	}
}

func TestMemoryStore_QueryLimit(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 10; i++ {
		s.Append(context.Background(), &Event{Type: EventSystem, RunID: "run-1"})
	}
	events, err := s.Query(context.Background(), "run-1", 3)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(events) > 3 {
		t.Errorf("expected at most 3 events, got %d", len(events))
	}
}

func TestMemoryStore_IDAutoIncrement(t *testing.T) {
	s := NewMemoryStore()
	e1 := &Event{Type: EventSystem}
	s.Append(context.Background(), e1)
	e2 := &Event{Type: EventSystem}
	s.Append(context.Background(), e2)
	if e2.ID <= e1.ID {
		t.Errorf("expected auto-increment: e1=%d e2=%d", e1.ID, e2.ID)
	}
}

func TestLog(t *testing.T) {
	s := NewMemoryStore()
	Log(s, EventSystem, "run-1", "step-1", "agent-1", "hello", map[string]string{"key": "val"})
	events, _ := s.List(context.Background(), 1)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Message != "hello" {
		t.Errorf("expected message 'hello', got '%s'", events[0].Message)
	}
}

func TestMemoryStore_Close(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
