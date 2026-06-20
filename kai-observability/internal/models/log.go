package models

import "time"

type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelDebug LogLevel = "debug"
)

func ValidLevel(s string) (LogLevel, bool) {
	switch LogLevel(s) {
	case LevelInfo, LevelWarn, LevelError, LevelDebug:
		return LogLevel(s), true
	default:
		return LevelInfo, false
	}
}

type LogEntry struct {
	ID         string            `json:"id"`
	Service    string            `json:"service"`
	Level      LogLevel          `json:"level"`
	Message    string            `json:"message"`
	Timestamp  int64             `json:"timestamp"`
	RunID      string            `json:"run_id,omitempty"`
	StepID     string            `json:"step_id,omitempty"`
	MissionID  string            `json:"mission_id,omitempty"`
	AgentID    string            `json:"agent_id,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
	ReceivedAt time.Time         `json:"received_at"`
}

type QueryFilter struct {
	Service   string    `json:"service,omitempty"`
	Level     LogLevel  `json:"level,omitempty"`
	From      time.Time `json:"from,omitempty"`
	To        time.Time `json:"to,omitempty"`
	Search    string    `json:"search,omitempty"`
	RunID     string    `json:"run_id,omitempty"`
	StepID    string    `json:"step_id,omitempty"`
	MissionID string    `json:"mission_id,omitempty"`
	AgentID   string    `json:"agent_id,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

type RunSummary struct {
	RunID      string   `json:"run_id"`
	EntryCount int      `json:"entry_count"`
	StartTime  int64    `json:"start_time"`
	EndTime    int64    `json:"end_time"`
	Services   []string `json:"services"`
	Steps      []string `json:"steps"`
}
