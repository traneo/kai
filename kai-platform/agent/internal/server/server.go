package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
)

type MissionHandler interface {
	HandleMission(ctx context.Context, mission *kaipb.Mission, report func(*kaipb.LogEntry), result func(*kaipb.MissionResult))
}

type AgentServer struct {
	kaipb.UnimplementedAgentServer
	handler    MissionHandler
	mu         sync.Mutex
	current    *missionState
	onComplete func(*kaipb.MissionResult)
}

type missionState struct {
	missionID string
	cancel    context.CancelFunc
}

func New(handler MissionHandler) *AgentServer {
	return &AgentServer{handler: handler}
}

func (s *AgentServer) OnComplete(fn func(*kaipb.MissionResult)) {
	s.onComplete = fn
}

func (s *AgentServer) AssignMission(mission *kaipb.Mission, stream grpc.ServerStreamingServer[kaipb.MissionEvent]) error {
	s.mu.Lock()
	if s.current != nil {
		s.mu.Unlock()
		return status.Error(codes.Unavailable, "agent already has an active mission")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.current = &missionState{
		missionID: mission.Id,
		cancel:    cancel,
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.current = nil
		s.mu.Unlock()
	}()

	log.Printf("received mission %q: %s", mission.Id, mission.Prompt)

	report := func(entry *kaipb.LogEntry) {
		evt := &kaipb.MissionEvent{
			Event: &kaipb.MissionEvent_Log{Log: entry},
		}
		if err := stream.Send(evt); err != nil {
			log.Printf("send log event error: %v", err)
		}
	}

	resultCh := make(chan *kaipb.MissionResult, 1)
	go func() {
		var res *kaipb.MissionResult
		s.handler.HandleMission(ctx, mission, report, func(r *kaipb.MissionResult) {
			res = r
		})
		if res == nil {
			res = &kaipb.MissionResult{
				MissionId: mission.Id,
				Success:   false,
				ExitCode:  1,
			}
		}
		resultCh <- res
	}()

	select {
	case <-ctx.Done():
		return status.Error(codes.Canceled, "mission cancelled")
	case res := <-resultCh:
		report(&kaipb.LogEntry{
			MissionId: mission.Id,
			Source:    "system",
			Level:     kaipb.LogLevel_LOG_LEVEL_INFO,
			Message:   fmt.Sprintf("mission completed: success=%v exit_code=%d", res.Success, res.ExitCode),
		})

		s.mu.Lock()
		s.current = nil
		s.mu.Unlock()

		if s.onComplete != nil {
			s.onComplete(res)
		}

		return nil
	}
}

func (s *AgentServer) CancelMission(_ context.Context, req *kaipb.MissionID) (*kaipb.Ack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && s.current.missionID == req.Id {
		s.current.cancel()
		log.Printf("cancelled mission %s", req.Id)
		return &kaipb.Ack{}, nil
	}

	return nil, status.Error(codes.NotFound, "no active mission with that id")
}

func (s *AgentServer) HealthCheck(_ context.Context, _ *kaipb.Empty) (*kaipb.HealthStatus, error) {
	s.mu.Lock()
	busy := s.current != nil
	s.mu.Unlock()

	hs := &kaipb.HealthStatus{
		Healthy:  true,
		UptimeMs: int64(0),
	}
	if busy {
		hs.MissionsCompleted = 1
	}
	return hs, nil
}
