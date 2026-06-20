import { useState, useEffect } from 'react'
import type { RunSummary, LogEntry } from '../types'
import { fetchSummaries, fetchLogs } from '../api'
import { formatDuration, formatTime, levelClass } from '../utils'
import { MetricsCard } from './MetricsCard'
import { LogDetail } from './LogDetail'

export function Dashboard() {
  const [summaries, setSummaries] = useState<RunSummary[]>([])
  const [recentErrors, setRecentErrors] = useState<LogEntry[]>([])
  const [selected, setSelected] = useState<LogEntry | null>(null)

  useEffect(() => {
    fetchSummaries().then(setSummaries).catch(() => setSummaries([]))
    fetchLogs({ level: 'error', limit: 20 })
      .then(setRecentErrors)
      .catch(() => setRecentErrors([]))
  }, [])

  const activeRuns = summaries.filter((rs) => rs.end_time === 0)
  const totalEntries = summaries.reduce((s, r) => s + r.entry_count, 0)
  const avgDuration = summaries.length > 0
    ? summaries.reduce((s, r) => s + (r.end_time - r.start_time), 0) / summaries.length
    : 0

  const volumeByService = computeVolumeByService(summaries)

  return (
    <div className="dashboard">
      <div className="dashboard-cards">
        <MetricsCard label="Total Runs" value={summaries.length} />
        <MetricsCard label="Active Runs" value={activeRuns.length} color="var(--accent-green)" />
        <MetricsCard label="Total Log Entries" value={totalEntries} />
        <MetricsCard label="Avg Run Duration" value={formatDuration(0, avgDuration)} />
      </div>
      <div className="dashboard-sections">
        <div className="dashboard-section">
          <h3>Recent Errors</h3>
          <div className="dashboard-error-list">
            {recentErrors.map((e) => (
              <div
                key={e.id}
                className={`dashboard-error ${levelClass(e.level)}`}
                onClick={() => setSelected(e)}
              >
                <span className="dashboard-error-time">{formatTime(e.timestamp)}</span>
                <span className="dashboard-error-svc">{e.service}</span>
                <span className="dashboard-error-msg">{e.message}</span>
              </div>
            ))}
            {recentErrors.length === 0 && (
              <div className="dashboard-empty">No recent errors</div>
            )}
          </div>
        </div>
        <div className="dashboard-section">
          <h3>Log Volume by Service</h3>
          <div className="dashboard-volume">
            {Object.entries(volumeByService).map(([svc, count]) => (
              <div key={svc} className="dashboard-volume-row">
                <span className="dashboard-volume-svc">{svc}</span>
                <div className="dashboard-volume-track">
                  <div
                    className="dashboard-volume-bar"
                    style={{ width: `${Math.min(100, (count / Math.max(...Object.values(volumeByService))) * 100)}%` }}
                  />
                </div>
                <span className="dashboard-volume-count">{count}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="dashboard-section">
          <h3>Recent Runs</h3>
          <div className="dashboard-run-list">
            {summaries.slice(0, 10).map((rs) => (
              <div key={rs.run_id} className="dashboard-run">
                <span className="dashboard-run-id">{rs.run_id}</span>
                <span className="dashboard-run-entries">{rs.entry_count} entries</span>
                <span className="dashboard-run-duration">{formatDuration(rs.start_time, rs.end_time)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
      {selected && <LogDetail entry={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}

function computeVolumeByService(summaries: RunSummary[]): Record<string, number> {
  const counts: Record<string, number> = {}
  for (const rs of summaries) {
    for (const svc of rs.services) {
      counts[svc] = (counts[svc] || 0) + 1
    }
  }
  return counts
}
