package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	"kaiplatform.com/orchestrator/internal/api"
	"kaiplatform.com/orchestrator/internal/api/coordinator"
	"kaiplatform.com/orchestrator/internal/api/handlers"
	"kaiplatform.com/orchestrator/internal/archive"
	"kaiplatform.com/orchestrator/internal/audit"
	"kaiplatform.com/orchestrator/internal/auth"
	"kaiplatform.com/orchestrator/internal/gitprovider"
	"kaiplatform.com/orchestrator/internal/runstore"
	"kaiplatform.com/orchestrator/internal/secrets"
	"kaiplatform.com/orchestrator/internal/validation"
	"kaiplatform.com/orchestrator/internal/validation/gates"
	sdkgokit "kaiplatform.com/observability-sdk"
)

func startServer(httpPort, configServiceURL string) {
	grpcPort := getEnv("PORT", "50051")
	ctx := context.Background()

	authCfg := auth.LoadConfig()
	authenticator := auth.New(authCfg)

	auditStore := audit.NewStoreFromEnv(ctx)

	secMgr := secrets.NewManagerFromEnv(ctx)

	obsEndpoint := os.Getenv("OBSERVABILITY_URL")
	var obsLogger *sdkgokit.Logger
	if obsEndpoint != "" {
		obsLogger = sdkgokit.New(obsEndpoint, "orchestrator")
		obsLogger.Info("observability forwarding", sdkgokit.F("endpoint", obsEndpoint))
	}

	gitProvReg := gitprovider.NewRegistry(gitprovider.DefaultPluginDir())
	gitProvReg.RegisterBuiltins()
	if err := gitProvReg.Discover(); err != nil {
		logf(obsLogger, "git provider plugin discovery: %v", err)
	}

	valRunner := validation.NewRunner()
	valRunner.RegisterAll(
		validation.NewExitCodeGate(),
		validation.NewDiffReviewGate(),
		validation.NewApprovalGate(false),
		validation.NewLintGate(),
		validation.NewTypecheckGate(),
		validation.NewTestsGate(),
		gates.NewSecurityScanGate(),
		gates.NewLicenseGate(),
		gates.NewBreakingChangesGate(),
		gates.NewCodeQualityGate(),
	)

	pluginMgr := validation.NewPluginManager(validation.DefaultPluginDir())
	if err := pluginMgr.Discover(); err != nil {
		logf(obsLogger, "plugin discovery: %v", err)
	}
	for _, pg := range pluginMgr.LoadGates() {
		logf(obsLogger, "registering plugin gate: %s", pg.Name())
		valRunner.Register(pg)
	}

	convStore := api.NewMemoryConversationStore()
	defer convStore.Close()

	archiveStore := archive.NewStoreFromEnv(ctx)
	defer archiveStore.Close()

	runStore := runstore.NewStoreFromEnv(ctx)
	defer runStore.Close()
	defer archiveStore.Close()

	secretStore := api.NewMemorySecretStore()

	srv := api.NewServer(valRunner)
	srv.SetObsLogger(obsLogger)
	srv.GetCoordinator().SetServer(srv)
	srv.GetCoordinator().SetAuditStore(auditStore)
	srv.GetCoordinator().SetArchiveStore(archiveStore)
	srv.GetCoordinator().SetConversationStore(convStore)
	srv.GetCoordinator().SetSecretsManager(secMgr)
	srv.GetCoordinator().SetGitProviderRegistry(gitProvReg)
	srv.GetCoordinator().SetRunStore(runStore)
	srv.GetCoordinator().RestoreRuns(ctx)
	srv.SetSecretStore(secretStore)
	srv.StartPool()
	defer srv.StopPool()
	httpSrv := api.NewHTTPServer(srv, srv.GetCoordinator())
	httpSrv.SetSecretStore(secretStore)
	srv.GetCoordinator().SetEventPublisher(httpSrv.PublishEvent)

	grpcOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(50 * 1024 * 1024),
		grpc.MaxSendMsgSize(50 * 1024 * 1024),
	}

	if creds, err := authenticator.ServerCredentials(); err != nil {
		log.Fatalf("server credentials: %v", err)
	} else if creds != nil {
		grpcOpts = append(grpcOpts, creds)
	}

	if !authCfg.Insecure {
		grpcOpts = append(grpcOpts,
			grpc.UnaryInterceptor(authenticator.UnaryServerInterceptor()),
			grpc.StreamInterceptor(authenticator.StreamServerInterceptor()),
		)
	}

	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpcOpts...)
	kaipb.RegisterOrchestratorServer(grpcServer, srv)
	if authCfg.Insecure {
		reflection.Register(grpcServer)
	}

	go func() {
		logf(obsLogger, "gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// Block until initial config is loaded from the config service.
	// Agents can connect via gRPC but missions can't run without config.
	if configServiceURL != "" {
		logf(obsLogger, "fetching initial config from %s ...", configServiceURL)
		fetchInitialConfig(configServiceURL, srv, srv.GetCoordinator(), &authCfg.PreSharedToken, obsLogger)
		logf(obsLogger, "initial config loaded")
	}

	apiHandler := loggingMiddleware(obsLogger, httpSrv.Handler())

	startedAt := time.Now()

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","uptime":"%s"}`, time.Since(startedAt).Round(time.Second))
	})
	httpMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		agents := srv.Pool().ListAgents()
		fmt.Fprintf(w, `{"status":"ok","agents_connected":%t,"config_loaded":%t}`,
			len(agents) > 0, srv.GetCoordinator().ConfigLoaded())
	})
	httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		stats := srv.CostTracker().Stats()
		agents := len(srv.GetAgents())
		queueDepth := srv.Pool().QueueDepth()
		fmt.Fprintf(w, "# HELP kai_agents_total Total agents connected\n")
		fmt.Fprintf(w, "# TYPE kai_agents_total gauge\n")
		fmt.Fprintf(w, "kai_agents_total %d\n", agents)
		fmt.Fprintf(w, "# HELP kai_agents_idle Idle agents\n")
		fmt.Fprintf(w, "# TYPE kai_agents_idle gauge\n")
		fmt.Fprintf(w, "kai_agents_idle %d\n", srv.Pool().AgentCount("idle"))
		fmt.Fprintf(w, "# HELP kai_agents_busy Busy agents\n")
		fmt.Fprintf(w, "# TYPE kai_agents_busy gauge\n")
		fmt.Fprintf(w, "kai_agents_busy %d\n", srv.Pool().AgentCount("busy"))
		fmt.Fprintf(w, "# HELP kai_queue_depth Mission queue depth\n")
		fmt.Fprintf(w, "# TYPE kai_queue_depth gauge\n")
		fmt.Fprintf(w, "kai_queue_depth %d\n", queueDepth)
		fmt.Fprintf(w, "# HELP kai_pipelines_total Total pipeline runs\n")
		fmt.Fprintf(w, "# TYPE kai_pipelines_total counter\n")
		fmt.Fprintf(w, "kai_pipelines_total %d\n", stats.TotalRuns)
		fmt.Fprintf(w, "# HELP kai_steps_total Total steps executed\n")
		fmt.Fprintf(w, "# TYPE kai_steps_total counter\n")
		fmt.Fprintf(w, "kai_steps_total %d\n", stats.TotalSteps)
		fmt.Fprintf(w, "# HELP kai_tokens_total Total tokens used\n")
		fmt.Fprintf(w, "# TYPE kai_tokens_total counter\n")
		fmt.Fprintf(w, "kai_tokens_total %d\n", stats.TotalTokens)
		fmt.Fprintf(w, "# HELP kai_tokens_prompt Prompt tokens used\n")
		fmt.Fprintf(w, "# TYPE kai_tokens_prompt counter\n")
		fmt.Fprintf(w, "kai_tokens_prompt %d\n", stats.TotalPromptToken)
		fmt.Fprintf(w, "# HELP kai_tokens_completion Completion tokens used\n")
		fmt.Fprintf(w, "# TYPE kai_tokens_completion counter\n")
		fmt.Fprintf(w, "kai_tokens_completion %d\n", stats.TotalCompToken)
		fmt.Fprintf(w, "# HELP kai_uptime_seconds Server uptime\n")
		fmt.Fprintf(w, "# TYPE kai_uptime_seconds gauge\n")
		fmt.Fprintf(w, "kai_uptime_seconds %.0f\n", time.Since(startedAt).Seconds())
	})
	httpMux.HandleFunc("/api/platform/config/reload", httpSrv.HandleConfigReload)

	if u, err := url.Parse(configServiceURL); err == nil && u.Host != "" {
		configProxy := httputil.NewSingleHostReverseProxy(u)
		httpMux.Handle("/api/v1/config", configProxy)
		httpMux.Handle("/api/v1/config/", configProxy)
		log.Printf("proxying /api/v1/config/* -> %s", configServiceURL)
	} else {
		log.Print("config service proxy not configured")
	}

	httpMux.Handle("/api/", apiHandler)

	authToken := &authCfg.PreSharedToken
	tokenMiddleware := auth.HTTPTokenMiddleware(authToken)
	httpSrv.SetAuthToken(authToken)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: corsMiddleware(tokenMiddleware(httpMux)),
	}

	go func() {
		logf(obsLogger, "HTTP API listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logf(obsLogger, "shutting down")
	if obsLogger != nil {
		obsLogger.Close()
	}
	grpcServer.GracefulStop()
	httpServer.Close()
}

func fetchInitialConfig(configServiceURL string, srv handlers.ServerIface, coord *coordinator.Coordinator, authToken *string, obsLogger *sdkgokit.Logger) {
	configURL := configServiceURL + "/api/v1/config"
	client := &http.Client{Timeout: 10 * time.Second}

	deps := &handlers.Deps{
		Coordinator: coord,
		Server:      srv,
		AuthToken:   authToken,
	}

	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		resp, err := client.Get(configURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				logf(obsLogger, "read config response: %v, retrying...", readErr)
			} else if applyErr := handlers.ApplyConfigFromBytes(deps, body); applyErr != nil {
				logf(obsLogger, "apply initial config: %v, retrying...", applyErr)
			} else {
				return
			}
		} else {
			if resp != nil {
				resp.Body.Close()
			}
			if err != nil {
				logf(obsLogger, "config-service unreachable (%v), retrying in %v...", err, backoff)
			} else if resp.StatusCode == http.StatusNotFound {
				logf(obsLogger, "config-service has no active config yet, retrying in %v...", backoff)
			} else {
				logf(obsLogger, "config-service returned %d, retrying in %v...", resp.StatusCode, backoff)
			}
		}

		time.Sleep(backoff + time.Duration(rand.Intn(1000))*time.Millisecond)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

type logWriter struct {
	http.ResponseWriter
	status int
}

func (w *logWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *logWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func loggingMiddleware(logger *sdkgokit.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &logWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(lw, r)
		elapsed := time.Since(start)
		if logger != nil {
			logger.Info("http request",
				sdkgokit.F("method", r.Method),
				sdkgokit.F("path", r.URL.Path),
				sdkgokit.F("status", lw.status),
				sdkgokit.F("duration_ms", elapsed.Milliseconds()),
			)
		} else {
			log.Printf("%s %s %d (%v)", r.Method, r.URL.Path, lw.status, elapsed)
		}
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logf(logger *sdkgokit.Logger, format string, args ...any) {
	if logger != nil {
		logger.Info(fmt.Sprintf(format, args...))
	} else {
		log.Printf(format, args...)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
