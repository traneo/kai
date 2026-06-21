package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sdkgokit "kaiplatform.com/observability-sdk"
	"github.com/kaiplatform/plan-builder/internal/api"
	"github.com/kaiplatform/plan-builder/internal/chat"
	"github.com/kaiplatform/plan-builder/internal/config"
	"github.com/kaiplatform/plan-builder/internal/llm"
)

func main() {
	port := getEnv("PORT", "8083")
	configServiceURL := os.Getenv("CONFIG_SERVICE_URL")
	obsURL := os.Getenv("OBSERVABILITY_URL")

	if configServiceURL == "" {
		log.Fatal("CONFIG_SERVICE_URL environment variable is required")
	}
	if obsURL == "" {
		log.Fatal("OBSERVABILITY_URL environment variable is required")
	}

	cfg, err := config.Fetch(configServiceURL)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	llmClient := llm.New(cfg.PlanBuilder.LLM.Endpoint, cfg.PlanBuilder.LLM.Model, cfg.PlanBuilder.LLM.APIKey)

	obsLogger := sdkgokit.New(obsURL, "plan-builder")
	defer obsLogger.Close()

	sessionStore := chat.NewStore()

	deps := &api.Deps{
		LLMClient: llmClient,
		Sessions:  sessionStore,
		Logger:    obsLogger,
	}

	handler := loggingMiddleware(obsLogger, api.Handler(deps))

	addr := fmt.Sprintf(":%s", port)
	log.Printf("plan-builder listening on %s", addr)
	obsLogger.Info("plan-builder started", sdkgokit.F("port", port), sdkgokit.F("config_service", configServiceURL))

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
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

func loggingMiddleware(logger *sdkgokit.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &logWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(lw, r)
		elapsed := time.Since(start)
		logger.Info("http request",
			sdkgokit.F("method", r.Method),
			sdkgokit.F("path", r.URL.Path),
			sdkgokit.F("status", lw.status),
			sdkgokit.F("duration_ms", elapsed.Milliseconds()),
		)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
