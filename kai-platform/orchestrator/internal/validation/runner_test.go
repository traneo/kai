package validation

import (
	"context"
	"testing"
)

func TestRunner_RegisterAndRun(t *testing.T) {
	r := NewRunner()
	r.Register(NewExitCodeGate())

	ctx := &Context{Context: context.Background(), ExitCode: 0}
	results := r.Run(ctx, []string{"exit_zero"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Failed() {
		t.Errorf("expected passed, got %s", results[0].Status)
	}
}

func TestRunner_UnregisteredGate(t *testing.T) {
	r := NewRunner()
	ctx := &Context{Context: context.Background(), ExitCode: 0}
	results := r.Run(ctx, []string{"nonexistent"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkipped {
		t.Errorf("expected skipped, got %s", results[0].Status)
	}
}

func TestRunner_MultipleGates(t *testing.T) {
	r := NewRunner()
	r.RegisterAll(NewExitCodeGate(), NewApprovalGate(false))

	ctx := &Context{Context: context.Background(), ExitCode: 0}
	results := r.Run(ctx, []string{"exit_zero", "approval"})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Failed() {
		t.Errorf("gate 1: expected passed, got %s", results[0].Status)
	}
	if results[1].Status != StatusSkipped {
		t.Errorf("gate 2: expected skipped, got %s", results[1].Status)
	}
}

func TestRunner_AllPassed(t *testing.T) {
	r := NewRunner()
	results := []*Result{
		{Gate: TypeExitCode, Status: StatusPassed},
		{Gate: TypeDiffReview, Status: StatusPassed},
	}
	if !r.AllPassed(results) {
		t.Error("expected all passed")
	}
}

func TestRunner_NotAllPassed(t *testing.T) {
	r := NewRunner()
	results := []*Result{
		{Gate: TypeExitCode, Status: StatusPassed},
		{Gate: TypeDiffReview, Status: StatusFailed},
	}
	if r.AllPassed(results) {
		t.Error("expected not all passed")
	}
}

func TestRunner_FailedGates(t *testing.T) {
	r := NewRunner()
	results := []*Result{
		{Gate: TypeExitCode, Status: StatusPassed},
		{Gate: TypeLint, Status: StatusFailed},
		{Gate: TypeTests, Status: StatusFailed},
	}
	failed := r.FailedGates(results)
	if len(failed) != 2 {
		t.Fatalf("expected 2 failed, got %d: %v", len(failed), failed)
	}
}

func TestRunner_Summary(t *testing.T) {
	r := NewRunner()
	results := []*Result{
		{Gate: TypeExitCode, Status: StatusPassed, Message: "ok"},
		{Gate: TypeLint, Status: StatusFailed, Message: "lint errors"},
	}
	s := r.Summary(results)
	if s != "exit_zero=passed, lint=failed" {
		t.Errorf("unexpected summary: %s", s)
	}
}
