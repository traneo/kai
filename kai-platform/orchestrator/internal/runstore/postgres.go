package runstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"kaiplatform.com/orchestrator/internal/workflow"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(ctx context.Context, connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate postgres: %w", err)
	}

	log.Print("run store: connected to postgres")
	return &PostgresStore{db: db}, nil
}

type runRow struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Error      string          `json:"error"`
	RawYAML    string          `json:"raw_yaml"`
	OutputURL  string          `json:"output_url"`
	OutputSHA  string          `json:"output_sha"`
	StepStates json.RawMessage `json:"step_states"`
}

func stepStatesJSON(states map[string]*workflow.StepState) ([]byte, error) {
	type stepStateView struct {
		Step        workflow.Step       `json:"step"`
		Status      workflow.StepStatus `json:"status"`
		Retries     int                 `json:"retries"`
		NextRetryAt *time.Time          `json:"next_retry_at,omitempty"`
		AssignedTo  string              `json:"assigned_to"`
		StartedAt   *time.Time          `json:"started_at,omitempty"`
		CompletedAt *time.Time          `json:"completed_at,omitempty"`
		Error       string              `json:"error"`
		GateResults []workflow.GateResult `json:"gate_results,omitempty"`
		Diff        string              `json:"diff,omitempty"`
		BeforeSHA   string              `json:"before_sha,omitempty"`
	}
	view := make(map[string]*stepStateView, len(states))
	for id, s := range states {
		view[id] = &stepStateView{
			Step:        s.Step,
			Status:      s.Status,
			Retries:     s.Retries,
			NextRetryAt: s.NextRetryAt,
			AssignedTo:  s.AssignedTo,
			StartedAt:   s.StartedAt,
			CompletedAt: s.CompletedAt,
			Error:       s.Error,
			GateResults: s.GateResults,
			Diff:        s.Diff,
			BeforeSHA:   s.BeforeSHA,
		}
	}
	return json.Marshal(view)
}

func stepStatesFromJSON(data []byte) (map[string]*workflow.StepState, error) {
	var view map[string]struct {
		Step        workflow.Step        `json:"step"`
		Status      workflow.StepStatus  `json:"status"`
		Retries     int                  `json:"retries"`
		NextRetryAt *time.Time           `json:"next_retry_at,omitempty"`
		AssignedTo  string               `json:"assigned_to"`
		StartedAt   *time.Time           `json:"started_at,omitempty"`
		CompletedAt *time.Time           `json:"completed_at,omitempty"`
		Error       string               `json:"error"`
		GateResults []workflow.GateResult `json:"gate_results,omitempty"`
		Diff        string               `json:"diff,omitempty"`
		BeforeSHA   string               `json:"before_sha,omitempty"`
	}
	if err := json.Unmarshal(data, &view); err != nil {
		return nil, err
	}
	states := make(map[string]*workflow.StepState, len(view))
	for id, v := range view {
		states[id] = &workflow.StepState{
			Step:        v.Step,
			Status:      v.Status,
			Retries:     v.Retries,
			NextRetryAt: v.NextRetryAt,
			AssignedTo:  v.AssignedTo,
			StartedAt:   v.StartedAt,
			CompletedAt: v.CompletedAt,
			Error:       v.Error,
			GateResults: v.GateResults,
			Diff:        v.Diff,
			BeforeSHA:   v.BeforeSHA,
		}
	}
	return states, nil
}

func (s *PostgresStore) Save(ctx context.Context, run *workflow.Run) error {
	snap := run.Snapshot()
	states, err := stepStatesJSON(snap.StepStates)
	if err != nil {
		return fmt.Errorf("marshal step states: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO run_snapshots (id, status, created_at, updated_at, error, raw_yaml, output_url, output_sha, step_states)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			error = EXCLUDED.error,
			raw_yaml = EXCLUDED.raw_yaml,
			output_url = EXCLUDED.output_url,
			output_sha = EXCLUDED.output_sha,
			step_states = EXCLUDED.step_states
	`,
		snap.ID, string(snap.Status), snap.CreatedAt, snap.UpdatedAt,
		snap.Error, snap.RawYAML, snap.OutputURL, snap.OutputSHA, states,
	)
	if err != nil {
		return fmt.Errorf("upsert run snapshot: %w", err)
	}
	return nil
}

func (s *PostgresStore) LoadAll(ctx context.Context) ([]*workflow.Run, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, status, created_at, updated_at, error, raw_yaml, output_url, output_sha, step_states
		FROM run_snapshots
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query run snapshots: %w", err)
	}
	defer rows.Close()

	var runs []*workflow.Run
	for rows.Next() {
		var r runRow
		if err := rows.Scan(&r.ID, &r.Status, &r.CreatedAt, &r.UpdatedAt,
			&r.Error, &r.RawYAML, &r.OutputURL, &r.OutputSHA, &r.StepStates); err != nil {
			return nil, fmt.Errorf("scan run snapshot: %w", err)
		}

		pipeline, err := workflow.ParsePipelineBytes([]byte(r.RawYAML))
		if err != nil {
			log.Printf("run store: parse pipeline for run %s: %v, skipping", r.ID, err)
			continue
		}

		states, err := stepStatesFromJSON(r.StepStates)
		if err != nil {
			log.Printf("run store: parse step states for run %s: %v, skipping", r.ID, err)
			continue
		}

		run, err := workflow.RestoreRun(r.ID, pipeline, r.RawYAML, states)
		if err != nil {
			log.Printf("run store: restore run %s: %v, skipping", r.ID, err)
			continue
		}

		run.Status = workflow.PipelineStatus(r.Status)
		run.CreatedAt = r.CreatedAt
		run.UpdatedAt = r.UpdatedAt
		run.Error = r.Error
		run.OutputURL = r.OutputURL
		run.OutputSHA = r.OutputSHA

		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func migrate(ctx context.Context, db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS run_snapshots (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		error TEXT NOT NULL DEFAULT '',
		raw_yaml TEXT NOT NULL DEFAULT '',
		output_url TEXT NOT NULL DEFAULT '',
		output_sha TEXT NOT NULL DEFAULT '',
		step_states JSONB NOT NULL DEFAULT '{}'
	);
	`
	_, err := db.ExecContext(ctx, schema)
	return err
}
