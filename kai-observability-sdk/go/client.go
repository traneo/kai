// Package sdkgokit provides a Go client for the kai-observability log API.
// It buffers log entries and flushes them asynchronously in batches.
package sdkgokit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelDebug LogLevel = "debug"
)

type Field struct {
	Key   string
	Value any
}

type entry struct {
	Service   string         `json:"service"`
	Level     LogLevel       `json:"level"`
	Message   string         `json:"message"`
	Timestamp int64          `json:"timestamp,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	StepID    string         `json:"step_id,omitempty"`
	MissionID string         `json:"mission_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Logger struct {
	endpoint   string
	service    string
	batchSize  int
	flushEvery time.Duration
	queueCap   int

	queue chan entry
	http  *http.Client
	done  chan struct{}
	wg    sync.WaitGroup
	once  sync.Once

	mu       sync.RWMutex
	runID    string
	stepID   string
	missionID string
	agentID  string
}

type Option func(*Logger)

func WithBatchSize(n int) Option {
	return func(l *Logger) { l.batchSize = n }
}

func WithFlushInterval(d time.Duration) Option {
	return func(l *Logger) { l.flushEvery = d }
}

func WithQueueCap(n int) Option {
	return func(l *Logger) { l.queueCap = n }
}

func New(endpoint, service string, opts ...Option) *Logger {
	l := &Logger{
		endpoint:   endpoint,
		service:    service,
		batchSize:  50,
		flushEvery: time.Second,
		queueCap:   10000,
		queue:      nil,
		http:       &http.Client{Timeout: 5 * time.Second},
		done:       make(chan struct{}),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Logger) start() {
	l.once.Do(func() {
		l.queue = make(chan entry, l.queueCap)
		l.wg.Add(1)
		go l.flushLoop()
	})
}

func (l *Logger) log(level LogLevel, msg string, fields []Field) {
	l.start()

	meta := make(map[string]any, len(fields))
	for _, f := range fields {
		meta[f.Key] = f.Value
	}

	l.mu.RLock()
	e := entry{
		Service:   l.service,
		Level:     level,
		Message:   msg,
		Timestamp: time.Now().UnixMilli(),
		RunID:     l.runID,
		StepID:    l.stepID,
		MissionID: l.missionID,
		AgentID:   l.agentID,
		Metadata:  meta,
	}
	l.mu.RUnlock()

	select {
	case l.queue <- e:
	default:
	}
}

func (l *Logger) Info(msg string, fields ...Field)  { l.log(LevelInfo, msg, fields) }
func (l *Logger) Warn(msg string, fields ...Field)  { l.log(LevelWarn, msg, fields) }
func (l *Logger) Error(msg string, fields ...Field) { l.log(LevelError, msg, fields) }
func (l *Logger) Debug(msg string, fields ...Field) { l.log(LevelDebug, msg, fields) }

// WithRunID returns a scoped logger that includes the run_id in every entry.
func (l *Logger) WithRunID(runID string) *Logger { return l.withLock(func(l *Logger) { l.runID = runID }) }

// WithStepID returns a scoped logger that includes the step_id in every entry.
func (l *Logger) WithStepID(stepID string) *Logger { return l.withLock(func(l *Logger) { l.stepID = stepID }) }

// WithMissionID returns a scoped logger that includes the mission_id in every entry.
func (l *Logger) WithMissionID(missionID string) *Logger { return l.withLock(func(l *Logger) { l.missionID = missionID }) }

// WithAgentID returns a scoped logger that includes the agent_id in every entry.
func (l *Logger) WithAgentID(agentID string) *Logger { return l.withLock(func(l *Logger) { l.agentID = agentID }) }

func (l *Logger) withLock(fn func(*Logger)) *Logger {
	cp := &Logger{
		endpoint:   l.endpoint,
		service:    l.service,
		batchSize:  l.batchSize,
		flushEvery: l.flushEvery,
		queueCap:   l.queueCap,
		queue:      l.queue,
		http:       l.http,
		done:       l.done,
		once:       l.once,
		runID:      l.runID,
		stepID:     l.stepID,
		missionID:  l.missionID,
		agentID:    l.agentID,
	}
	fn(cp)
	return cp
}

func (l *Logger) Close() {
	if l.queue == nil {
		return
	}
	close(l.done)
	l.wg.Wait()
	l.flush(l.drain())
}

func (l *Logger) flushLoop() {
	ticker := time.NewTicker(l.flushEvery)
	defer ticker.Stop()
	defer l.wg.Done()

	var buf []entry
	for {
		select {
		case <-l.done:
			return
		case e := <-l.queue:
			buf = append(buf, e)
			if len(buf) >= l.batchSize {
				l.flush(buf)
				buf = buf[:0]
			}
		case <-ticker.C:
			if len(buf) > 0 {
				l.flush(buf)
				buf = buf[:0]
			}
		}
	}
}

func (l *Logger) drain() []entry {
	n := len(l.queue)
	buf := make([]entry, 0, n)
	for i := 0; i < n; i++ {
		select {
		case e := <-l.queue:
			buf = append(buf, e)
		default:
			return buf
		}
	}
	return buf
}

func (l *Logger) flush(entries []entry) {
	if len(entries) == 0 {
		return
	}
	body, _ := json.Marshal(map[string]any{"entries": entries})
	resp, err := l.http.Post(l.endpoint+"/api/v1/logs/batch", "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	resp.Body.Close()
}

func Fields(m map[string]any) []Field {
	f := make([]Field, 0, len(m))
	for k, v := range m {
		f = append(f, Field{Key: k, Value: v})
	}
	return f
}

func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

var _ = fmt.Sprintf
