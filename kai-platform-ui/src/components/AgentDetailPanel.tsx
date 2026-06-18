import type { Agent } from '../types'

interface Props {
  agent: Agent
  onClose: () => void
}

export function AgentDetailPanel({ agent, onClose }: Props) {
  const healthClass = agent.healthy ? 'healthy' : agent.state === 'offline' ? 'offline' : 'unhealthy'
  const uptime = formatUptime(agent.uptime_ms)

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="agent-detail-modal" onClick={e => e.stopPropagation()}>
        <div className="agent-detail-header">
          <div className="agent-detail-title">
            <span className={`agent-detail-dot ${agent.state}`} />
            <h3>{agent.id}</h3>
          </div>
          <button className="btn-small" onClick={onClose} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
            Close
          </button>
        </div>

        <div className="agent-detail-body">
          <div className="agent-detail-grid">
            <div className="agent-detail-field">
              <span className="agent-detail-label">State</span>
              <span className="agent-detail-value">
                <span className={`status-badge ${agent.state}`}>
                  <span className="status-dot" />
                  {agent.state}
                </span>
              </span>
            </div>

            <div className="agent-detail-field">
              <span className="agent-detail-label">Health</span>
              <span className="agent-detail-value">
                <span className={`health-indicator ${healthClass}`}>
                  <span className={`health-dot ${healthClass}`} />
                  {agent.healthy ? 'Healthy' : 'Unhealthy'}
                </span>
              </span>
            </div>

            <div className="agent-detail-field">
              <span className="agent-detail-label">Address</span>
              <span className="agent-detail-value mono">{agent.addr}</span>
            </div>

            <div className="agent-detail-field">
              <span className="agent-detail-label">Uptime</span>
              <span className="agent-detail-value">{uptime}</span>
            </div>

            <div className="agent-detail-field">
              <span className="agent-detail-label">Missions Completed</span>
              <span className="agent-detail-value">{agent.missions_completed}</span>
            </div>

            <div className="agent-detail-field">
              <span className="agent-detail-label">Connected</span>
              <span className="agent-detail-value">{new Date(agent.connected_at).toLocaleString()}</span>
            </div>

            <div className="agent-detail-field">
              <span className="agent-detail-label">Last Heartbeat</span>
              <span className="agent-detail-value">{new Date(agent.last_heartbeat).toLocaleString()}</span>
            </div>

            {agent.mission_id && (
              <div className="agent-detail-field">
                <span className="agent-detail-label">Current Mission</span>
                <span className="agent-detail-value mono">{agent.mission_id}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function formatUptime(ms: number): string {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  const parts: string[] = []
  if (days > 0) parts.push(`${days}d`)
  if (hours % 24 > 0) parts.push(`${hours % 24}h`)
  if (minutes % 60 > 0) parts.push(`${minutes % 60}m`)
  if (seconds % 60 > 0 || parts.length === 0) parts.push(`${seconds % 60}s`)

  return parts.join(' ')
}
