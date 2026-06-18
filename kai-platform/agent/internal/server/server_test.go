package server

import (
	"context"
	"testing"

	kaipb "kaiplatform.com/gen/kaiplatform/v1"
)

type mockHandler struct {
	missions int
}

func (m *mockHandler) HandleMission(ctx context.Context, mission *kaipb.Mission, report func(*kaipb.LogEntry), result func(*kaipb.MissionResult)) {
	m.missions++
	result(&kaipb.MissionResult{
		MissionId: mission.Id,
		Success:   true,
		ExitCode:  0,
	})
}

func TestHealthCheck_Idle(t *testing.T) {
	s := New(&mockHandler{})
	resp, err := s.HealthCheck(context.Background(), &kaipb.Empty{})
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if !resp.Healthy {
		t.Error("expected healthy")
	}
}

func TestCancelMission_NotFound(t *testing.T) {
	s := New(&mockHandler{})
	_, err := s.CancelMission(context.Background(), &kaipb.MissionID{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent mission")
	}
}
