package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body, dst any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, r)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

type StatusResponse struct {
	Uptime     string `json:"uptime"`
	Agents     int    `json:"agents"`
	IdleAgents int    `json:"idle_agents"`
	BusyAgents int    `json:"busy_agents"`
	QueueDepth int    `json:"queue_depth"`
	Pipelines  int    `json:"pipelines"`
	Version    string `json:"version"`
}

type AgentResponse struct {
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

type PipelineRunResponse struct {
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

type GateResultResponse struct {
	Gate     string `json:"gate"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Duration string `json:"duration"`
}

type PolicyResponse struct {
	AllowedDirs       []string `json:"allowed_dirs,omitempty"`
	AllowedTools      []string `json:"allowed_tools,omitempty"`
	AllowedCommands   []string `json:"allowed_commands,omitempty"`
	Agent             string   `json:"agent,omitempty"`
	MaxRetries        int      `json:"max_retries,omitempty"`
	RetryDelaySeconds int      `json:"retry_delay_seconds,omitempty"`
	RetryBackoff      string   `json:"retry_backoff,omitempty"`
	TimeoutSeconds    int      `json:"timeout_seconds,omitempty"`
}

type StepDetailResponse struct {
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
	Policy      PolicyResponse       `json:"policy,omitempty"`
	GateResults []GateResultResponse `json:"gate_results,omitempty"`
}

type PipelineDetailResponse struct {
	ID        string               `json:"id"`
	Project   string               `json:"project"`
	Status    string               `json:"status"`
	Steps     []StepDetailResponse `json:"steps"`
	CreatedAt string               `json:"created_at"`
	UpdatedAt string               `json:"updated_at"`
	Error     string               `json:"error,omitempty"`
	OutputURL string               `json:"output_url,omitempty"`
	OutputSHA string               `json:"output_sha,omitempty"`
}

type AuditEventResponse struct {
	ID      int64  `json:"id"`
	Time    string `json:"time"`
	Type    string `json:"type"`
	RunID   string `json:"run_id,omitempty"`
	StepID  string `json:"step_id,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	Message string `json:"message,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

type StatsResponse struct {
	Agents     map[string]int `json:"agents"`
	QueueDepth int            `json:"queue_depth"`
	Pipelines  int            `json:"pipelines"`
	Steps      int            `json:"steps"`
	Tokens     map[string]any `json:"tokens"`
	DurationMs int64          `json:"duration_ms"`
	Runs       []RunStats     `json:"runs,omitempty"`
}

type RunStats struct {
	RunID       string `json:"run_id"`
	Project     string `json:"project"`
	TotalTokens int64  `json:"total_tokens"`
}

type ConversationEntry struct {
	MissionID string `json:"mission_id"`
	RunID     string `json:"run_id"`
	StepID    string `json:"step_id"`
	Sequence  int64  `json:"sequence"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}

func (c *Client) GetStatus() (*StatusResponse, error) {
	var s StatusResponse
	if err := c.do("GET", "/api/status", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) ListAgents() ([]AgentResponse, error) {
	var agents []AgentResponse
	if err := c.do("GET", "/api/agents", nil, &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

func (c *Client) ListPipelines() ([]PipelineRunResponse, error) {
	var runs []PipelineRunResponse
	if err := c.do("GET", "/api/pipelines", nil, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (c *Client) GetPipeline(id string) (*PipelineDetailResponse, error) {
	var p PipelineDetailResponse
	if err := c.do("GET", "/api/pipelines/"+id, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) CreatePipeline(yaml string) (map[string]any, error) {
	var result map[string]any
	if err := c.do("POST", "/api/pipelines", map[string]string{"yaml": yaml}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CancelPipeline(id string) (map[string]any, error) {
	var result map[string]any
	if err := c.do("POST", "/api/pipelines/"+id+"/cancel", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ApproveStep(id, step, action, message string) (map[string]any, error) {
	body := map[string]string{"action": action}
	if message != "" {
		body["message"] = message
	}
	var result map[string]any
	if err := c.do("POST", "/api/pipelines/"+id+"/steps/"+step+"/approve", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetConversation(runID, stepID string, limit int) ([]ConversationEntry, error) {
	path := fmt.Sprintf("/api/pipelines/%s/steps/%s/conversation?limit=%d", runID, stepID, limit)
	var entries []ConversationEntry
	if err := c.do("GET", path, nil, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *Client) GetStats(runs bool) (*StatsResponse, error) {
	path := "/api/stats"
	if runs {
		path += "?runs=true"
	}
	var s StatsResponse
	if err := c.do("GET", path, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) ListAudit(limit int, runID string) ([]AuditEventResponse, error) {
	path := fmt.Sprintf("/api/audit?limit=%d", limit)
	if runID != "" {
		path += "&run_id=" + runID
	}
	var events []AuditEventResponse
	if err := c.do("GET", path, nil, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) StreamEvents(onEvent func(map[string]any)) error {
	req, err := http.NewRequest("GET", c.baseURL+"/api/events", nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var evt map[string]any
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}
		onEvent(evt)
	}
	return scanner.Err()
}
