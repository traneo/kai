package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	"kaiplatform.com/orchestrator/internal/agentpool"
	"kaiplatform.com/orchestrator/internal/api/coordinator"
	"kaiplatform.com/orchestrator/internal/audit"
)

type statusResponse struct {
	Uptime     string `json:"uptime"`
	Agents     int    `json:"agents"`
	IdleAgents int    `json:"idle_agents"`
	BusyAgents int    `json:"busy_agents"`
	QueueDepth int    `json:"queue_depth"`
	Pipelines  int    `json:"pipelines"`
	Version    string `json:"version"`
}

func HandleStatus(d *Deps, w http.ResponseWriter, r *http.Request) {
	pool := d.Server.Pool()
	json.NewEncoder(w).Encode(statusResponse{
		Uptime:     time.Now().Format(time.RFC3339),
		Agents:     len(d.Server.Pool().ListAgents()),
		IdleAgents: pool.AgentCount(agentpool.AgentIdle),
		BusyAgents: pool.AgentCount(agentpool.AgentBusy),
		QueueDepth: pool.QueueDepth(),
		Pipelines:  len(d.Coordinator.ListRuns()),
		Version:    "0.1.0",
	})
}

type agentResponse struct {
	ID                string `json:"id"`
	Addr              string `json:"addr"`
	State             string `json:"state"`
	MissionID         string `json:"mission_id,omitempty"`
	MissionStatus     string `json:"mission_status,omitempty"`
	MissionsCompleted int    `json:"missions_completed"`
	Healthy           bool   `json:"healthy"`
	UptimeMs          int64  `json:"uptime_ms"`
	ConnectedAt       string `json:"connected_at"`
	LastHeartbeat     string `json:"last_heartbeat"`
}

func HandleAgents(d *Deps, w http.ResponseWriter, r *http.Request) {
	records := d.Server.Pool().ListAgents()
	resp := make([]agentResponse, 0, len(records))
	for _, a := range records {
		ms := kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED
		if a.MissionID != "" {
			ms = kaipb.MissionStatus_MISSION_STATUS_RUNNING
		}
		resp = append(resp, agentResponse{
			ID:                a.ID,
			Addr:              a.Addr,
			State:             string(a.State),
			MissionID:         a.MissionID,
			MissionStatus:     ms.String(),
			MissionsCompleted: a.MissionsCompleted,
			Healthy:           a.Healthy,
			UptimeMs:          a.UptimeMs,
			ConnectedAt:       a.ConnectedAt.Format(time.RFC3339),
			LastHeartbeat:     a.LastHeartbeat.Format(time.RFC3339),
		})
	}
	json.NewEncoder(w).Encode(resp)
}

func HandleEvents(d *Deps, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan any, 64)
	d.Mu.Lock()
	(*d.Subscribers)[ch] = struct{}{}
	d.Mu.Unlock()

	defer func() {
		d.Mu.Lock()
		delete(*d.Subscribers, ch)
		d.Mu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-ch:
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func HandleStats(d *Deps, w http.ResponseWriter, r *http.Request) {
	pool := d.Server.Pool()
	costTracker := d.Server.CostTracker()
	stats := costTracker.Stats()

	runCosts := costTracker.AllRuns()

	resp := map[string]any{
		"agents": map[string]int{
			"total": len(pool.ListAgents()),
			"idle":  pool.AgentCount(agentpool.AgentIdle),
			"busy":  pool.AgentCount(agentpool.AgentBusy),
		},
		"queue_depth": pool.QueueDepth(),
		"pipelines":   stats.TotalRuns,
		"steps":       stats.TotalSteps,
		"tokens": map[string]int64{
			"total":      stats.TotalTokens,
			"prompt":     stats.TotalPromptToken,
			"completion": stats.TotalCompToken,
		},
		"duration_ms": stats.TotalDurationMs,
	}

	if r.URL.Query().Get("runs") == "true" {
		type runResp struct {
			RunID       string `json:"run_id"`
			Project     string `json:"project"`
			TotalTokens int64  `json:"total_tokens"`
		}
		var runs []runResp
		for _, rc := range runCosts {
			runs = append(runs, runResp{
				RunID:       rc.RunID,
				Project:     rc.Project,
				TotalTokens: rc.TotalTokens,
			})
		}
		resp["runs"] = runs
	}

	json.NewEncoder(w).Encode(resp)
}

func HandleAudit(d *Deps, w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
	}
	runID := r.URL.Query().Get("run_id")

	auditStore := d.Coordinator.GetAuditStore()
	if auditStore == nil {
		json.NewEncoder(w).Encode([]*audit.Event{})
		return
	}

	var events []*audit.Event
	var err error
	if runID != "" {
		events, err = auditStore.Query(r.Context(), runID, limit)
	} else {
		events, err = auditStore.List(r.Context(), limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query audit: %v", err)
		return
	}
	if events == nil {
		events = []*audit.Event{}
	}
	json.NewEncoder(w).Encode(events)
}

func HandlePolicies(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(presets)
}

func HandleQueue(d *Deps, w http.ResponseWriter, r *http.Request) {
	entries := d.Server.Pool().ListQueue()
	type queueEntry struct {
		RunID    string `json:"run_id"`
		StepID   string `json:"step_id"`
		Prompt   string `json:"prompt"`
		AgentID  string `json:"agent_id"`
		Position int    `json:"position"`
	}
	resp := make([]queueEntry, 0, len(entries))
	for i, req := range entries {
		resp = append(resp, queueEntry{
			RunID:    req.RunID,
			StepID:   req.StepID,
			Prompt:   req.Prompt,
			AgentID:  req.AgentID,
			Position: i,
		})
	}
	json.NewEncoder(w).Encode(resp)
}

type poolBlob struct {
	Runner string          `json:"runner"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type poolReloadReq struct {
	Name string   `json:"name"`
	Blob poolBlob `json:"blob"`
}

type configReloadReq struct {
	Platform struct {
		Auth struct {
			PreSharedToken string `json:"pre_shared_token"`
		} `json:"auth"`
		Pool struct {
			HeartbeatTimeout string `json:"heartbeat_timeout"`
			HealthInterval   string `json:"health_interval"`
		} `json:"pool"`
	} `json:"platform"`
	Pools []poolReloadReq `json:"pools"`
}

type configApplyResult struct {
	Status          string   `json:"status"`
	AppliedAt       string   `json:"applied_at"`
	HotReloaded     []string `json:"hot_reloaded"`
	RequiresRestart []string `json:"requires_restart"`
	Errors          []string `json:"errors,omitempty"`
}

func applyConfig(d *Deps, req *configReloadReq) *configApplyResult {
	r := &configApplyResult{
		Status:          "applied",
		AppliedAt:       time.Now().UTC().Format(time.RFC3339),
		HotReloaded:     []string{},
		RequiresRestart: []string{},
	}

	if d.AuthToken != nil {
		*d.AuthToken = req.Platform.Auth.PreSharedToken
		r.HotReloaded = append(r.HotReloaded, "auth.token")
	}

	pool := d.Server.Pool()
	if req.Platform.Pool.HeartbeatTimeout != "" {
		dur, err := time.ParseDuration(req.Platform.Pool.HeartbeatTimeout)
		if err == nil {
			pool.SetHeartbeatTimeout(dur)
			r.HotReloaded = append(r.HotReloaded, "pool.heartbeat_timeout")
		} else {
			r.Errors = append(r.Errors, "invalid heartbeat_timeout: "+err.Error())
		}
	}

	if req.Platform.Pool.HealthInterval != "" {
		dur, err := time.ParseDuration(req.Platform.Pool.HealthInterval)
		if err == nil {
			pool.SetHealthInterval(dur)
			r.HotReloaded = append(r.HotReloaded, "pool.health_interval")
		} else {
			r.Errors = append(r.Errors, "invalid health_interval: "+err.Error())
		}
	}

	if d.Coordinator != nil {
		pools := make([]coordinator.AgentPoolConfig, 0, len(req.Pools))
		for _, p := range req.Pools {
			blobBytes, _ := json.Marshal(p.Blob)
			pools = append(pools, coordinator.AgentPoolConfig{
				Name:       p.Name,
				ConfigBlob: string(blobBytes),
			})
		}
		if len(pools) > 0 {
			d.Coordinator.SetAgentPools(pools)
			r.HotReloaded = append(r.HotReloaded, "pools")
		}
		d.Coordinator.SetConfigLoaded()
	}

	if len(r.HotReloaded) == 0 && len(r.Errors) == 0 {
		r.HotReloaded = append(r.HotReloaded, "config_received")
	}

	return r
}

func HandleConfigReload(d *Deps, w http.ResponseWriter, r *http.Request) {
	var req configReloadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}
	result := applyConfig(d, &req)
	json.NewEncoder(w).Encode(result)
}

func ApplyConfigFromBytes(d *Deps, data []byte) error {
	var req configReloadReq
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	applyConfig(d, &req)
	return nil
}
