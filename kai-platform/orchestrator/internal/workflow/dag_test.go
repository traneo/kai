package workflow

import (
	"testing"
)

func samplePipeline() *Pipeline {
	return &Pipeline{
		Version: 1,
		Project: "test",
		Output:  Output{Type: "pr", BranchPrefix: "feat/"},
		Steps: []Step{
			{ID: "init", Prompt: "init", Validation: []string{"exit_zero"}},
			{ID: "auth", Prompt: "auth", DependsOn: []string{"init"}, Validation: []string{"exit_zero"}},
			{ID: "api", Prompt: "api", DependsOn: []string{"auth"}, Validation: []string{"exit_zero"}},
		},
	}
}

func TestBuildDAG(t *testing.T) {
	dag, err := BuildDAG(samplePipeline())
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	if dag == nil {
		t.Fatal("dag is nil")
	}

	order := dag.ExecutionOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(order))
	}

	if order[0] != "init" {
		t.Errorf("expected init first, got %s", order[0])
	}

	if order[2] != "api" {
		t.Errorf("expected api last, got %s", order[2])
	}
}

func TestBuildDAG_MissingDependency(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "a", Prompt: "a"},
			{ID: "b", Prompt: "b", DependsOn: []string{"c"}},
		},
	}
	_, err := BuildDAG(p)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestBuildDAG_CircularDependency(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "a", Prompt: "a", DependsOn: []string{"b"}},
			{ID: "b", Prompt: "b", DependsOn: []string{"c"}},
			{ID: "c", Prompt: "c", DependsOn: []string{"a"}},
		},
	}
	_, err := BuildDAG(p)
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
}

func TestDAG_ReadySteps(t *testing.T) {
	dag, _ := BuildDAG(samplePipeline())

	state := map[string]StepStatus{
		"init": StepPending,
		"auth": StepPending,
		"api":  StepPending,
	}

	ready := dag.ReadySteps(state)
	if len(ready) != 1 || ready[0] != "init" {
		t.Fatalf("expected [init], got %v", ready)
	}

	state["init"] = StepPassed
	ready = dag.ReadySteps(state)
	if len(ready) != 1 || ready[0] != "auth" {
		t.Fatalf("expected [auth], got %v", ready)
	}

	state["auth"] = StepPassed
	ready = dag.ReadySteps(state)
	if len(ready) != 1 || ready[0] != "api" {
		t.Fatalf("expected [api], got %v", ready)
	}
}

func TestDAG_Dependents(t *testing.T) {
	dag, _ := BuildDAG(samplePipeline())

	deps := dag.Dependents("init")
	if len(deps) != 1 || deps[0] != "auth" {
		t.Fatalf("expected [auth], got %v", deps)
	}

	deps = dag.Dependents("auth")
	if len(deps) != 1 || deps[0] != "api" {
		t.Fatalf("expected [api], got %v", deps)
	}
}

func TestDAG_Dependencies(t *testing.T) {
	dag, _ := BuildDAG(samplePipeline())

	deps := dag.Dependencies("init")
	if len(deps) != 0 {
		t.Fatalf("expected [], got %v", deps)
	}

	deps = dag.Dependencies("auth")
	if len(deps) != 1 || deps[0] != "init" {
		t.Fatalf("expected [init], got %v", deps)
	}
}

func TestDAG_MultipleReady(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "a", Prompt: "a"},
			{ID: "b", Prompt: "b"},
			{ID: "c", Prompt: "c", DependsOn: []string{"a", "b"}},
		},
	}
	dag, _ := BuildDAG(p)

	state := map[string]StepStatus{"a": StepPending, "b": StepPending, "c": StepPending}
	ready := dag.ReadySteps(state)
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready steps, got %v", ready)
	}
}
