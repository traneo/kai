import type { LogEntry } from '../types'
import { formatTime, levelClass } from '../utils'

interface Props {
  entry: LogEntry
  onClose: () => void
}

export function LogDetail({ entry, onClose }: Props) {
  return (
    <div className="log-detail-overlay" onClick={onClose}>
      <div className="log-detail-panel" onClick={(e) => e.stopPropagation()}>
        <div className="log-detail-header">
          <span className={`level-badge ${levelClass(entry.level)}`}>{entry.level}</span>
          <span className="log-detail-service">{entry.service}</span>
          <button className="log-detail-close" onClick={onClose}>✕</button>
        </div>
        <div className="log-detail-body">
          <div className="log-detail-field">
            <label>ID</label>
            <code>{entry.id}</code>
          </div>
          <div className="log-detail-field">
            <label>Message</label>
            <p>{entry.message}</p>
          </div>
          <div className="log-detail-field">
            <label>Timestamp</label>
            <code>{formatTime(entry.timestamp)}</code>
          </div>
          <div className="log-detail-field">
            <label>Received</label>
            <code>{entry.received_at}</code>
          </div>
          {entry.run_id && (
            <div className="log-detail-field">
              <label>Run ID</label>
              <code>{entry.run_id}</code>
            </div>
          )}
          {entry.step_id && (
            <div className="log-detail-field">
              <label>Step ID</label>
              <code>{entry.step_id}</code>
            </div>
          )}
          {entry.mission_id && (
            <div className="log-detail-field">
              <label>Mission ID</label>
              <code>{entry.mission_id}</code>
            </div>
          )}
          {entry.agent_id && (
            <div className="log-detail-field">
              <label>Agent ID</label>
              <code>{entry.agent_id}</code>
            </div>
          )}
          {entry.metadata && Object.keys(entry.metadata).length > 0 && (
            <div className="log-detail-field">
              <label>Metadata</label>
              <pre className="log-detail-json">{JSON.stringify(entry.metadata, null, 2)}</pre>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
