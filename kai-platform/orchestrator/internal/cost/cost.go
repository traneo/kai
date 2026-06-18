package cost

import (
	"context"
	"log"
	"sync"
	"time"
)

type TokenUsage struct {
	Prompt     int64 `json:"prompt"`
	Completion int64 `json:"completion"`
	Total      int64 `json:"total"`
}

type StepCost struct {
	StepID      string     `json:"step_id"`
	RunID       string     `json:"run_id"`
	AgentID     string     `json:"agent_id"`
	TokenUsage  TokenUsage `json:"token_usage"`
	DurationMs  int64      `json:"duration_ms"`
	CompletedAt time.Time  `json:"completed_at"`
}

type RunCost struct {
	RunID       string      `json:"run_id"`
	Project     string      `json:"project"`
	Steps       []*StepCost `json:"steps"`
	TotalTokens int64       `json:"total_tokens"`
	DurationMs  int64       `json:"duration_ms"`
}

type Tracker struct {
	mu    sync.RWMutex
	steps []*StepCost
	runs  map[string]*RunCost
}

func NewTracker() *Tracker {
	return &Tracker{
		runs: make(map[string]*RunCost),
	}
}

func (t *Tracker) Record(runID, stepID, agentID, project string, promptTokens, completionTokens, durationMs int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	sc := &StepCost{
		StepID:  stepID,
		RunID:   runID,
		AgentID: agentID,
		TokenUsage: TokenUsage{
			Prompt:     promptTokens,
			Completion: completionTokens,
			Total:      promptTokens + completionTokens,
		},
		DurationMs:  durationMs,
		CompletedAt: time.Now().UTC(),
	}
	t.steps = append(t.steps, sc)

	rc, exists := t.runs[runID]
	if !exists {
		rc = &RunCost{RunID: runID, Project: project}
		t.runs[runID] = rc
	}
	rc.Steps = append(rc.Steps, sc)
	rc.TotalTokens += promptTokens + completionTokens
	rc.DurationMs += durationMs

	log.Printf("tracker: run=%s step=%s tokens=%d/%d duration=%dms",
		runID, stepID, promptTokens, completionTokens, durationMs)
}

func (t *Tracker) GetRunCost(runID string) *RunCost {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.runs[runID]
}

func (t *Tracker) AllRuns() []*RunCost {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*RunCost, 0, len(t.runs))
	for _, rc := range t.runs {
		result = append(result, rc)
	}
	return result
}

type AggregatedStats struct {
	TotalRuns        int     `json:"total_runs"`
	TotalSteps       int     `json:"total_steps"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalPromptToken int64   `json:"total_prompt_tokens"`
	TotalCompToken   int64   `json:"total_completion_tokens"`
	TotalDurationMs  int64   `json:"total_duration_ms"`
	AgentUsage       map[string]*AgentStat `json:"agent_usage"`
}

type AgentStat struct {
	AgentID         string `json:"agent_id"`
	StepsDone       int    `json:"steps_done"`
	TotalTokens     int64  `json:"total_tokens"`
	TotalDurationMs int64  `json:"total_duration_ms"`
}

func (t *Tracker) Stats() *AggregatedStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	s := &AggregatedStats{
		AgentUsage: make(map[string]*AgentStat),
	}
	for _, sc := range t.steps {
		s.TotalSteps++
		s.TotalTokens += sc.TokenUsage.Total
		s.TotalPromptToken += sc.TokenUsage.Prompt
		s.TotalCompToken += sc.TokenUsage.Completion
		s.TotalDurationMs += sc.DurationMs

		st, ok := s.AgentUsage[sc.AgentID]
		if !ok {
			st = &AgentStat{AgentID: sc.AgentID}
			s.AgentUsage[sc.AgentID] = st
		}
		st.StepsDone++
		st.TotalTokens += sc.TokenUsage.Total
		st.TotalDurationMs += sc.DurationMs
	}
	s.TotalRuns = len(t.runs)
	return s
}

func (t *Tracker) RecordFromResult(ctx context.Context, runID, stepID, agentID, project string, promptTokens, completionTokens, durationMs int64) {
	go t.Record(runID, stepID, agentID, project, promptTokens, completionTokens, durationMs)
}
