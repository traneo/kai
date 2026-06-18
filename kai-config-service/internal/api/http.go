package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kaiplatform/config/internal/config"
	"github.com/kaiplatform/config/internal/store"
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	writeJSON := func(v any) {
		json.NewEncoder(w).Encode(v)
	}

	writeError := func(code int, msg string) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
	}

	path := r.URL.Path

	switch {

	case r.Method == "GET" && path == "/api/v1/config":
		v, err := h.store.GetActiveVersion()
		if err != nil {
			writeError(http.StatusNotFound, "no active config")
			return
		}
		writeJSON(v.Config)

	case r.Method == "GET" && path == "/api/v1/config/versions":
		status := r.URL.Query().Get("status")
		versions, err := h.store.ListVersions(status)
		if err != nil {
			writeError(http.StatusInternalServerError, err.Error())
			return
		}
		if versions == nil {
			versions = []*config.ConfigVersion{}
		}
		writeJSON(versions)

	case r.Method == "POST" && path == "/api/v1/config/versions":
		v, err := h.store.CreateDraft()
		if err != nil {
			writeError(http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(v)

	case r.Method == "GET" && match(path, "/api/v1/config/versions/{id}"):
		id := extract(path, "/api/v1/config/versions/{id}")
		v, err := h.store.GetVersion(id)
		if err != nil {
			writeError(http.StatusNotFound, "version not found")
			return
		}
		writeJSON(v)

	case r.Method == "PUT" && match(path, "/api/v1/config/versions/{id}"):
		id := extract(path, "/api/v1/config/versions/{id}")
		var req struct {
			Config  json.RawMessage `json:"config"`
			Message string          `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(http.StatusBadRequest, "invalid json")
			return
		}
		v, err := h.store.UpdateDraft(id, req.Config, req.Message)
		if err != nil {
			writeError(http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(v)

	case r.Method == "POST" && match(path, "/api/v1/config/versions/{id}/publish"):
		id := extract(path, "/api/v1/config/versions/{id}/publish")
		v, err := h.store.Publish(id)
		if err != nil {
			writeError(http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(v)

	case r.Method == "POST" && match(path, "/api/v1/config/versions/{id}/activate"):
		id := extract(path, "/api/v1/config/versions/{id}/activate")
		result, err := h.store.Activate(id)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(result)
			return
		}
		writeJSON(result)

	case r.Method == "POST" && match(path, "/api/v1/config/versions/{id}/rollback"):
		id := extract(path, "/api/v1/config/versions/{id}/rollback")
		v, err := h.store.Rollback(id)
		if err != nil {
			writeError(http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(v)

	case r.Method == "GET" && path == "/api/v1/config/status":
		active, err := h.store.GetActiveVersion()
		status := map[string]any{"service": "config"}
		if err != nil {
			status["active_version"] = nil
		} else {
			status["active_version"] = active.Version
			status["active_id"] = active.ID
			status["active_updated"] = active.UpdatedAt
		}
		versions, _ := h.store.ListVersions("")
		status["total_versions"] = len(versions)
		writeJSON(status)

	default:
		writeError(http.StatusNotFound, "not found")
	}
}

func match(path, pattern string) bool {
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

func extract(path, pattern string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	pats := strings.Split(strings.Trim(pattern, "/"), "/")
	for i, p := range pats {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			if i < len(parts) {
				return parts[i]
			}
		}
	}
	return ""
}
