package workflow

import (
	"fmt"
	"sync"
	"time"
)

type PipelineStatus string

const (
	PipelinePending   PipelineStatus = "pending"
	PipelineRunning   PipelineStatus = "running"
	PipelineCompleted PipelineStatus = "completed"
	PipelineFailed    PipelineStatus = "failed"
	PipelineCancelled PipelineStatus = "cancelled"
)

type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepReady      StepStatus = "ready"
	StepRunning    StepStatus = "running"
	StepValidating StepStatus = "validating"
	StepPassed     StepStatus = "passed"
	StepFailed     StepStatus = "failed"
	StepBlocked    StepStatus = "blocked"
	StepCancelled  StepStatus = "cancelled"
)

type Run struct {
	mu         sync.Mutex
	ID         string
	PipelineID string
	Pipeline   *Pipeline
	DAG        *DAG
	Status     PipelineStatus
	StepStates map[string]*StepState
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Error      string
	RawYAML    string
	OutputURL  string
	OutputSHA  string
}

type StepState struct {
	Step        Step
	Status      StepStatus
	Retries     int
	NextRetryAt *time.Time
	AssignedTo  string
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       string
	GateResults []GateResult
	Diff        string
	BeforeSHA   string
}

func NewRun(id string, p *Pipeline) (*Run, error) {
	dag, err := BuildDAG(p)
	if err != nil {
		return nil, fmt.Errorf("build dag: %w", err)
	}

	steps := make(map[string]*StepState, len(p.Steps))
	for _, s := range p.Steps {
		steps[s.ID] = &StepState{
			Step:   s,
			Status: StepPending,
		}
	}

	now := time.Now().UTC()
	return &Run{
		ID:         id,
		Pipeline:   p,
		DAG:        dag,
		Status:     PipelinePending,
		StepStates: steps,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// RestoreRun creates a Run from previously persisted state (recovered after restart).
func RestoreRun(id string, p *Pipeline, rawYAML string, states map[string]*StepState) (*Run, error) {
	dag, err := BuildDAG(p)
	if err != nil {
		return nil, fmt.Errorf("build dag: %w", err)
	}

	return &Run{
		ID:         id,
		Pipeline:   p,
		DAG:        dag,
		StepStates: states,
		RawYAML:    rawYAML,
	}, nil
}

func (r *Run) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Status = PipelineRunning
	r.UpdatedAt = time.Now().UTC()
}

func (r *Run) NextReadySteps() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	ready := r.DAG.ReadySteps(r.stepStatusMapLocked())
	now := time.Now()
	filtered := make([]string, 0, len(ready))

	for _, id := range ready {
		if state, ok := r.StepStates[id]; ok {
			if state.NextRetryAt != nil && state.NextRetryAt.After(now) {
				continue
			}
		}
		filtered = append(filtered, id)
	}

	return filtered
}

func (r *Run) CanStartStep(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.StepStates[id] != nil && r.StepStates[id].Status == StepPending && r.Status == PipelineRunning
}

func (r *Run) StartStep(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.StepStates[id]
	if !ok {
		return fmt.Errorf("step %q not found", id)
	}
	if state.Status != StepPending {
		return fmt.Errorf("step %q cannot start from status %s", id, state.Status)
	}

	allDepsPassed := true
	for _, dep := range state.Step.DependsOn {
		if r.StepStates[dep].Status != StepPassed {
			allDepsPassed = false
			break
		}
	}
	if !allDepsPassed {
		return fmt.Errorf("step %q has unmet dependencies", id)
	}

	now := time.Now().UTC()
	state.Status = StepRunning
	state.StartedAt = &now
	state.Error = ""
	r.UpdatedAt = now
	return nil
}

func (r *Run) CompleteStep(id string, success bool, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.StepStates[id]
	if !ok {
		return
	}

	if state.Status != StepRunning {
		return
	}

	now := time.Now().UTC()
	state.CompletedAt = &now
	r.UpdatedAt = now

	if success {
		state.Status = StepValidating
		state.Error = ""
	} else {
		state.Error = errMsg
		if state.Retries < state.Step.Policy.MaxRetries {
			state.Retries++
			delay := state.Step.Policy.RetryDelay(state.Retries - 1)
			next := now.Add(delay)
			state.NextRetryAt = &next
			state.Status = StepPending
		} else {
			state.Status = StepFailed
		}
	}

	r.evaluatePipelineStatusLocked()
}

func (r *Run) ValidateStep(id string, passed bool, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.StepStates[id]
	if !ok {
		return
	}

	if passed {
		state.Status = StepPassed
		state.NextRetryAt = nil
		state.Error = ""
	} else {
		if state.Retries < state.Step.Policy.MaxRetries {
			state.Retries++
			delay := state.Step.Policy.RetryDelay(state.Retries - 1)
			next := time.Now().UTC().Add(delay)
			state.NextRetryAt = &next
			state.Status = StepPending
		} else {
			state.Status = StepFailed
			state.NextRetryAt = nil
			state.Error = errMsg
		}
	}

	r.UpdatedAt = time.Now().UTC()
	r.evaluatePipelineStatusLocked()
}

func (r *Run) BlockStep(id string, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state, ok := r.StepStates[id]; ok {
		state.Status = StepBlocked
		state.Error = reason
		r.UpdatedAt = time.Now().UTC()
		r.evaluatePipelineStatusLocked()
	}
}

func (r *Run) ResetStep(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.StepStates[id]
	if !ok || state.Status != StepFailed {
		return
	}

	state.Status = StepPending
	state.Retries = 0
	state.Error = ""
	state.NextRetryAt = nil
	r.UpdatedAt = time.Now().UTC()

	// If this was the only failed step, transition back to running.
	hasFailed := false
	for _, s := range r.StepStates {
		if s.Status == StepFailed {
			hasFailed = true
			break
		}
	}
	if !hasFailed && r.Status == PipelineFailed {
		r.Status = PipelineRunning
	}

	r.evaluatePipelineStatusLocked()
}

func (r *Run) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Status = PipelineCancelled
	for _, state := range r.StepStates {
		if state.Status == StepPending || state.Status == StepReady || state.Status == StepRunning {
			state.Status = StepCancelled
		}
	}
	r.UpdatedAt = time.Now().UTC()
}

func (r *Run) SetGateResults(id string, gates []GateResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.StepStates[id]; ok {
		state.GateResults = gates
	}
}

func (r *Run) SetStepDiff(id string, diff string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.StepStates[id]; ok {
		state.Diff = diff
	}
}

func (r *Run) SetStepBeforeSHA(id string, sha string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.StepStates[id]; ok {
		state.BeforeSHA = sha
	}
}

func (r *Run) Snapshot() *Run {
	r.mu.Lock()
	defer r.mu.Unlock()

	steps := make(map[string]*StepState, len(r.StepStates))
	for id, s := range r.StepStates {
		gates := make([]GateResult, len(s.GateResults))
		copy(gates, s.GateResults)
		var startedAt *time.Time
		if s.StartedAt != nil {
			t := *s.StartedAt
			startedAt = &t
		}
		var completedAt *time.Time
		if s.CompletedAt != nil {
			t := *s.CompletedAt
			completedAt = &t
		}
		var nextRetryAt *time.Time
		if s.NextRetryAt != nil {
			t := *s.NextRetryAt
			nextRetryAt = &t
		}
		steps[id] = &StepState{
			Step:        s.Step,
			Status:      s.Status,
			Retries:     s.Retries,
			NextRetryAt: nextRetryAt,
			AssignedTo:  s.AssignedTo,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			Error:       s.Error,
			GateResults: gates,
			Diff:        s.Diff,
			BeforeSHA:   s.BeforeSHA,
		}
	}
	return &Run{
		ID:         r.ID,
		Pipeline:   r.Pipeline,
		DAG:        r.DAG,
		Status:     r.Status,
		StepStates: steps,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		Error:      r.Error,
		RawYAML:    r.RawYAML,
		OutputURL:  r.OutputURL,
		OutputSHA:  r.OutputSHA,
	}
}

func (r *Run) evaluatePipelineStatusLocked() {
	for _, state := range r.StepStates {
		if state.Status == StepFailed {
			r.Status = PipelineFailed
			return
		}
	}

	allPassed := true
	for _, state := range r.StepStates {
		if state.Status != StepPassed {
			allPassed = false
			break
		}
	}
	if allPassed {
		r.Status = PipelineCompleted
	}
}

func (r *Run) stepStatusMapLocked() map[string]StepStatus {
	m := make(map[string]StepStatus, len(r.StepStates))
	for id, state := range r.StepStates {
		m[id] = state.Status
	}
	return m
}

func (r *Run) HasBlockedStep() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, state := range r.StepStates {
		if state.Status == StepBlocked {
			return true
		}
	}
	return false
}
