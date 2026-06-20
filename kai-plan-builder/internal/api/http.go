package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	sdkgokit "kaiplatform.com/observability-sdk"
	"github.com/kaiplatform/plan-builder/internal/chat"
	"github.com/kaiplatform/plan-builder/internal/llm"
)

type Deps struct {
	LLMClient *llm.Client
	Sessions  *chat.Store
	Logger    *sdkgokit.Logger
}

func Handler(deps *Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/plan-builder/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleChat(deps, w, r)
	})
	mux.HandleFunc("/api/v1/plan-builder/generate-pipeline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleGeneratePipeline(deps, w, r)
	})
	mux.HandleFunc("/api/v1/plan-builder/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	return corsMiddleware(mux)
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

func writeError(w http.ResponseWriter, code int, format string, args ...any) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf(format, args...),
	})
}
