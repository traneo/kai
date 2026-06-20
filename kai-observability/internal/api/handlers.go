package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"kaiplatform.com/observability/internal/models"
	"kaiplatform.com/observability/internal/store"
)

type Handlers struct {
	store store.Store
	hub   *SSEHub
}

func NewHandlers(s store.Store, hub *SSEHub) *Handlers {
	return &Handlers{store: s, hub: hub}
}

type logRequestBody struct {
	Service   string         `json:"service"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Timestamp int64          `json:"timestamp,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	StepID    string         `json:"step_id,omitempty"`
	MissionID string         `json:"mission_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func (h *Handlers) HandlePostLog(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	entries, err := parseLogBody(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.Append(r.Context(), entries); err != nil {
		log.Printf("store append: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, e := range entries {
		h.hub.Publish(e)
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"accepted": len(entries)})
}

func (h *Handlers) HandlePostBatch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	var req struct {
		Entries []logRequestBody `json:"entries"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}
	entries := make([]models.LogEntry, 0, len(req.Entries))
	for _, rb := range req.Entries {
		e, err := toLogEntry(rb)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid entry: %v", err), http.StatusBadRequest)
			return
		}
		entries = append(entries, e)
	}
	if err := h.store.Append(r.Context(), entries); err != nil {
		log.Printf("store append batch: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, e := range entries {
		h.hub.Publish(e)
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"accepted": len(entries)})
}

func (h *Handlers) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/logs/")
	if id != "" && id != r.URL.Path {
		h.HandleGetLogByID(w, r, id)
		return
	}

	q := r.URL.Query()
	filter := models.QueryFilter{
		Service: q.Get("service"),
		Search:  q.Get("search"),
		RunID:   q.Get("run_id"),
		StepID:  q.Get("step_id"),
		MissionID: q.Get("mission_id"),
		AgentID: q.Get("agent_id"),
		Limit:   100,
		Offset:  0,
	}
	if lvl := q.Get("level"); lvl != "" {
		if v, ok := models.ValidLevel(lvl); ok {
			filter.Level = v
		}
	}
	if t := q.Get("from"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			filter.From = parsed
		}
	}
	if t := q.Get("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			filter.To = parsed
		}
	}
	fmt.Sscanf(q.Get("limit"), "%d", &filter.Limit)
	fmt.Sscanf(q.Get("offset"), "%d", &filter.Offset)

	entries, err := h.store.Query(r.Context(), filter)
	if err != nil {
		log.Printf("store query: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []models.LogEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *Handlers) HandleGetSummaries(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.store.RunSummaries(r.Context())
	if err != nil {
		log.Printf("store run summaries: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if summaries == nil {
		summaries = []models.RunSummary{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

func (h *Handlers) HandleGetLogByID(w http.ResponseWriter, r *http.Request, id string) {
	entry, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("store get by id: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if entry == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func (h *Handlers) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := h.hub.Subscribe()
	defer h.hub.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-ch:
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func parseLogBody(body []byte) ([]models.LogEntry, error) {
	var single logRequestBody
	if err := json.Unmarshal(body, &single); err == nil && single.Message != "" {
		e, err := toLogEntry(single)
		if err != nil {
			return nil, err
		}
		return []models.LogEntry{e}, nil
	}

	var batch struct {
		Entries []logRequestBody `json:"entries"`
	}
	if err := json.Unmarshal(body, &batch); err == nil && len(batch.Entries) > 0 {
		entries := make([]models.LogEntry, 0, len(batch.Entries))
		for _, rb := range batch.Entries {
			e, err := toLogEntry(rb)
			if err != nil {
				return nil, err
			}
			entries = append(entries, e)
		}
		return entries, nil
	}

	var arr []logRequestBody
	if err := json.Unmarshal(body, &arr); err == nil && len(arr) > 0 {
		entries := make([]models.LogEntry, 0, len(arr))
		for _, rb := range arr {
			e, err := toLogEntry(rb)
			if err != nil {
				return nil, err
			}
			entries = append(entries, e)
		}
		return entries, nil
	}

	return nil, fmt.Errorf("invalid log entry body")
}

func toLogEntry(rb logRequestBody) (models.LogEntry, error) {
	level, ok := models.ValidLevel(rb.Level)
	if !ok {
		level = models.LevelInfo
	}
	ts := rb.Timestamp
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	return models.LogEntry{
		ID:         fmt.Sprintf("%s-%s-%d", rb.Service, strings.ReplaceAll(rb.Message, " ", "_"), ts),
		Service:    rb.Service,
		Level:      level,
		Message:    rb.Message,
		Timestamp:  ts,
		RunID:      rb.RunID,
		StepID:     rb.StepID,
		MissionID:  rb.MissionID,
		AgentID:    rb.AgentID,
		Metadata:   rb.Metadata,
		ReceivedAt: time.Now(),
	}, nil
}
