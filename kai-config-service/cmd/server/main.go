package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/kaiplatform/config/internal/api"
	"github.com/kaiplatform/config/internal/store"
)

func main() {
	port := flag.String("port", getEnv("CONFIG_PORT", "8081"), "HTTP port")
	dataDir := flag.String("data-dir", getEnv("CONFIG_DATA_DIR", "data/config"), "config data directory")
	orchestratorURL := flag.String("orchestrator-url", getEnv("ORCHESTRATOR_URL", ""), "orchestrator reload URL")
	flag.Parse()

	s, err := store.New(*dataDir, *orchestratorURL)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	handler := api.New(s)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", handler)

	addr := ":" + *port
	log.Printf("config service listening on %s (data: %s)", addr, *dataDir)
	if *orchestratorURL != "" {
		log.Printf("orchestrator push URL: %s/api/platform/config/reload", *orchestratorURL)
	}

	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatalf("serve: %v", err)
	}
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
