package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"google.golang.org/grpc"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	"kaiplatform.com/orchestrator/internal/agentpool"
	"kaiplatform.com/orchestrator/internal/api/coordinator"
	"kaiplatform.com/orchestrator/internal/cost"
	"kaiplatform.com/orchestrator/internal/validation"
	"kaiplatform.com/orchestrator/internal/workflow"
	sdkgokit "kaiplatform.com/observability-sdk"
)

type AgentInfo struct {
	ID            string
	Addr          string
	MissionID     string
	MissionStatus kaipb.MissionStatus
}

type Server struct {
	kaipb.UnimplementedOrchestratorServer
	pool        *agentpool.Pool
	coordinator *coordinator.Coordinator
	valRunner   *validation.Runner
	costTracker *cost.Tracker
	secretStore coordinator.SecretStore
	obsLogger   *sdkgokit.Logger
}

func (s *Server) SetObsLogger(l *sdkgokit.Logger) {
	s.obsLogger = l
	s.coordinator.SetObsLogger(l)
}

func (s *Server) SetSecretStore(store coordinator.SecretStore) {
	s.secretStore = store
	s.coordinator.SetSecretStore(store)
}

func NewServer(valRunner *validation.Runner) *Server {
	s := &Server{
		pool:        agentpool.New(agentpool.DefaultConfig()),
		valRunner:   valRunner,
		costTracker: cost.NewTracker(),
	}
	s.coordinator = coordinator.NewCoordinator(valRunner, s.pool)
	s.coordinator.SetServer(s)
	return s
}

func (s *Server) CostTracker() *cost.Tracker {
	return s.costTracker
}

func (s *Server) GetCoordinator() *coordinator.Coordinator {
	return s.coordinator
}

func (s *Server) Pool() *agentpool.Pool {
	return s.pool
}

func (s *Server) StartPool() {
	s.pool.Start()
}

func (s *Server) StopPool() {
	s.pool.Stop()
}

func (s *Server) GetAgents() []*AgentInfo {
	records := s.pool.ListAgents()
	result := make([]*AgentInfo, 0, len(records))
	for _, r := range records {
		status := kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED
		if r.MissionID != "" {
			status = kaipb.MissionStatus_MISSION_STATUS_RUNNING
		}
		result = append(result, &AgentInfo{
			ID:            r.ID,
			Addr:          r.Addr,
			MissionID:     r.MissionID,
			MissionStatus: status,
		})
	}
	return result
}

func (s *Server) Heartbeat(stream grpc.ClientStreamingServer[kaipb.HeartbeatRequest, kaipb.HeartbeatResponse]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&kaipb.HeartbeatResponse{})
		}
		if err != nil {
			return err
		}

		s.pool.RegisterOrUpdate(req.AgentId, req.AgentAddr, req.MissionId, req.MissionStatus)

		if req.MissionId == "" {
			s.coordinator.DrainQueue()
		}

		log.Printf("heartbeat: agent=%s addr=%s mission=%s status=%s",
			req.AgentId, req.AgentAddr, req.MissionId, req.MissionStatus)
	}
}

func (s *Server) ReportLog(stream grpc.ClientStreamingServer[kaipb.LogEntry, kaipb.LogAck]) error {
	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&kaipb.LogAck{})
		}
		if err != nil {
			return err
		}
		log.Printf("[log] %s | %s", entry.Source, entry.Message)
		if s.obsLogger != nil {
			level := string(sdkgokit.LevelInfo)
			switch entry.Level {
			case kaipb.LogLevel_LOG_LEVEL_WARN:
				level = string(sdkgokit.LevelWarn)
			case kaipb.LogLevel_LOG_LEVEL_ERROR:
				level = string(sdkgokit.LevelError)
			case kaipb.LogLevel_LOG_LEVEL_DEBUG:
				level = string(sdkgokit.LevelDebug)
			}
			s.obsLogger.Info(entry.Message, sdkgokit.F("source", entry.Source), sdkgokit.F("level", level), sdkgokit.F("mission_id", entry.MissionId), sdkgokit.F("sequence", entry.Sequence))
		}
	}
}

func (s *Server) ReportFileChange(stream grpc.ClientStreamingServer[kaipb.FileChange, kaipb.Ack]) error {
	for {
		chg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&kaipb.Ack{})
		}
		if err != nil {
			return err
		}
		log.Printf("[file] %s %s", chg.Type, chg.Path)
	}
}

func (s *Server) ReportResult(_ context.Context, r *kaipb.MissionResult) (*kaipb.ResultAck, error) {
	if s.obsLogger != nil {
		s.obsLogger.WithMissionID(r.MissionId).Info("mission result",
			sdkgokit.F("success", r.Success), sdkgokit.F("exit_code", r.ExitCode),
			sdkgokit.F("prompt_tokens", r.TokenUsagePrompt), sdkgokit.F("completion_tokens", r.TokenUsageCompletion),
			sdkgokit.F("duration_ms", r.DurationMs))
	}
	log.Printf("result: mission=%s success=%v exit_code=%d tokens=%d/%d duration=%dms",
		r.MissionId, r.Success, r.ExitCode,
		r.TokenUsagePrompt, r.TokenUsageCompletion, r.DurationMs)

	agentRec := s.pool.GetAgentByMission(r.MissionId)
	var agentID string
	if agentRec != nil {
		agentID = agentRec.ID
	}

	for _, run := range s.coordinator.ListRuns() {
		prefix := run.ID + "-"
		if strings.HasPrefix(r.MissionId, prefix) {
			stepID := strings.TrimPrefix(r.MissionId, prefix)

			s.costTracker.Record(run.ID, stepID, agentID, run.Pipeline.Project, r.TokenUsagePrompt, r.TokenUsageCompletion, r.DurationMs)

			repoDir := ""
			if gc := s.coordinator.GetGitClient(run.ID); gc != nil {
				repoDir = gc.RepoDir()
			}
			s.coordinator.HandleMissionResult(context.Background(), run.ID, stepID, agentID, r, repoDir)
			break
		}
	}

	return &kaipb.ResultAck{}, nil
}

func (s *Server) AssignMissionToAgent(ctx context.Context, agentID, missionID, prompt string, policy *workflow.Policy, ws *kaipb.Workspace) error {
	rec := s.pool.GetAgent(agentID)
	if rec == nil {
		return fmt.Errorf("agent %s not found", agentID)
	}

	client, err := s.pool.AgentClient(rec.Addr)
	if err != nil {
		return fmt.Errorf("get client for %s: %w", agentID, err)
	}

	s.pool.AssignMission(agentID, missionID)

	if ws == nil {
		ws = &kaipb.Workspace{}
	}

	if policy == nil {
		policy = &workflow.Policy{}
	}

	allowedDirs := policy.AllowedDirs
	if allowedDirs == nil {
		allowedDirs = []string{}
	}
	allowedTools := policy.AllowedTools
	if allowedTools == nil {
		allowedTools = []string{}
	}
	allowedCommands := policy.AllowedCommands
	if allowedCommands == nil {
		allowedCommands = []string{}
	}

	// Look up the pool config to determine runner type
	configBlob := ""
	runnerType := "kai-code"
	if poolCfg, ok := s.coordinator.GetAgentPool(agentID); ok && poolCfg.ConfigBlob != "" {
		configBlob = poolCfg.ConfigBlob
		var blob struct {
			Runner string `json:"runner"`
		}
		if err := json.Unmarshal([]byte(configBlob), &blob); err == nil && blob.Runner != "" {
			runnerType = blob.Runner
		}
	}

	stream, err := client.AssignMission(ctx, &kaipb.Mission{
		Id:     missionID,
		Prompt: prompt,
		Policy: &kaipb.Policy{
			MaxRetries:      int32(policy.MaxRetries),
			TimeoutSeconds:  int32(policy.TimeoutSeconds),
			AllowedDirs:     allowedDirs,
			AllowedTools:    allowedTools,
			AllowedCommands: allowedCommands,
			Runner:          runnerType,
			SaveState:       policy.SaveState,
		},
		Workspace:   ws,
		Env:         map[string]string{},
		ConfigBlob:  configBlob,
	})
	if err != nil {
		s.pool.CompleteMission(agentID)
		return fmt.Errorf("assign mission to %s: %w", agentID, err)
	}

	go func() {
		for {
			evt, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("mission %s stream error: %v", missionID, err)
				if s.obsLogger != nil {
					s.obsLogger.WithMissionID(missionID).Error("mission stream error",
						sdkgokit.F("error", err.Error()))
				}
				return
			}
			s.coordinator.StoreMissionEvent(missionID, evt)
		}
	}()

	return nil
}

func (s *Server) GetAgentClient(addr string) (kaipb.AgentClient, error) {
	return s.pool.AgentClient(addr)
}
