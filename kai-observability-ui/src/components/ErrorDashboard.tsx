import { useState, useEffect } from 'react'
import type { LogEntry } from '../types'
import { fetchLogs } from '../api'
import { formatTime, levelClass } from '../utils'
import { MetricsCard } from './MetricsCard'
import { LogDetail } from './LogDetail'

export function ErrorDashboard() {
  const [errors, setErrors] = useState<LogEntry[]>([])
  const [warnings, setWarnings] = useState<LogEntry[]>([])
  const [selected, setSelected] = useState<LogEntry | null>(null)

  useEffect(() => {
    Promise.all([
      fetchLogs({ level: 'error', limit: 200 }),
      fetchLogs({ level: 'warn', limit: 200 }),
    ])
      .then(([errs, warns]) => {
        setErrors(errs)
        setWarnings(warns)
      })
      .catch(() => {})
  }, [])

  const byService = groupByService(errors)

  return (
    <div className="error-dashboard">
      <div className="error-summary-cards">
        <MetricsCard label="Errors" value={errors.length} color="var(--accent-red)" />
        <MetricsCard label="Warnings" value={warnings.length} color="var(--accent-yellow)" />
        <MetricsCard label="Services with Errors" value={Object.keys(byService).length} />
      </div>
      <div className="error-sections">
        <div className="error-section">
          <h3>Errors by Service</h3>
          <div className="error-service-list">
            {Object.entries(byService).map(([svc, entries]) => (
              <div key={svc} className="error-service-group">
                <div className="error-service-header">
                  <span className="error-service-name">{svc}</span>
                  <span className="error-service-count">{entries.length}</span>
                </div>
                <div className="error-service-entries">
                  {entries.slice(0, 10).map((e) => (
                    <div
                      key={e.id}
                      className={`error-entry ${levelClass(e.level)}`}
                      onClick={() => setSelected(e)}
                    >
                      <span className="error-entry-time">{formatTime(e.timestamp)}</span>
                      <span className="error-entry-msg">{e.message}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="error-section">
          <h3>Recent Errors</h3>
          <div className="error-entry-list">
            {errors.slice(0, 50).map((e) => (
              <div
                key={e.id}
                className={`error-entry ${levelClass(e.level)}`}
                onClick={() => setSelected(e)}
              >
                <span className="error-entry-time">{formatTime(e.timestamp)}</span>
                <span className="error-entry-svc">{e.service}</span>
                <span className="error-entry-msg">{e.message}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
      {selected && <LogDetail entry={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}

function groupByService(entries: LogEntry[]): Record<string, LogEntry[]> {
  const groups: Record<string, LogEntry[]> = {}
  for (const e of entries) {
    if (!groups[e.service]) groups[e.service] = []
    groups[e.service].push(e)
  }
  return groups
}
