package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"kaiplatform.com/observability/internal/models"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS log_entries (
			id          TEXT PRIMARY KEY,
			service     TEXT NOT NULL,
			level       TEXT NOT NULL,
			message     TEXT NOT NULL,
			ts          BIGINT NOT NULL,
			run_id      TEXT DEFAULT '',
			step_id     TEXT DEFAULT '',
			mission_id  TEXT DEFAULT '',
			agent_id    TEXT DEFAULT '',
			metadata    JSONB DEFAULT '{}',
			received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_log_ts     ON log_entries(ts DESC);
		CREATE INDEX IF NOT EXISTS idx_log_svc_ts ON log_entries(service, ts DESC);
		CREATE INDEX IF NOT EXISTS idx_log_lvl_ts ON log_entries(level, ts DESC);
	`)
	return err
}

func (s *PostgresStore) Append(_ context.Context, entries []models.LogEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO log_entries (id, service, level, message, ts, run_id, step_id, mission_id, agent_id, metadata, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		meta, _ := json.Marshal(e.Metadata)
		if _, err := stmt.Exec(e.ID, e.Service, string(e.Level), e.Message, e.Timestamp,
			e.RunID, e.StepID, e.MissionID, e.AgentID, meta, e.ReceivedAt); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) Query(_ context.Context, filter models.QueryFilter) ([]models.LogEntry, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var where []string
	var args []any
	n := 1

	if filter.Service != "" {
		where = append(where, fmt.Sprintf("service = $%d", n))
		args = append(args, filter.Service)
		n++
	}
	if filter.Level != "" {
		where = append(where, fmt.Sprintf("level = $%d", n))
		args = append(args, string(filter.Level))
		n++
	}
	if !filter.From.IsZero() {
		where = append(where, fmt.Sprintf("ts >= $%d", n))
		args = append(args, filter.From.UnixMilli())
		n++
	}
	if !filter.To.IsZero() {
		where = append(where, fmt.Sprintf("ts <= $%d", n))
		args = append(args, filter.To.UnixMilli())
		n++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("LOWER(message) LIKE $%d", n))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		n++
	}
	if filter.RunID != "" {
		where = append(where, fmt.Sprintf("run_id = $%d", n))
		args = append(args, filter.RunID)
		n++
	}
	if filter.StepID != "" {
		where = append(where, fmt.Sprintf("step_id = $%d", n))
		args = append(args, filter.StepID)
		n++
	}
	if filter.MissionID != "" {
		where = append(where, fmt.Sprintf("mission_id = $%d", n))
		args = append(args, filter.MissionID)
		n++
	}
	if filter.AgentID != "" {
		where = append(where, fmt.Sprintf("agent_id = $%d", n))
		args = append(args, filter.AgentID)
		n++
	}

	query := "SELECT id, service, level, message, ts, COALESCE(run_id,''), COALESCE(step_id,''), COALESCE(mission_id,''), COALESCE(agent_id,''), metadata, received_at FROM log_entries"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY ts DESC LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var entries []models.LogEntry
	for rows.Next() {
		var e models.LogEntry
		var level string
		var meta []byte
		if err := rows.Scan(&e.ID, &e.Service, &level, &e.Message, &e.Timestamp,
			&e.RunID, &e.StepID, &e.MissionID, &e.AgentID, &meta, &e.ReceivedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		e.Level = models.LogLevel(level)
		json.Unmarshal(meta, &e.Metadata)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *PostgresStore) GetByID(_ context.Context, id string) (*models.LogEntry, error) {
	var e models.LogEntry
	var level string
	var meta []byte
	err := s.db.QueryRow(
		`SELECT id, service, level, message, ts, COALESCE(run_id,''), COALESCE(step_id,''), COALESCE(mission_id,''), COALESCE(agent_id,''), metadata, received_at FROM log_entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.Service, &level, &e.Message, &e.Timestamp, &e.RunID, &e.StepID, &e.MissionID, &e.AgentID, &meta, &e.ReceivedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}
	e.Level = models.LogLevel(level)
	json.Unmarshal(meta, &e.Metadata)
	return &e, nil
}

func (s *PostgresStore) RunSummaries(_ context.Context) ([]models.RunSummary, error) {
	rows, err := s.db.Query(`
		SELECT run_id, COUNT(*) as entry_count, MIN(ts) as start_time, MAX(ts) as end_time,
			array_agg(DISTINCT service) as services,
			array_agg(DISTINCT step_id) FILTER (WHERE step_id != '') as steps
		FROM log_entries
		WHERE run_id != ''
		GROUP BY run_id
		ORDER BY MAX(ts) DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("run summaries: %w", err)
	}
	defer rows.Close()

	var summaries []models.RunSummary
	for rows.Next() {
		var rs models.RunSummary
		var services, steps []string
		if err := rows.Scan(&rs.RunID, &rs.EntryCount, &rs.StartTime, &rs.EndTime, pqArray(&services), pqArray(&steps)); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		if services == nil {
			services = []string{}
		}
		if steps == nil {
			steps = []string{}
		}
		rs.Services = services
		rs.Steps = steps
		summaries = append(summaries, rs)
	}
	if summaries == nil {
		summaries = []models.RunSummary{}
	}
	return summaries, rows.Err()
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// pqArray scans a PostgreSQL array into a Go string slice.
func pqArray(dst *[]string) interface{} {
	return &pgArrayScanner{dst: dst}
}

type pgArrayScanner struct {
	dst *[]string
}

func (s *pgArrayScanner) Scan(src interface{}) error {
	if src == nil {
		*s.dst = []string{}
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return s.scanString(string(v))
	case string:
		return s.scanString(v)
	default:
		*s.dst = []string{}
		return nil
	}
}

func (s *pgArrayScanner) scanString(raw string) error {
	if len(raw) < 2 || raw[0] != '{' || raw[len(raw)-1] != '}' {
		*s.dst = []string{}
		return nil
	}
	inner := raw[1 : len(raw)-1]
	if inner == "" {
		*s.dst = []string{}
		return nil
	}
	var result []string
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"")
		if part != "" {
			result = append(result, part)
		}
	}
	*s.dst = result
	return nil
}