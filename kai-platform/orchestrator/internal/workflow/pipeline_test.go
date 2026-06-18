package workflow

import "testing"

func TestRequiresApproval(t *testing.T) {
	tests := []struct {
		approval string
		want     bool
	}{
		{"required", true},
		{"optional", false},
		{"", false},
	}
	for _, tt := range tests {
		s := Step{ID: "t", Prompt: "t", Approval: tt.approval}
		if s.RequiresApproval() != tt.want {
			t.Errorf("approval=%q: got %v, want %v", tt.approval, s.RequiresApproval(), tt.want)
		}
	}
}

func TestValidationGateConstants(t *testing.T) {
	if GateExitCode != "exit_zero" {
		t.Errorf("unexpected GateExitCode: %s", GateExitCode)
	}
	if GateLint != "lint" {
		t.Errorf("unexpected GateLint: %s", GateLint)
	}
	if GateTypecheck != "typecheck" {
		t.Errorf("unexpected GateTypecheck: %s", GateTypecheck)
	}
	if GateTests != "tests" {
		t.Errorf("unexpected GateTests: %s", GateTests)
	}
	if GateDiffReview != "diff_review" {
		t.Errorf("unexpected GateDiffReview: %s", GateDiffReview)
	}
}

func TestNewRun_InvalidPipeline(t *testing.T) {
	_, err := NewRun("fail", &Pipeline{Steps: []Step{
		{ID: "a", Prompt: "a", DependsOn: []string{"nonexistent"}},
	}})
	if err == nil {
		t.Fatal("expected error for invalid pipeline")
	}
}

func TestRun_CancelAlreadyRunning(t *testing.T) {
	run, _ := NewRun("r", samplePipeline())
	run.Start()
	run.StartStep("init")

	run.Cancel()

	if run.StepStates["init"].Status != StepCancelled {
		t.Errorf("running step should be cancelled, got %s", run.StepStates["init"].Status)
	}
}

func TestRun_ValidateFailureExhaustsRetries(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "a", Prompt: "a", Policy: Policy{MaxRetries: 1}},
		},
	}
	run, _ := NewRun("r", p)
	run.Start()
	run.StartStep("a")
	run.CompleteStep("a", true, "")
	run.ValidateStep("a", false, "lint failed")

	if run.StepStates["a"].Status != StepPending {
		t.Fatalf("expected pending for retry, got %s", run.StepStates["a"].Status)
	}

	run.StartStep("a")
	run.CompleteStep("a", true, "")
	run.ValidateStep("a", false, "still failing")

	if run.StepStates["a"].Status != StepFailed {
		t.Errorf("expected failed, got %s", run.StepStates["a"].Status)
	}
}

func TestRun_UpdateTimestamps(t *testing.T) {
	run, _ := NewRun("r", samplePipeline())
	run.Start()

	if run.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
	if run.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	run.StartStep("init")
	if run.StepStates["init"].StartedAt == nil || run.StepStates["init"].StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}

	run.CompleteStep("init", true, "")
	if run.StepStates["init"].CompletedAt == nil || run.StepStates["init"].CompletedAt.IsZero() {
		t.Error("expected non-zero CompletedAt")
	}
}

func TestRun_AssignAndTrackAgent(t *testing.T) {
	run, _ := NewRun("r", samplePipeline())
	run.Start()
	run.StartStep("init")
	run.StepStates["init"].AssignedTo = "agent-42"

	if run.StepStates["init"].AssignedTo != "agent-42" {
		t.Errorf("expected agent-42, got %s", run.StepStates["init"].AssignedTo)
	}
}
