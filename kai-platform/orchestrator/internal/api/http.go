package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"kaiplatform.com/orchestrator/internal/api/coordinator"
	"kaiplatform.com/orchestrator/internal/api/handlers"
)

type HTTPServer struct {
	server      *Server
	coordinator *coordinator.Coordinator
	secretStore coordinator.SecretStore
	authToken   *string
	mu          sync.RWMutex
	events      []any
	subscribers map[chan any]struct{}
}

func (h *HTTPServer) SetAuthToken(token *string) {
	h.authToken = token
}

func NewHTTPServer(srv *Server, coord *coordinator.Coordinator) *HTTPServer {
	return &HTTPServer{
		server:      srv,
		coordinator: coord,
		subscribers: make(map[chan any]struct{}),
	}
}

func (h *HTTPServer) SetSecretStore(store coordinator.SecretStore) {
	h.secretStore = store
}

func (h *HTTPServer) deps() *handlers.Deps {
	return &handlers.Deps{
		Coordinator:  h.coordinator,
		Server:       h.server,
		SecretStore:  h.secretStore,
		AuthToken:    h.authToken,
		PublishEvent: h.PublishEvent,
		Mu:           &h.mu,
		Subscribers:  &h.subscribers,
	}
}

func (h *HTTPServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		d := h.deps()

		switch {
		case r.Method == "GET" && r.URL.Path == "/api/status":
			handlers.HandleStatus(d, w, r)
		case r.Method == "GET" && r.URL.Path == "/api/agents":
			handlers.HandleAgents(d, w, r)
		case r.Method == "GET" && r.URL.Path == "/api/pipelines":
			handlers.HandlePipelines(d, w, r)
		case r.Method == "GET" && matchPath(r.URL.Path, "/api/pipelines/{id}"):
			id, _ := extractPathParams(r.URL.Path, "/api/pipelines/{id}")
			handlers.HandlePipelineDetail(d, w, r, id)
		case r.Method == "GET" && matchPath(r.URL.Path, "/api/pipelines/{id}/steps/{step}/conversation"):
			id, step := extractPathParams(r.URL.Path, "/api/pipelines/{id}/steps/{step}/conversation")
			handlers.HandleConversation(d, w, r, id, step)
		case r.Method == "GET" && r.URL.Path == "/api/stats":
			handlers.HandleStats(d, w, r)
		case r.Method == "GET" && r.URL.Path == "/api/events":
			handlers.HandleEvents(d, w, r)
		case r.Method == "POST" && r.URL.Path == "/api/pipelines":
			handlers.HandleCreatePipeline(d, w, r)
		case r.Method == "GET" && r.URL.Path == "/api/policies":
			handlers.HandlePolicies(w, r)
		case r.Method == "GET" && r.URL.Path == "/api/queue":
			handlers.HandleQueue(d, w, r)
		case r.Method == "GET" && r.URL.Path == "/api/audit":
			handlers.HandleAudit(d, w, r)
		case r.Method == "POST" && matchPath(r.URL.Path, "/api/pipelines/{id}/cancel"):
			id, _ := extractPathParams(r.URL.Path, "/api/pipelines/{id}/cancel")
			handlers.HandleCancel(d, w, r, id)
		case r.Method == "GET" && matchPath(r.URL.Path, "/api/pipelines/{id}/yaml"):
			id, _ := extractPathParams(r.URL.Path, "/api/pipelines/{id}/yaml")
			handlers.HandlePipelineYAML(d, w, r, id)
		case r.Method == "POST" && matchPath(r.URL.Path, "/api/pipelines/{id}/steps/{step}/approve"):
			id, step := extractPathParams(r.URL.Path, "/api/pipelines/{id}/steps/{step}/approve")
			handlers.HandleApprove(d, w, r, id, step)
		case r.Method == "GET" && r.URL.Path == "/api/secrets":
			handlers.HandleSecretsList(d, w, r)
		case r.Method == "POST" && r.URL.Path == "/api/secrets":
			handlers.HandleSecretsSet(d, w, r)
		case r.Method == "DELETE" && matchPath(r.URL.Path, "/api/secrets/{name}"):
			name, _ := extractPathParams(r.URL.Path, "/api/secrets/{name}")
			handlers.HandleSecretsDelete(d, w, r, name)
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	})
}

func (h *HTTPServer) HandleConfigReload(w http.ResponseWriter, r *http.Request) {
	handlers.HandleConfigReload(h.deps(), w, r)
}

func (h *HTTPServer) PublishEvent(evt any) {
	h.mu.Lock()
	h.events = append(h.events, evt)
	for ch := range h.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}
	h.mu.Unlock()
}

func matchPath(path, pattern string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	pats := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(parts) != len(pats) {
		return false
	}
	for i, p := range pats {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			continue
		}
		if parts[i] != p {
			return false
		}
	}
	return true
}

func extractPathParams(path, pattern string) (id, step string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	pats := strings.Split(strings.Trim(pattern, "/"), "/")
	for i, p := range pats {
		switch p {
		case "{id}":
			id = parts[i]
		case "{step}":
			step = parts[i]
		}
	}
	return
}

func writeError(w http.ResponseWriter, code int, format string, args ...any) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf(format, args...),
	})
}

func newRunID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("run-%s-%d", hex.EncodeToString(b), time.Now().UnixMilli())
}
