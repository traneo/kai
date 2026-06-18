package workflow

import (
	"testing"
	"time"
)

func TestNewRun(t *testing.T) {
	run, err := NewRun("run-1", samplePipeline())
	if err != nil {
		t.Fatalf("NewRun failed: %v", err)
	}

	if run.Status != PipelinePending {
		t.Errorf("expected pending, got %s", run.Status)
	}

	if len(run.StepStates) != 3 {
		t.Fatalf("expected 3 step states, got %d", len(run.StepStates))
	}

	for id, state := range run.StepStates {
		if state.Status != StepPending {
			t.Errorf("step %s: expected pending, got %s", id, state.Status)
		}
	}
}

func TestRun_NextReadySteps(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()

	ready := run.NextReadySteps()
	if len(ready) != 1 || ready[0] != "init" {
		t.Fatalf("expected [init], got %v", ready)
	}
}

func TestRun_StartAndCompleteStep(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()

	if err := run.StartStep("init"); err != nil {
		t.Fatalf("StartStep failed: %v", err)
	}

	if run.StepStates["init"].Status != StepRunning {
		t.Errorf("expected running, got %s", run.StepStates["init"].Status)
	}

	run.CompleteStep("init", true, "")
	if run.StepStates["init"].Status != StepValidating {
		t.Errorf("expected validating, got %s", run.StepStates["init"].Status)
	}
}

func TestRun_ValidateStep(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()
	run.StartStep("init")
	run.CompleteStep("init", true, "")
	run.ValidateStep("init", true, "")

	if run.StepStates["init"].Status != StepPassed {
		t.Errorf("expected passed, got %s", run.StepStates["init"].Status)
	}
}

func TestRun_FullPipelineSuccess(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()

	for _, id := range []string{"init", "auth", "api"} {
		if err := run.StartStep(id); err != nil {
			t.Fatalf("start step %s: %v", id, err)
		}
		run.CompleteStep(id, true, "")
		run.ValidateStep(id, true, "")
	}

	if run.Status != PipelineCompleted {
		t.Errorf("expected completed, got %s", run.Status)
	}
}

func TestRun_StepFailureExhaustsRetries(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "init", Prompt: "init", Policy: Policy{MaxRetries: 2}},
		},
	}
	run, _ := NewRun("run-1", p)
	run.Start()

	run.StartStep("init")
	run.CompleteStep("init", false, "err") // retry 1
	if run.StepStates["init"].Retries != 1 {
		t.Fatalf("expected 1 retry, got %d", run.StepStates["init"].Retries)
	}

	run.StartStep("init")
	run.CompleteStep("init", false, "err") // retry 2
	if run.StepStates["init"].Retries != 2 {
		t.Fatalf("expected 2 retries, got %d", run.StepStates["init"].Retries)
	}

	run.StartStep("init")
	run.CompleteStep("init", false, "err") // exhausted
	if run.StepStates["init"].Status != StepFailed {
		t.Errorf("expected failed after retry exhaustion, got %s", run.StepStates["init"].Status)
	}
}

func TestRun_StepFailureNoRetries(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "init", Prompt: "init", Policy: Policy{MaxRetries: 0}},
		},
	}
	run, _ := NewRun("run-1", p)
	run.Start()
	run.StartStep("init")
	run.CompleteStep("init", false, "oops")

	if run.StepStates["init"].Status != StepFailed {
		t.Errorf("expected failed, got %s", run.StepStates["init"].Status)
	}
}

func TestRun_StepRetrySetsNextRetryAt(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "init", Prompt: "init", Policy: Policy{MaxRetries: 2, RetryDelaySeconds: 30}},
		},
	}
	run, _ := NewRun("run-1", p)
	run.Start()
	run.StartStep("init")

	run.CompleteStep("init", false, "err")
	if run.StepStates["init"].NextRetryAt == nil {
		t.Fatal("expected NextRetryAt to be set after retry")
	}
	if run.StepStates["init"].NextRetryAt.Before(time.Now()) {
		t.Error("expected NextRetryAt to be in the future")
	}
}

func TestRun_StepRetryAndRecover(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "init", Prompt: "init", Policy: Policy{MaxRetries: 3}},
		},
	}
	run, _ := NewRun("run-1", p)
	run.Start()
	run.StartStep("init")

	run.CompleteStep("init", false, "oops")
	if run.StepStates["init"].Status != StepPending {
		t.Errorf("expected pending for retry, got %s", run.StepStates["init"].Status)
	}

	run.StartStep("init")
	run.CompleteStep("init", true, "")
	if run.StepStates["init"].Status != StepValidating {
		t.Errorf("expected validating, got %s", run.StepStates["init"].Status)
	}
}

func TestRun_Cancel(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()
	run.StartStep("init")
	run.Cancel()

	if run.Status != PipelineCancelled {
		t.Errorf("expected cancelled, got %s", run.Status)
	}
}

func TestRun_BlockStep(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()
	run.BlockStep("init", "needs human review")

	if run.StepStates["init"].Status != StepBlocked {
		t.Errorf("expected blocked, got %s", run.StepStates["init"].Status)
	}

	if run.Status != PipelineRunning {
		t.Errorf("expected running, got %s", run.Status)
	}
}

func TestRun_CannotStartUnmetDependency(t *testing.T) {
	run, _ := NewRun("run-1", samplePipeline())
	run.Start()

	err := run.StartStep("auth")
	if err == nil {
		t.Fatal("expected error starting auth before init")
	}
}

func TestRun_HasBlockedStep(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "a", Prompt: "a"},
			{ID: "b", Prompt: "b", DependsOn: []string{"a"}},
			{ID: "c", Prompt: "c", DependsOn: []string{"a"}},
		},
	}
	run, _ := NewRun("run-blocked", p)
	run.Start()
	run.StartStep("a")

	if run.HasBlockedStep() {
		t.Error("expected no blocked step initially")
	}

	run.BlockStep("a", "awaiting approval")
	if !run.HasBlockedStep() {
		t.Error("expected HasBlockedStep true after blocking a")
	}

	run.BlockStep("b", "also blocked")
	if !run.HasBlockedStep() {
		t.Error("expected HasBlockedStep true with multiple blocked")
	}
}

func TestRun_HasBlockedStep_Passed(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "a", Prompt: "a"},
			{ID: "b", Prompt: "b"},
		},
	}
	run, _ := NewRun("run-passed", p)
	run.Start()
	run.StartStep("a")
	run.CompleteStep("a", true, "")
	run.ValidateStep("a", true, "")

	if run.HasBlockedStep() {
		t.Error("expected no blocked step when step passed")
	}
}

func TestRun_StatusString(t *testing.T) {
	tests := []struct {
		status PipelineStatus
		want   string
	}{
		{PipelinePending, "pending"},
		{PipelineRunning, "running"},
		{PipelineCompleted, "completed"},
		{PipelineFailed, "failed"},
		{PipelineCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("got %s, want %s", tt.status, tt.want)
		}
	}
}
