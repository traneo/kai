package store

import (
	"context"
	"kaiplatform.com/observability/internal/models"
)

type Store interface {
	Append(ctx context.Context, entries []models.LogEntry) error
	Query(ctx context.Context, filter models.QueryFilter) ([]models.LogEntry, error)
	GetByID(ctx context.Context, id string) (*models.LogEntry, error)
	RunSummaries(ctx context.Context) ([]models.RunSummary, error)
	Close() error
}
