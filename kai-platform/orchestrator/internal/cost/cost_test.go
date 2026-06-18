package cost

import (
	"testing"
)

func TestNewTracker(t *testing.T) {
	tr := NewTracker()
	if tr == nil {
		t.Fatal("tracker is nil")
	}
}

func TestRecord(t *testing.T) {
	tr := NewTracker()
	tr.Record("run-1", "step-1", "agent-1", "proj", 100, 50, 2000)
	rc := tr.GetRunCost("run-1")
	if rc == nil {
		t.Fatal("run cost not found")
	}
	if len(rc.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(rc.Steps))
	}
	if rc.TotalTokens != 150 {
		t.Errorf("expected 150 tokens, got %d", rc.TotalTokens)
	}
	if rc.DurationMs != 2000 {
		t.Errorf("expected 2000ms, got %d", rc.DurationMs)
	}
	if rc.Project != "proj" {
		t.Errorf("expected proj, got %s", rc.Project)
	}
}

func TestMultipleRecords(t *testing.T) {
	tr := NewTracker()
	tr.Record("run-1", "s1", "a1", "proj", 100, 50, 1000)
	tr.Record("run-1", "s2", "a1", "proj", 200, 100, 2000)
	tr.Record("run-2", "s1", "a2", "proj2", 50, 25, 500)

	rc := tr.GetRunCost("run-1")
	if rc.TotalTokens != 450 {
		t.Errorf("expected 450 tokens, got %d", rc.TotalTokens)
	}
	if rc.DurationMs != 3000 {
		t.Errorf("expected 3000ms, got %d", rc.DurationMs)
	}

	rc2 := tr.GetRunCost("run-2")
	if rc2.TotalTokens != 75 {
		t.Errorf("expected 75 tokens, got %d", rc2.TotalTokens)
	}
}

func TestStats(t *testing.T) {
	tr := NewTracker()
	tr.Record("run-1", "s1", "a1", "proj", 100, 50, 1000)
	tr.Record("run-2", "s1", "a1", "proj2", 200, 100, 2000)

	s := tr.Stats()
	if s.TotalRuns != 2 {
		t.Errorf("expected 2 runs, got %d", s.TotalRuns)
	}
	if s.TotalSteps != 2 {
		t.Errorf("expected 2 steps, got %d", s.TotalSteps)
	}
	if s.TotalTokens != 450 {
		t.Errorf("expected 450 tokens, got %d", s.TotalTokens)
	}
	if s.TotalPromptToken != 300 {
		t.Errorf("expected 300 prompt tokens, got %d", s.TotalPromptToken)
	}
	if s.TotalCompToken != 150 {
		t.Errorf("expected 150 comp tokens, got %d", s.TotalCompToken)
	}
	if s.TotalDurationMs != 3000 {
		t.Errorf("expected 3000ms, got %d", s.TotalDurationMs)
	}
}

func TestStats_AgentUsage(t *testing.T) {
	tr := NewTracker()
	tr.Record("run-1", "s1", "agent-a", "proj", 100, 50, 1000)
	tr.Record("run-1", "s2", "agent-b", "proj", 200, 100, 2000)
	tr.Record("run-2", "s1", "agent-a", "proj", 50, 25, 500)

	s := tr.Stats()
	if len(s.AgentUsage) != 2 {
		t.Errorf("expected 2 agents, got %d", len(s.AgentUsage))
	}
	aa := s.AgentUsage["agent-a"]
	if aa == nil {
		t.Fatal("agent-a not found")
	}
	if aa.StepsDone != 2 {
		t.Errorf("expected 2 steps for agent-a, got %d", aa.StepsDone)
	}
	if aa.TotalTokens != 225 {
		t.Errorf("expected 225 tokens for agent-a, got %d", aa.TotalTokens)
	}
}

func TestAllRuns(t *testing.T) {
	tr := NewTracker()
	tr.Record("run-1", "s1", "a1", "proj", 10, 5, 100)
	tr.Record("run-2", "s1", "a1", "proj", 20, 10, 200)

	runs := tr.AllRuns()
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestGetRunCost_NotFound(t *testing.T) {
	tr := NewTracker()
	rc := tr.GetRunCost("nonexistent")
	if rc != nil {
		t.Error("expected nil for nonexistent run")
	}
}


