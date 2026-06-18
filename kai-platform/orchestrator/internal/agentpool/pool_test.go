package agentpool

import (
	"context"
	"testing"
	"time"

	kaipb "kaiplatform.com/gen/kaiplatform/v1"
)

func TestNewPool(t *testing.T) {
	p := New(DefaultConfig())
	if p == nil {
		t.Fatal("pool is nil")
	}
	if p.QueueDepth() != 0 {
		t.Errorf("expected queue depth 0, got %d", p.QueueDepth())
	}
}

func TestRegisterOrUpdate_NewAgent(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "localhost:50052", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)

	agents := p.ListAgents()
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].ID != "agent-1" {
		t.Errorf("expected agent-1, got %s", agents[0].ID)
	}
	if agents[0].State != AgentIdle {
		t.Errorf("expected idle, got %s", agents[0].State)
	}
}

func TestRegisterOrUpdate_Busy(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "localhost:50052", "mission-1", kaipb.MissionStatus_MISSION_STATUS_RUNNING)

	agents := p.ListAgents()
	if agents[0].State != AgentBusy {
		t.Errorf("expected busy, got %s", agents[0].State)
	}
	if agents[0].MissionID != "mission-1" {
		t.Errorf("expected mission-1, got %s", agents[0].MissionID)
	}
}

func TestRegisterOrUpdate_Completed(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "localhost:50052", "mission-1", kaipb.MissionStatus_MISSION_STATUS_COMPLETED)

	agents := p.ListAgents()
	if agents[0].State != AgentIdle {
		t.Errorf("expected idle after complete, got %s", agents[0].State)
	}
	if agents[0].MissionID != "" {
		t.Errorf("expected empty mission id, got %s", agents[0].MissionID)
	}
	if agents[0].MissionsCompleted != 1 {
		t.Errorf("expected 1 completed, got %d", agents[0].MissionsCompleted)
	}
}

func TestGetIdleAgent(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "localhost:50052", "mission-1", kaipb.MissionStatus_MISSION_STATUS_RUNNING)
	p.RegisterOrUpdate("agent-2", "localhost:50053", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)

	id, ok := p.GetIdleAgent()
	if !ok {
		t.Fatal("expected an idle agent")
	}
	if id != "agent-2" {
		t.Errorf("expected agent-2, got %s", id)
	}
}

func TestGetIdleAgent_None(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "localhost:50052", "mission-1", kaipb.MissionStatus_MISSION_STATUS_RUNNING)

	_, ok := p.GetIdleAgent()
	if ok {
		t.Fatal("expected no idle agent")
	}
}

func TestAssignAndCompleteMission(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "localhost:50052", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)
	p.AssignMission("agent-1", "mission-1")

	rec := p.GetAgent("agent-1")
	if rec.State != AgentBusy {
		t.Errorf("expected busy, got %s", rec.State)
	}
	if rec.MissionID != "mission-1" {
		t.Errorf("expected mission-1, got %s", rec.MissionID)
	}

	p.CompleteMission("agent-1")
	rec = p.GetAgent("agent-1")
	if rec.State != AgentIdle {
		t.Errorf("expected idle, got %s", rec.State)
	}
	if rec.MissionID != "" {
		t.Errorf("expected empty mission, got %s", rec.MissionID)
	}
	if rec.MissionsCompleted != 1 {
		t.Errorf("expected 1 completed, got %d", rec.MissionsCompleted)
	}
}

func TestEnqueueDequeue(t *testing.T) {
	p := New(DefaultConfig())
	p.EnqueueMission(&MissionRequest{RunID: "run-1", StepID: "step-1", Prompt: "do something"})
	p.EnqueueMission(&MissionRequest{RunID: "run-1", StepID: "step-2", Prompt: "do more"})

	if p.QueueDepth() != 2 {
		t.Errorf("expected queue depth 2, got %d", p.QueueDepth())
	}

	req := p.DequeueMission()
	if req == nil {
		t.Fatal("expected request")
	}
	if req.RunID != "run-1" || req.StepID != "step-1" {
		t.Errorf("expected run-1/step-1, got %s/%s", req.RunID, req.StepID)
	}

	if p.QueueDepth() != 1 {
		t.Errorf("expected queue depth 1, got %d", p.QueueDepth())
	}

	_ = p.DequeueMission()
	_ = p.DequeueMission()
	if p.QueueDepth() != 0 {
		t.Errorf("expected queue depth 0 after drain, got %d", p.QueueDepth())
	}
}

func TestAgentCount(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", ":5001", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)
	p.RegisterOrUpdate("agent-2", ":5002", "m1", kaipb.MissionStatus_MISSION_STATUS_RUNNING)
	p.RegisterOrUpdate("agent-3", ":5003", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)

	if p.AgentCount(AgentIdle) != 2 {
		t.Errorf("expected 2 idle, got %d", p.AgentCount(AgentIdle))
	}
	if p.AgentCount(AgentBusy) != 1 {
		t.Errorf("expected 1 busy, got %d", p.AgentCount(AgentBusy))
	}
}

func TestPruneStale(t *testing.T) {
	p := New(Config{
		HeartbeatTimeout: 1 * time.Hour,
		HealthInterval:   100 * time.Second,
	})
	p.RegisterOrUpdate("agent-1", ":5001", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)

	p.pruneStale()

	rec := p.GetAgent("agent-1")
	if rec.State == AgentOffline {
		t.Errorf("expected NOT offline before timeout, got offline")
	}

	p.RegisterOrUpdate("agent-1", ":5001", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)
	p.heartbeatTimeout = 1 * time.Millisecond
	time.Sleep(10 * time.Millisecond)

	p.pruneStale()

	rec = p.GetAgent("agent-1")
	if rec.State != AgentOffline {
		t.Errorf("expected offline after timeout, got %s", rec.State)
	}
}

func TestRegisterOrUpdate_UpdatesExisting(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", "old:5001", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)
	p.RegisterOrUpdate("agent-1", "new:5002", "m1", kaipb.MissionStatus_MISSION_STATUS_RUNNING)

	rec := p.GetAgent("agent-1")
	if rec.Addr != "new:5002" {
		t.Errorf("expected new:5002, got %s", rec.Addr)
	}
	if rec.MissionID != "m1" {
		t.Errorf("expected m1, got %s", rec.MissionID)
	}
	if rec.State != AgentBusy {
		t.Errorf("expected busy, got %s", rec.State)
	}
}

func TestWaitForIdleAgent_Immediate(t *testing.T) {
	p := New(DefaultConfig())
	p.RegisterOrUpdate("agent-1", ":5001", "", kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	id := p.WaitForIdleAgent(ctx)
	if id != "agent-1" {
		t.Errorf("expected agent-1, got %s", id)
	}
}

func TestWaitForIdleAgent_Timeout(t *testing.T) {
	p := New(DefaultConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	id := p.WaitForIdleAgent(ctx)
	if id != "" {
		t.Errorf("expected empty on timeout, got %s", id)
	}
}

func TestWaitForIdleAgent_WaitThenSignal(t *testing.T) {
	p := New(DefaultConfig())

	done := make(chan string, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- p.WaitForIdleAgent(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	p.RegisterOrUpdate("agent-1", ":5001", "m1", kaipb.MissionStatus_MISSION_STATUS_RUNNING)
	time.Sleep(50 * time.Millisecond)
	p.CompleteMission("agent-1")

	select {
	case id := <-done:
		if id != "agent-1" {
			t.Errorf("expected agent-1, got %s", id)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for agent")
	}
}
