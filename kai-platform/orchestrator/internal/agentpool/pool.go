package agentpool

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
)

type AgentState string

const (
	AgentIdle      AgentState = "idle"
	AgentBusy      AgentState = "busy"
	AgentUnhealthy AgentState = "unhealthy"
	AgentOffline   AgentState = "offline"
)

type AgentRecord struct {
	ID                string     `json:"id"`
	Addr              string     `json:"addr"`
	State             AgentState `json:"state"`
	MissionID         string     `json:"mission_id,omitempty"`
	LastHeartbeat     time.Time  `json:"last_heartbeat"`
	MissionsCompleted int        `json:"missions_completed"`
	Healthy           bool       `json:"healthy"`
	UptimeMs          int64      `json:"uptime_ms"`
	ConnectedAt       time.Time  `json:"connected_at"`
}

type MissionRequest struct {
	RunID   string
	StepID  string
	Prompt  string
	AgentID string
}

type Pool struct {
	mu           sync.RWMutex
	agents       map[string]*AgentRecord
	agentClients map[string]kaipb.AgentClient
	missionQueue []*MissionRequest
	idleSignal   chan struct{}

	heartbeatTimeout time.Duration
	healthInterval   time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type Config struct {
	HeartbeatTimeout time.Duration
	HealthInterval   time.Duration
}

func DefaultConfig() Config {
	return Config{
		HeartbeatTimeout: 30 * time.Second,
		HealthInterval:   10 * time.Second,
	}
}

func New(cfg Config) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		agents:           make(map[string]*AgentRecord),
		agentClients:     make(map[string]kaipb.AgentClient),
		idleSignal:       make(chan struct{}, 1),
		heartbeatTimeout: cfg.HeartbeatTimeout,
		healthInterval:   cfg.HealthInterval,
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (p *Pool) Start() {
	p.wg.Add(2)
	go p.healthCheckLoop()
	go p.pruneStaleLoop()
	log.Printf("agent pool started (heartbeat timeout=%v, health interval=%v)", p.heartbeatTimeout, p.healthInterval)
}

func (p *Pool) Stop() {
	p.cancel()
	p.wg.Wait()
	p.mu.Lock()
	for addr, client := range p.agentClients {
		_ = client // close eventually
		_ = addr
	}
	p.mu.Unlock()
	log.Print("agent pool stopped")
}

func (p *Pool) RegisterOrUpdate(agentID, addr, missionID string, status kaipb.MissionStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	record, exists := p.agents[agentID]
	if !exists {
		record = &AgentRecord{
			ID:          agentID,
			Addr:        addr,
			State:       AgentIdle,
			ConnectedAt: now,
		}
		p.agents[agentID] = record
		log.Printf("agent pool: new agent %s registered at %s", agentID, addr)
	}

	record.Addr = addr
	record.LastHeartbeat = now

	prevMissionID := record.MissionID
	record.MissionID = missionID

	switch {
	case missionID == "" && prevMissionID != "":
		record.State = AgentIdle
		p.signalIdle()
	case missionID == "" && prevMissionID == "":
		record.State = AgentIdle
		p.signalIdle()
	case missionID != "":
		record.State = AgentBusy
	}

	switch status {
	case kaipb.MissionStatus_MISSION_STATUS_COMPLETED:
		record.MissionsCompleted++
		record.State = AgentIdle
		record.MissionID = ""
		p.signalIdle()
	case kaipb.MissionStatus_MISSION_STATUS_FAILED:
		record.MissionsCompleted++
		record.State = AgentIdle
		record.MissionID = ""
		p.signalIdle()
	}
}

func (p *Pool) signalIdle() {
	select {
	case p.idleSignal <- struct{}{}:
	default:
	}
}

func (p *Pool) GetIdleAgent() (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for id, rec := range p.agents {
		if rec.State == AgentIdle {
			return id, true
		}
	}
	return "", false
}

func (p *Pool) AssignMission(agentID, missionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if rec, ok := p.agents[agentID]; ok {
		rec.State = AgentBusy
		rec.MissionID = missionID
	}
}

func (p *Pool) CompleteMission(agentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if rec, ok := p.agents[agentID]; ok {
		rec.State = AgentIdle
		rec.MissionID = ""
		rec.MissionsCompleted++
		p.signalIdle()
	}
}

func (p *Pool) EnqueueMission(req *MissionRequest) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.missionQueue = append(p.missionQueue, req)
	log.Printf("agent pool: queued run=%s step=%s (queue depth: %d)", req.RunID, req.StepID, len(p.missionQueue))
}

func (p *Pool) DequeueMission() *MissionRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.missionQueue) == 0 {
		return nil
	}
	req := p.missionQueue[0]
	p.missionQueue = p.missionQueue[1:]
	return req
}

func (p *Pool) ListQueue() []*MissionRequest {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*MissionRequest, len(p.missionQueue))
	copy(result, p.missionQueue)
	return result
}

func (p *Pool) QueueDepth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.missionQueue)
}

func (p *Pool) AgentCount(state AgentState) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	count := 0
	for _, rec := range p.agents {
		if rec.State == state {
			count++
		}
	}
	return count
}

func (p *Pool) ListAgents() []*AgentRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*AgentRecord, 0, len(p.agents))
	for _, rec := range p.agents {
		result = append(result, rec)
	}
	return result
}

func (p *Pool) GetAgentByMission(missionID string) *AgentRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, rec := range p.agents {
		if rec.MissionID == missionID {
			return rec
		}
	}
	return nil
}

func (p *Pool) GetAgent(id string) *AgentRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.agents[id]
}

func (p *Pool) AgentClient(addr string) (kaipb.AgentClient, error) {
	if client, ok := p.agentClients[addr]; ok {
		return client, nil
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial agent %s: %w", addr, err)
	}
	client := kaipb.NewAgentClient(conn)
	p.agentClients[addr] = client
	return client, nil
}



func (p *Pool) WaitForIdleAgent(ctx context.Context) string {
	for {
		if id, ok := p.GetIdleAgent(); ok {
			return id
		}
		select {
		case <-p.idleSignal:
			if id, ok := p.GetIdleAgent(); ok {
				return id
			}
		case <-ctx.Done():
			return ""
		}
	}
}

func (p *Pool) SetHeartbeatTimeout(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.heartbeatTimeout = d
}

func (p *Pool) SetHealthInterval(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthInterval = d
}
