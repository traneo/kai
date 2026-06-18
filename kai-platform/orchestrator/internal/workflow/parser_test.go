package workflow

import (
	"testing"
)

func TestParsePipelineBytes(t *testing.T) {
	yaml := `
version: 1
project: my-service
output:
  type: pr
  branch_prefix: feat/
steps:
  - id: scaffold
    prompt: "initialize project"
    validation:
      - exit_zero
  - id: implement
    prompt: "implement feature"
    depends_on:
      - scaffold
    policy:
      allowed_dirs:
        - src/
      max_retries: 2
      timeout_seconds: 300
    validation:
      - exit_zero
      - lint
      - tests
    approval: required
`

	p, err := ParsePipelineBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParsePipelineBytes failed: %v", err)
	}

	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}

	if p.Project != "my-service" {
		t.Errorf("expected my-service, got %s", p.Project)
	}

	if len(p.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(p.Steps))
	}

	if p.Steps[0].ID != "scaffold" {
		t.Errorf("expected scaffold, got %s", p.Steps[0].ID)
	}

	if p.Steps[0].Approval != "" {
		t.Errorf("expected empty approval, got %s", p.Steps[0].Approval)
	}

	if p.Steps[1].ID != "implement" {
		t.Errorf("expected implement, got %s", p.Steps[1].ID)
	}

	if len(p.Steps[1].DependsOn) != 1 || p.Steps[1].DependsOn[0] != "scaffold" {
		t.Errorf("expected depends_on [scaffold], got %v", p.Steps[1].DependsOn)
	}

	if p.Steps[1].Policy.MaxRetries != 2 {
		t.Errorf("expected max_retries 2, got %d", p.Steps[1].Policy.MaxRetries)
	}

	if p.Steps[1].Policy.TimeoutSeconds != 300 {
		t.Errorf("expected timeout 300, got %d", p.Steps[1].Policy.TimeoutSeconds)
	}

	if !p.Steps[1].RequiresApproval() {
		t.Errorf("expected requires approval")
	}

	if len(p.Steps[1].Validation) != 3 {
		t.Errorf("expected 3 validation gates, got %d", len(p.Steps[1].Validation))
	}
}

func TestParsePipelineBytes_EmptySteps(t *testing.T) {
	_, err := ParsePipelineBytes([]byte("steps: []"))
	if err == nil {
		t.Fatal("expected error for empty steps")
	}
}

func TestParsePipelineBytes_MissingStepID(t *testing.T) {
	_, err := ParsePipelineBytes([]byte(`
steps:
  - prompt: "hello"
`))
	if err == nil {
		t.Fatal("expected error for missing step id")
	}
}

func TestParsePipelineBytes_MissingPrompt(t *testing.T) {
	_, err := ParsePipelineBytes([]byte(`
steps:
  - id: test
`))
	if err == nil {
		t.Fatal("expected error for missing prompt")
	}
}

func TestParsePipelineBytes_AllValidationGates(t *testing.T) {
	yaml := `
steps:
  - id: full
    prompt: "full"
    validation:
      - exit_zero
      - lint
      - typecheck
      - tests
      - diff_review
`
	p, err := ParsePipelineBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParsePipelineBytes failed: %v", err)
	}

	if len(p.Steps[0].Validation) != 5 {
		t.Fatalf("expected 5 validation gates, got %d", len(p.Steps[0].Validation))
	}

	expected := []string{"exit_zero", "lint", "typecheck", "tests", "diff_review"}
	for i, v := range p.Steps[0].Validation {
		if v != expected[i] {
			t.Errorf("gate %d: expected %s, got %s", i, expected[i], v)
		}
	}
}
