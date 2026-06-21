package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kaiplatform/config/internal/api"
	"github.com/kaiplatform/config/internal/store"
	sdkgokit "kaiplatform.com/observability-sdk"
)

func main() {
	port := flag.String("port", getEnv("CONFIG_PORT", "8081"), "HTTP port")
	dataDir := flag.String("data-dir", getEnv("CONFIG_DATA_DIR", "data/config"), "config data directory")
	orchestratorURL := flag.String("orchestrator-url", getEnv("ORCHESTRATOR_URL", ""), "orchestrator reload URL")
	flag.Parse()

	obsEndpoint := os.Getenv("OBSERVABILITY_URL")
	var obsLogger *sdkgokit.Logger
	if obsEndpoint != "" {
		obsLogger = sdkgokit.New(obsEndpoint, "config-service")
		defer obsLogger.Close()
	}

	s, err := store.New(*dataDir, *orchestratorURL)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	handler := api.New(s)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", loggingMiddleware(obsLogger, handler))

	addr := ":" + *port
	log.Printf("config service listening on %s (data: %s)", addr, *dataDir)
	if obsLogger != nil {
		obsLogger.Info("config service listening", sdkgokit.F("addr", addr), sdkgokit.F("data_dir", *dataDir))
	}
	if *orchestratorURL != "" {
		log.Printf("orchestrator push URL: %s/api/platform/config/reload", *orchestratorURL)
	}

	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
