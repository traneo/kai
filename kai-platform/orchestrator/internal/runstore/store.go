package runstore

import (
	"context"
	"log"
	"os"

	"kaiplatform.com/orchestrator/internal/workflow"
)

type Store interface {
	Save(ctx context.Context, run *workflow.Run) error
	LoadAll(ctx context.Context) ([]*workflow.Run, error)
	Close() error
}

type noopStore struct{}

func (noopStore) Save(_ context.Context, _ *workflow.Run) error { return nil }
func (noopStore) LoadAll(_ context.Context) ([]*workflow.Run, error) { return nil, nil }
func (noopStore) Close() error { return nil }

func NewStoreFromEnv(ctx context.Context) Store {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Print("run store: DATABASE_URL not set, using noop store")
		return noopStore{}
	}

	store, err := NewPostgresStore(ctx, connStr)
	if err != nil {
		log.Printf("run store: postgres unavailable: %v, using noop store", err)
		return noopStore{}
	}

	return store
}
