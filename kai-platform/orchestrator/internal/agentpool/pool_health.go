package agentpool

import (
	"context"
	"log"
	"time"

	kaipb "kaiplatform.com/gen/kaiplatform/v1"
)

func (p *Pool) healthCheckLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.healthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkAllAgents()
		}
	}
}

func (p *Pool) checkAllAgents() {
	p.mu.RLock()
	snap := make([]*AgentRecord, 0, len(p.agents))
	for _, rec := range p.agents {
		snap = append(snap, rec)
	}
	p.mu.RUnlock()

	for _, rec := range snap {
		if rec.State == AgentOffline {
			continue
		}
		client, err := p.AgentClient(rec.Addr)
		if err != nil {
			log.Printf("health check: %s connect error: %v", rec.ID, err)
			p.markUnhealthy(rec.ID)
			continue
		}
		ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
		hs, err := client.HealthCheck(ctx, &kaipb.Empty{})
		cancel()
		if err != nil {
			log.Printf("health check: %s rpc error: %v", rec.ID, err)
			p.markUnhealthy(rec.ID)
			continue
		}
		p.mu.Lock()
		if r, ok := p.agents[rec.ID]; ok {
			r.Healthy = hs.Healthy
			r.UptimeMs = hs.UptimeMs
			if hs.Healthy && r.State == AgentUnhealthy {
				r.State = AgentIdle
				log.Printf("agent pool: %s recovered, now idle", rec.ID)
				p.signalIdle()
			}
		}
		p.mu.Unlock()
	}
}

func (p *Pool) markUnhealthy(agentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if rec, ok := p.agents[agentID]; ok {
		if rec.State != AgentOffline && rec.State != AgentUnhealthy {
			rec.State = AgentUnhealthy
			log.Printf("agent pool: %s marked unhealthy", agentID)
		}
	}
}

func (p *Pool) pruneStaleLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.heartbeatTimeout)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.pruneStale()
		}
	}
}

func (p *Pool) pruneStale() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for id, rec := range p.agents {
		if rec.State == AgentOffline {
			continue
		}
		if now.Sub(rec.LastHeartbeat) > p.heartbeatTimeout {
			log.Printf("agent pool: %s stale (last heartbeat %v ago)", id, now.Sub(rec.LastHeartbeat))
			rec.State = AgentOffline
		}
	}
}
