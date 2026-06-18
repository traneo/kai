package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type EventType string

const (
	EventPipelineCreated  EventType = "pipeline_created"
	EventPipelineStarted  EventType = "pipeline_started"
	EventPipelineCompleted EventType = "pipeline_completed"
	EventPipelineFailed   EventType = "pipeline_failed"
	EventStepStarted      EventType = "step_started"
	EventStepCompleted    EventType = "step_completed"
	EventStepFailed       EventType = "step_failed"
	EventStepRetry        EventType = "step_retry"
	EventStepBlocked      EventType = "step_blocked"
	EventAgentAssigned    EventType = "agent_assigned"
	EventApprovalGranted  EventType = "approval_granted"
	EventApprovalRejected EventType = "approval_rejected"
	EventSecretInjected   EventType = "secret_injected"
	EventSystem           EventType = "system"
)

type Event struct {
	ID        int64     `json:"id"`
	Time      time.Time `json:"time"`
	Type      EventType `json:"type"`
	RunID     string    `json:"run_id,omitempty"`
	StepID    string    `json:"step_id,omitempty"`
	AgentID   string    `json:"agent_id,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	Message   string    `json:"message,omitempty"`
}

type Store interface {
	Append(ctx context.Context, evt *Event) error
	Query(ctx context.Context, runID string, limit int) ([]*Event, error)
	List(ctx context.Context, limit int) ([]*Event, error)
	Close() error
}

type MemoryStore struct {
	mu     sync.RWMutex
	events []*Event
	nextID int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Append(_ context.Context, evt *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	evt.ID = s.nextID
	if evt.Time.IsZero() {
		evt.Time = time.Now().UTC()
	}
	s.events = append(s.events, evt)
	return nil
}

func (s *MemoryStore) Query(_ context.Context, runID string, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Event
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].RunID == runID {
			result = append(result, s.events[i])
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *MemoryStore) List(_ context.Context, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	result := make([]*Event, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.events[len(s.events)-1-i]
	}
	return result, nil
}

func (s *MemoryStore) Close() error {
	return nil
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(ctx context.Context, connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Print("audit: connected to postgres")
	return &PostgresStore{db: db}, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_events (
		id BIGSERIAL PRIMARY KEY,
		time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		type TEXT NOT NULL,
		run_id TEXT NOT NULL DEFAULT '',
		step_id TEXT NOT NULL DEFAULT '',
		agent_id TEXT NOT NULL DEFAULT '',
		payload JSONB DEFAULT '{}',
		message TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_audit_run_id ON audit_events(run_id);
	CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_events(time DESC);
	`
	_, err := db.ExecContext(ctx, schema)
	return err
}

func (s *PostgresStore) Append(ctx context.Context, evt *Event) error {
	payload, err := json.Marshal(evt.Payload)
	if err != nil {
		payload = []byte("{}")
	}

	err = s.db.QueryRowContext(ctx,
		`INSERT INTO audit_events (type, run_id, step_id, agent_id, payload, message)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, time`,
		evt.Type, evt.RunID, evt.StepID, evt.AgentID, payload, evt.Message,
	).Scan(&evt.ID, &evt.Time)
	return err
}

func (s *PostgresStore) Query(ctx context.Context, runID string, limit int) ([]*Event, error) {
	query := `SELECT id, time, type, run_id, step_id, agent_id, payload, message
		FROM audit_events WHERE run_id = $1 ORDER BY time DESC`
	args := []any{runID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		var payload []byte
		if err := rows.Scan(&e.ID, &e.Time, &e.Type, &e.RunID, &e.StepID, &e.AgentID, &payload, &e.Message); err != nil {
			return nil, err
		}
		json.Unmarshal(payload, &e.Payload)
		events = append(events, &e)
	}
	return events, nil
}

func (s *PostgresStore) List(ctx context.Context, limit int) ([]*Event, error) {
	query := `SELECT id, time, type, run_id, step_id, agent_id, payload, message
		FROM audit_events ORDER BY time DESC`
	args := []any{}
	if limit > 0 {
		query += " LIMIT $1"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		var payload []byte
		if err := rows.Scan(&e.ID, &e.Time, &e.Type, &e.RunID, &e.StepID, &e.AgentID, &payload, &e.Message); err != nil {
			return nil, err
		}
		json.Unmarshal(payload, &e.Payload)
		events = append(events, &e)
	}
	return events, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func NewStoreFromEnv(ctx context.Context) Store {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Print("audit: no DATABASE_URL, using in-memory store")
		return NewMemoryStore()
	}

	store, err := NewPostgresStore(ctx, connStr)
	if err != nil {
		log.Printf("audit: postgres unavailable (%v), falling back to in-memory", err)
		return NewMemoryStore()
	}
	return store
}

func Log(store Store, evtType EventType, runID, stepID, agentID, message string, payload any) {
	evt := &Event{
		Time:    time.Now().UTC(),
		Type:    evtType,
		RunID:   runID,
		StepID:  stepID,
		AgentID: agentID,
		Message: message,
		Payload: payload,
	}
	if err := store.Append(context.Background(), evt); err != nil {
		log.Printf("audit: append error: %v", err)
	}
}
