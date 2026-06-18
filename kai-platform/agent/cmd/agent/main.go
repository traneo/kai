package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	"kaiplatform.com/agent/internal/client"
	"kaiplatform.com/agent/internal/runner"
	"kaiplatform.com/agent/internal/sandbox"
	"kaiplatform.com/agent/internal/server"
)

func main() {
	orchAddr := flag.String("orchestrator", getEnv("ORCHESTRATOR_ADDR", "localhost:50051"), "orchestrator address")
	agentID := flag.String("agent-id", getEnv("AGENT_ID", "agent-1"), "unique agent identifier")
	agentAddr := flag.String("agent-addr", getEnv("AGENT_ADDR", "localhost:50052"), "agent gRPC server address")
	listenPort := flag.String("listen", getEnv("AGENT_LISTEN", ":50052"), "agent gRPC server listen address")
	timeout := flag.Duration("timeout", getEnvDuration("MISSION_TIMEOUT", 30*time.Minute), "per-mission timeout")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cl := client.New(client.Config{
		OrchestratorAddr: *orchAddr,
		AgentID:          *agentID,
		AgentAddr:        *agentAddr,
	})

	if err := cl.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer cl.Close()

	go func() {
		if err := cl.HeartbeatLoop(ctx); err != nil {
			log.Printf("heartbeat loop ended: %v", err)
		}
	}()

	runner.DiscoverAndLoadPlugins(runner.DefaultPluginDir())

	handler := &kaiMissionHandler{
		cl:      cl,
		timeout: *timeout,
	}

	agentSrv := server.New(handler)
	agentSrv.OnComplete(func(res *kaipb.MissionResult) {
		if _, err := cl.ReportResult(context.Background(), res); err != nil {
			log.Printf("report result to orchestrator: %v", err)
		}
	})

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024),
		grpc.MaxSendMsgSize(50*1024*1024),
	)
	kaipb.RegisterAgentServer(grpcServer, agentSrv)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", *listenPort)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	go func() {
		log.Printf("agent gRPC server listening on %s", *listenPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	grpcServer.GracefulStop()
}

type kaiMissionHandler struct {
	cl      *client.Client
	timeout time.Duration
}

func (h *kaiMissionHandler) HandleMission(ctx context.Context, mission *kaipb.Mission, report func(*kaipb.LogEntry), result func(*kaipb.MissionResult)) {
	h.cl.SetMissionID(mission.Id)
	defer h.cl.ClearMissionID()

	sb := sandbox.New(sandbox.Config{
		AllowedDirs:     mission.Policy.AllowedDirs,
		AllowedTools:    mission.Policy.AllowedTools,
		AllowedCommands: mission.Policy.AllowedCommands,
	})
	defer sb.Cleanup()

	var wsErr error
	if mission.Workspace != nil && mission.Workspace.RepoUrl != "" {
		wsErr = sb.SetupWithRepo(ctx, mission.Workspace.RepoUrl, mission.Workspace.Branch)
		report(&kaipb.LogEntry{
			MissionId: mission.Id,
			Source:    "system",
			Message:   fmt.Sprintf("cloning repo %s (branch: %s)", mission.Workspace.RepoUrl, mission.Workspace.Branch),
		})
	} else {
		wsErr = sb.Setup()
	}

	if wsErr != nil {
		log.Printf("sandbox setup failed: %v", wsErr)
		result(&kaipb.MissionResult{
			MissionId: mission.Id,
			Success:   false,
			ExitCode:  1,
		})
		return
	}

	report(&kaipb.LogEntry{
		MissionId: mission.Id,
		Source:    "system",
		Message:   fmt.Sprintf("workspace ready at %s", sb.RepoDir),
	})

	runnerType := "kai"
	if mission.Policy != nil && mission.Policy.Runner != "" {
		runnerType = mission.Policy.Runner
	}

	agentDir, _ := filepath.Split(os.Args[0])

	rnr, err := runner.New(runnerType, runner.Config{
		WorkDir:  sb.WorkDir,
		RepoDir:  sb.RepoDir,
		AgentDir: agentDir,
		Timeout:  h.timeout,
	})
	if err != nil {
		report(&kaipb.LogEntry{
			MissionId: mission.Id,
			Source:    "system",
			Message:   fmt.Sprintf("unknown runner %q: %v", runnerType, err),
		})
		log.Printf("unknown runner %q: %v", runnerType, err)
		result(&kaipb.MissionResult{
			MissionId: mission.Id,
			Success:   false,
			ExitCode:  1,
		})
		return
	}

	var policy *runner.Policy
	if mission.Policy != nil {
		policy = &runner.Policy{
			AllowedTools:    mission.Policy.AllowedTools,
			AllowedCommands: mission.Policy.AllowedCommands,
			AllowedDirs:     mission.Policy.AllowedDirs,
		}
	}

	if err := rnr.WriteConfig(runner.WriteConfigInput{
		ConfigBlob: mission.ConfigBlob,
		Policy:     policy,
	}); err != nil {
		report(&kaipb.LogEntry{
			MissionId: mission.Id,
			Source:    "system",
			Message:   fmt.Sprintf("write config failed: %v", err),
		})
		log.Printf("write config failed: %v", err)
		result(&kaipb.MissionResult{
			MissionId: mission.Id,
			Success:   false,
			ExitCode:  1,
		})
		return
	}

	report(&kaipb.LogEntry{
		MissionId: mission.Id,
		Source:    "system",
		Message:   fmt.Sprintf("runner = %q | config_blob = %s", runnerType, mission.ConfigBlob),
	})

	report(&kaipb.LogEntry{
		MissionId: mission.Id,
		Source:    "system",
		Message:   fmt.Sprintf("starting %s run with prompt: %s", runnerType, mission.Prompt),
	})

	onLine := func(line, source string) {
		report(&kaipb.LogEntry{
			MissionId: mission.Id,
			Source:    source,
			Message:   line,
		})
	}

	runResult, err := rnr.Run(ctx, mission.Prompt, onLine)
	if err != nil {
		log.Printf("run failed: %v", err)
		result(&kaipb.MissionResult{
			MissionId: mission.Id,
			Success:   false,
			ExitCode:  1,
		})
		return
	}

	res := &kaipb.MissionResult{
		MissionId:  mission.Id,
		Success:    runResult.Success,
		ExitCode:   int32(runResult.ExitCode),
		DurationMs: runResult.Duration.Milliseconds(),
	}

	if runResult.Success && mission.Workspace != nil && mission.Workspace.RepoUrl != "" {
		branch := mission.Workspace.Branch

		gitOut := func(args ...string) (string, error) {
			all := append([]string{"-c", "credential.helper="}, args...)
			cmd := exec.CommandContext(ctx, "git", all...)
			cmd.Dir = sb.RepoDir
			out, err := cmd.CombinedOutput()
			return strings.TrimSpace(string(out)), err
		}

		logGit := func(args ...string) {
			out, err := gitOut(args...)
			report(&kaipb.LogEntry{
				MissionId: mission.Id,
				Source:    "system",
				Message:   fmt.Sprintf("git %s: %s", strings.Join(args, " "), out),
			})
			if err != nil {
				log.Printf("git %s: %v", args[0], err)
			}
		}

		logGit("add", "-A")
		logGit("commit", "-m", fmt.Sprintf("kai: %s", runner.TruncatePrompt(mission.Prompt)))

		// Push with pull --rebase fallback for concurrent agents
		pushOut, pushErr := gitOut("push", "origin", branch)
		if pushErr != nil && strings.Contains(pushOut, "rejected") {
			report(&kaipb.LogEntry{
				MissionId: mission.Id,
				Source:    "system",
				Message:   "push rejected, pulling remote changes...",
			})
			gitOut("fetch", "origin")
			rebaseOut, rebaseErr := gitOut("pull", "--rebase", "origin", branch)
			if rebaseErr != nil {
				gitOut("rebase", "--abort")
				report(&kaipb.LogEntry{
					MissionId: mission.Id,
					Source:    "system",
					Message:   fmt.Sprintf("rebase failed, aborting. push skipped: %s", rebaseOut),
				})
			} else {
				pushOut, pushErr = gitOut("push", "origin", branch)
				report(&kaipb.LogEntry{
					MissionId: mission.Id,
					Source:    "system",
					Message:   fmt.Sprintf("git push after rebase: %s", pushOut),
				})
				if pushErr != nil {
					log.Printf("git push after rebase: %v", pushErr)
				}
			}
		} else if pushErr != nil {
			log.Printf("git push: %v", pushErr)
		} else {
			report(&kaipb.LogEntry{
				MissionId: mission.Id,
				Source:    "system",
				Message:   fmt.Sprintf("git push: %s", pushOut),
			})
		}
	}

	if mission.Policy.GetSaveState() {
		report(&kaipb.LogEntry{
			MissionId: mission.Id,
			Source:    "system",
			Message:   "saving workspace state as archive...",
		})
		archive, zipErr := sandbox.ZipDir(sb.RepoDir)
		if zipErr != nil {
			log.Printf("zip workspace state: %v", zipErr)
		} else {
			res.StateArchive = archive
			report(&kaipb.LogEntry{
				MissionId: mission.Id,
				Source:    "system",
				Message:   fmt.Sprintf("workspace state archive created (%d bytes)", len(archive)),
			})
		}
	}

	result(res)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}
