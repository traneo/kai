package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kaiplatform.com/orchestrator/internal/api/coordinator"
	"kaiplatform.com/orchestrator/internal/workflow"
)

type pipelineRunResponse struct {
	ID         string `json:"id"`
	Project    string `json:"project"`
	Status     string `json:"status"`
	Steps      int    `json:"steps"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	HasBlocked bool   `json:"has_blocked"`
	HasQueued  bool   `json:"has_queued"`
	CreatedAt  string `json:"created_at"`
}

func HandlePipelines(d *Deps, w http.ResponseWriter, r *http.Request) {
	runs := d.Coordinator.ListRuns()

	queue := d.Server.Pool().ListQueue()
	queuedRunIDs := make(map[string]bool, len(queue))
	for _, req := range queue {
		queuedRunIDs[req.RunID] = true
	}

	resp := make([]pipelineRunResponse, 0, len(runs))
	for _, run := range runs {
		snap := run.Snapshot()
		passed, failed := 0, 0
		hasBlocked := false
		for _, s := range snap.StepStates {
			if s.Status == workflow.StepPassed {
				passed++
			} else if s.Status == workflow.StepFailed {
				failed++
			} else if s.Status == workflow.StepBlocked {
				hasBlocked = true
			}
		}
		resp = append(resp, pipelineRunResponse{
			ID:         snap.ID,
			Project:    snap.Pipeline.Project,
			Status:     string(snap.Status),
			Steps:      len(snap.StepStates),
			Passed:     passed,
			Failed:     failed,
			HasBlocked: hasBlocked,
			HasQueued:  queuedRunIDs[snap.ID],
			CreatedAt:  snap.CreatedAt.Format(time.RFC3339),
		})
	}
	json.NewEncoder(w).Encode(resp)
}

type gateResultResponse struct {
	Gate     string `json:"gate"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Duration string `json:"duration"`
}

type policyResponse struct {
	AllowedDirs       []string `json:"allowed_dirs,omitempty"`
	AllowedTools      []string `json:"allowed_tools,omitempty"`
	AllowedCommands   []string `json:"allowed_commands,omitempty"`
	Agent             string   `json:"agent,omitempty"`
	MaxRetries        int      `json:"max_retries,omitempty"`
	RetryDelaySeconds int      `json:"retry_delay_seconds,omitempty"`
	RetryBackoff      string   `json:"retry_backoff,omitempty"`
	TimeoutSeconds    int      `json:"timeout_seconds,omitempty"`
}

type stepDetailResponse struct {
	ID          string               `json:"id"`
	Prompt      string               `json:"prompt"`
	Status      string               `json:"status"`
	DependsOn   []string             `json:"depends_on"`
	Validation  []string             `json:"validation"`
	Approval    string               `json:"approval"`
	Retries     int                  `json:"retries"`
	MaxRetries  int                  `json:"max_retries"`
	NextRetryAt *string              `json:"next_retry_at,omitempty"`
	AssignedTo  string               `json:"assigned_to"`
	Error       string               `json:"error,omitempty"`
	StartedAt   *string              `json:"started_at,omitempty"`
	Policy      policyResponse       `json:"policy,omitempty"`
	GateResults []gateResultResponse `json:"gate_results,omitempty"`
}

type pipelineDetailResponse struct {
	ID        string               `json:"id"`
	Project   string               `json:"project"`
	Status    string               `json:"status"`
	Steps     []stepDetailResponse `json:"steps"`
	CreatedAt string               `json:"created_at"`
	UpdatedAt string               `json:"updated_at"`
	Error     string               `json:"error,omitempty"`
	OutputURL string               `json:"output_url,omitempty"`
	OutputSHA string               `json:"output_sha,omitempty"`
}

func HandlePipelineDetail(d *Deps, w http.ResponseWriter, r *http.Request, id string) {
	run := d.Coordinator.GetRun(id)
	if run == nil {
		writeError(w, http.StatusNotFound, "pipeline %s not found", id)
		return
	}
	snap := run.Snapshot()

	std := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		s := t.Format(time.RFC3339)
		return &s
	}

	steps := make([]stepDetailResponse, 0, len(snap.Pipeline.Steps))
	for _, s := range snap.Pipeline.Steps {
		state := snap.StepStates[s.ID]
		gates := make([]gateResultResponse, 0, len(state.GateResults))
		for _, g := range state.GateResults {
			gates = append(gates, gateResultResponse{
				Gate:     g.Gate,
				Status:   g.Status,
				Message:  g.Message,
				Duration: g.Duration,
			})
		}
		var nextRetryAt *string
		if state.NextRetryAt != nil {
			s := state.NextRetryAt.Format(time.RFC3339)
			nextRetryAt = &s
		}

		detail := stepDetailResponse{
			ID:          s.ID,
			Prompt:      s.Prompt,
			Status:      string(state.Status),
			DependsOn:   s.DependsOn,
			Validation:  s.Validation,
			Approval:    s.Approval,
			Retries:     state.Retries,
			MaxRetries:  s.Policy.MaxRetries,
			NextRetryAt: nextRetryAt,
			AssignedTo:  state.AssignedTo,
			Error:       state.Error,
			StartedAt:   std(state.StartedAt),
			Policy: policyResponse{
				AllowedDirs:       s.Policy.AllowedDirs,
				AllowedTools:      s.Policy.AllowedTools,
				AllowedCommands:   s.Policy.AllowedCommands,
				Agent:             s.Policy.Agent,
				MaxRetries:        s.Policy.MaxRetries,
				RetryDelaySeconds: s.Policy.RetryDelaySeconds,
				RetryBackoff:      s.Policy.RetryBackoff,
				TimeoutSeconds:    s.Policy.TimeoutSeconds,
			},
			GateResults: gates,
		}
		steps = append(steps, detail)
	}

	resp := pipelineDetailResponse{
		ID:        snap.ID,
		Project:   snap.Pipeline.Project,
		Status:    string(snap.Status),
		Steps:     steps,
		CreatedAt: snap.CreatedAt.Format(time.RFC3339),
		UpdatedAt: snap.UpdatedAt.Format(time.RFC3339),
		OutputURL: snap.OutputURL,
		OutputSHA: snap.OutputSHA,
	}
	if snap.Error != "" {
		resp.Error = snap.Error
	}
	json.NewEncoder(w).Encode(resp)
}

type createPipelineRequest struct {
	YAML string `json:"yaml"`
}

func HandleCreatePipeline(d *Deps, w http.ResponseWriter, r *http.Request) {
	var req createPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}

	p, err := workflow.ParsePipelineBytes([]byte(req.YAML))
	if err != nil {
		writeError(w, http.StatusBadRequest, "parse pipeline: %v", err)
		return
	}

	id := newRunID()
	run, err := d.Coordinator.CreateRun(id, p, req.YAML)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create run: %v", err)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": string(run.Snapshot().Status), "project": p.Project})
}

type approveRequest struct {
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
}

func HandleApprove(d *Deps, w http.ResponseWriter, r *http.Request, id, step string) {
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}

	run := d.Coordinator.GetRun(id)
	if run == nil {
		writeError(w, http.StatusNotFound, "pipeline %s not found", id)
		return
	}

	snap := run.Snapshot()
	state, ok := snap.StepStates[step]
	if !ok {
		writeError(w, http.StatusNotFound, "step %s not found", step)
		return
	}

	if state.Status != workflow.StepBlocked {
		writeError(w, http.StatusBadRequest, "step %s is not blocked (status: %s)", step, state.Status)
		return
	}

	if req.Action == "approve" {
		run.ValidateStep(step, true, "approved by human")
		d.Coordinator.AuditLogApprove(id, step, "approved by human")
		d.PublishEvent(map[string]any{"type": "step_approved", "run_id": id, "step_id": step})
		d.Coordinator.CheckNextSteps(run)
		json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
	} else if req.Action == "reject" {
		run.ValidateStep(step, false, fmt.Sprintf("rejected by human: %s", req.Message))
		d.Coordinator.AuditLogReject(id, step, req.Message)
		d.PublishEvent(map[string]any{"type": "step_rejected", "run_id": id, "step_id": step})
		json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
	} else {
		writeError(w, http.StatusBadRequest, "action must be 'approve' or 'reject'")
	}
}

func HandleCancel(d *Deps, w http.ResponseWriter, r *http.Request, id string) {
	if err := d.Coordinator.CancelRun(id); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func HandlePipelineYAML(d *Deps, w http.ResponseWriter, r *http.Request, id string) {
	run := d.Coordinator.GetRun(id)
	if run == nil {
		writeError(w, http.StatusNotFound, "pipeline %s not found", id)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(run.Snapshot().RawYAML))
}

func HandleConversation(d *Deps, w http.ResponseWriter, r *http.Request, runID, stepID string) {
	convStore := d.Coordinator.GetConvStore()
	if convStore == nil {
		writeError(w, http.StatusNotFound, "conversation store not available")
		return
	}

	run := d.Coordinator.GetRun(runID)
	if run == nil {
		writeError(w, http.StatusNotFound, "pipeline %s not found", runID)
		return
	}

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	entries := convStore.List(runID, stepID, limit)
	if entries == nil {
		entries = []*coordinator.ConversationEntry{}
	}

	json.NewEncoder(w).Encode(entries)
}
