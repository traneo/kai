package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	"kaiplatform.com/orchestrator/internal/agentpool"
	"kaiplatform.com/orchestrator/internal/archive"
	"kaiplatform.com/orchestrator/internal/audit"
	"kaiplatform.com/orchestrator/internal/gitops"
	"kaiplatform.com/orchestrator/internal/gitprovider"
	"kaiplatform.com/orchestrator/internal/secrets"
	"kaiplatform.com/orchestrator/internal/validation"
	"kaiplatform.com/orchestrator/internal/workflow"
)

type EventPublisher func(any)

type AgentPoolConfig struct {
	Name       string `json:"name"`
	ConfigBlob string `json:"config_blob"` // opaque JSON blob passed to agent
}

type ConversationEntry struct {
	MissionID string    `json:"mission_id"`
	RunID     string    `json:"run_id"`
	StepID    string    `json:"step_id"`
	Sequence  int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

type ConversationStore interface {
	Append(entry *ConversationEntry)
	List(runID, stepID string, limit int) []*ConversationEntry
	Close()
}

type SecretMeta struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SecretInput struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type SecretStore interface {
	List(ctx context.Context) ([]SecretMeta, error)
	GetValue(ctx context.Context, name string) (string, error)
	Set(ctx context.Context, name, value, description string) error
	Delete(ctx context.Context, name string) error
}

type ServerIface interface {
	AssignMissionToAgent(ctx context.Context, agentID, missionID, prompt string, policy *workflow.Policy, ws *kaipb.Workspace) error
}

type Logger interface {
	Debugf(format string, args ...any)
	Printf(format string, args ...any)
}

type defaultLogger struct{}

func (defaultLogger) Debugf(format string, args ...any) {}
func (defaultLogger) Printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

type Coordinator struct {
	mu             sync.Mutex
	runs           map[string]*workflow.Run
	gitClients     map[string]*gitops.Client
	valRunner      *validation.Runner
	pool           *agentpool.Pool
	server         ServerIface
	eventFn        EventPublisher
	auditStore     audit.Store
	convStore      ConversationStore
	archiveStore   archive.Store
	secMgr         secrets.Manager
	secretStore    SecretStore
	gitProviderReg *gitprovider.Registry

	agentPools   map[string]*AgentPoolConfig
	agentPoolsMu sync.RWMutex

	configLoaded   bool
	configLoadedMu sync.RWMutex

	log Logger
}

func (c *Coordinator) SetConfigLoaded() {
	c.configLoadedMu.Lock()
	defer c.configLoadedMu.Unlock()
	c.configLoaded = true
}

func (c *Coordinator) ConfigLoaded() bool {
	c.configLoadedMu.RLock()
	defer c.configLoadedMu.RUnlock()
	return c.configLoaded
}

func (c *Coordinator) SetAgentPools(pools []AgentPoolConfig) {
	c.agentPoolsMu.Lock()
	defer c.agentPoolsMu.Unlock()
	c.agentPools = make(map[string]*AgentPoolConfig, len(pools))
	for i := range pools {
		p := &pools[i]
		c.agentPools[p.Name] = p
	}
}

func (c *Coordinator) GetAgentPool(name string) (*AgentPoolConfig, bool) {
	c.agentPoolsMu.RLock()
	defer c.agentPoolsMu.RUnlock()
	p, ok := c.agentPools[name]
	return p, ok
}

func (c *Coordinator) SetAuditStore(store audit.Store) {
	c.auditStore = store
}

func (c *Coordinator) SetConversationStore(store ConversationStore) {
	c.convStore = store
}

func (c *Coordinator) SetArchiveStore(store archive.Store) {
	c.archiveStore = store
}

func (c *Coordinator) SetSecretsManager(mgr secrets.Manager) {
	c.secMgr = mgr
}

func (c *Coordinator) SetGitProviderRegistry(reg *gitprovider.Registry) {
	c.gitProviderReg = reg
}

func (c *Coordinator) SetSecretStore(store SecretStore) {
	c.secretStore = store
}

func (c *Coordinator) SetLogger(log Logger) {
	c.log = log
}

func (c *Coordinator) logf(format string, args ...any) {
	if c.log != nil {
		c.log.Debugf(format, args...)
	}
}

func (c *Coordinator) printf(format string, args ...any) {
	if c.log != nil {
		c.log.Printf(format, args...)
	}
}

func (c *Coordinator) auditLog(evtType audit.EventType, runID, stepID, agentID, message string, payload any) {
	if c.auditStore != nil {
		audit.Log(c.auditStore, evtType, runID, stepID, agentID, message, payload)
	}
}

func (c *Coordinator) SetEventPublisher(pub EventPublisher) {
	c.eventFn = pub
}

func (c *Coordinator) publish(evt any) {
	if c.eventFn != nil {
		c.eventFn(evt)
	}
}

func NewCoordinator(valRunner *validation.Runner, pool *agentpool.Pool) *Coordinator {
	return &Coordinator{
		runs:       make(map[string]*workflow.Run),
		gitClients: make(map[string]*gitops.Client),
		valRunner:  valRunner,
		pool:       pool,
	}
}

func (c *Coordinator) SetServer(srv ServerIface) {
	c.server = srv
}

func (c *Coordinator) CreateRun(id string, p *workflow.Pipeline, rawYAML string) (*workflow.Run, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	run, err := workflow.NewRun(id, p)
	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}
	run.RawYAML = rawYAML
	run.Start()
	c.runs[id] = run

	c.auditLog(audit.EventPipelineCreated, id, "", "", "", map[string]any{"project": p.Project})
	c.publish(map[string]any{"type": "pipeline_created", "id": id, "project": p.Project})

	c.logf("RUN %s: created (project=%s, steps=%d)", id, p.Project, len(p.Steps))
	for _, s := range p.Steps {
		c.logf("RUN %s:   step %q: validation=%v approval=%q deps=%v", id, s.ID, s.Validation, s.Approval, s.DependsOn)
	}

	go c.SetupWorkspace(id, run, p)

	return run, nil
}

func (c *Coordinator) GetRun(id string) *workflow.Run {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.runs[id]
}

func (c *Coordinator) GetGitClient(runID string) *gitops.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gitClients[runID]
}

func (c *Coordinator) GetConvStore() ConversationStore {
	return c.convStore
}

func (c *Coordinator) GetAuditStore() audit.Store {
	return c.auditStore
}

func (c *Coordinator) ListRuns() []*workflow.Run {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]*workflow.Run, 0, len(c.runs))
	for _, run := range c.runs {
		result = append(result, run)
	}
	return result
}
