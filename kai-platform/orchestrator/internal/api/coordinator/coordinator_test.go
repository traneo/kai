package coordinator

import (
	"testing"

	"kaiplatform.com/orchestrator/internal/agentpool"
	"kaiplatform.com/orchestrator/internal/validation"
	"kaiplatform.com/orchestrator/internal/workflow"
)

func newTestCoordinator(vr *validation.Runner) *Coordinator {
	return NewCoordinator(vr, agentpool.New(agentpool.DefaultConfig()))
}

func TestCoordinator_CreateRun(t *testing.T) {
	valRunner := validation.NewRunner()
	valRunner.Register(validation.NewExitCodeGate())

	c := newTestCoordinator(valRunner)
	p := &workflow.Pipeline{
		Steps: []workflow.Step{
			{ID: "init", Prompt: "init", Validation: []string{"exit_zero"}},
		},
	}

	run, err := c.CreateRun("run-1", p, "")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if run.ID != "run-1" {
		t.Errorf("expected run ID 'run-1', got %q", run.ID)
	}
	if run.Status != workflow.PipelineRunning {
		t.Errorf("expected running status, got %v", run.Status)
	}
}

func TestCoordinator_ListRuns(t *testing.T) {
	valRunner := validation.NewRunner()
	c := newTestCoordinator(valRunner)

	runs := c.ListRuns()
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}

	p := &workflow.Pipeline{
		Steps: []workflow.Step{{ID: "a", Prompt: "a"}},
	}
	_, err := c.CreateRun("r1", p, "")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	runs = c.ListRuns()
	if len(runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(runs))
	}
}

func TestCoordinator_GetRun(t *testing.T) {
	valRunner := validation.NewRunner()
	c := newTestCoordinator(valRunner)

	run := c.GetRun("nonexistent")
	if run != nil {
		t.Error("expected nil for nonexistent run")
	}

	p := &workflow.Pipeline{
		Steps: []workflow.Step{{ID: "a", Prompt: "a"}},
	}
	c.CreateRun("r1", p, "")

	run = c.GetRun("r1")
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if run.ID != "r1" {
		t.Errorf("expected run ID 'r1', got %q", run.ID)
	}
}

type mockServer struct{}

func TestCoordinator_AgentPools(t *testing.T) {
	c := newTestCoordinator(validation.NewRunner())

	pools := []AgentPoolConfig{
		{
			Name:       "pool-1",
			ConfigBlob: `{"runner":"opencode","data":{}}`,
		},
	}
	c.SetAgentPools(pools)

	p, ok := c.GetAgentPool("pool-1")
	if !ok {
		t.Fatal("expected pool-1 to be found")
	}
	if p.ConfigBlob != `{"runner":"opencode","data":{}}` {
		t.Errorf("unexpected config blob: %s", p.ConfigBlob)
	}
}

func TestCoordinator_CancelRun(t *testing.T) {
	valRunner := validation.NewRunner()
	c := newTestCoordinator(valRunner)

	p := &workflow.Pipeline{
		Steps: []workflow.Step{{ID: "a", Prompt: "a"}},
	}
	c.CreateRun("r1", p, "")

	if err := c.CancelRun("r1"); err != nil {
		t.Fatalf("CancelRun: %v", err)
	}

	run := c.GetRun("r1")
	if run.Status != workflow.PipelineCancelled {
		t.Errorf("expected cancelled, got %v", run.Status)
	}
}

func TestCoordinator_CancelRun_NotFound(t *testing.T) {
	c := newTestCoordinator(validation.NewRunner())
	if err := c.CancelRun("nonexistent"); err == nil {
		t.Error("expected error for nonexistent run")
	}
}

func TestNewCoordinator(t *testing.T) {
	valRunner := validation.NewRunner()
	pool := agentpool.New(agentpool.DefaultConfig())
	c := NewCoordinator(valRunner, pool)

	if c == nil {
		t.Fatal("expected non-nil coordinator")
	}
	if c.valRunner != valRunner {
		t.Error("valRunner not set")
	}
	if c.pool != pool {
		t.Error("pool not set")
	}
	if c.runs == nil {
		t.Error("runs map not initialized")
	}
	if c.gitClients == nil {
		t.Error("gitClients map not initialized")
	}
}
