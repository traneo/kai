import { useState, useEffect } from 'react'
import type { RunSummary, LogEntry } from '../types'
import { fetchSummaries, fetchLogs } from '../api'
import { formatDuration } from '../utils'
import { MetricsCard } from './MetricsCard'

export function PerformanceCharts() {
  const [summaries, setSummaries] = useState<RunSummary[]>([])
  const [runLogs, setRunLogs] = useState<Map<string, LogEntry[]>>(new Map())
  const [selectedRun, setSelectedRun] = useState<string | null>(null)

  useEffect(() => {
    fetchSummaries().then(setSummaries).catch(() => setSummaries([]))
  }, [])

  async function selectRun(runId: string) {
    if (runLogs.has(runId)) {
      setSelectedRun(runId)
      return
    }
    const entries = await fetchLogs({ run_id: runId, limit: 5000 }).catch(() => [])
    setRunLogs((prev) => {
      const next = new Map(prev)
      next.set(runId, entries)
      return next
    })
    setSelectedRun(runId)
  }

  const logs = selectedRun ? runLogs.get(selectedRun) || [] : []

  const agentCounts = countByAgent(logs)
  const durationByAgent = computeDurationByAgent(logs)

  return (
    <div className="performance-charts">
      <div className="perf-summary-cards">
        <MetricsCard label="Total Runs" value={summaries.length} />
        <MetricsCard
          label="Total Entries"
          value={summaries.reduce((s, r) => s + r.entry_count, 0)}
        />
      </div>
      <div className="perf-section">
        <h3>Runs</h3>
        <div className="perf-run-list">
          {summaries.map((rs) => (
            <div
              key={rs.run_id}
              className={`perf-run-item${selectedRun === rs.run_id ? ' selected' : ''}`}
              onClick={() => selectRun(rs.run_id)}
            >
              <span className="perf-run-id">{rs.run_id}</span>
              <span className="perf-run-duration">{formatDuration(rs.start_time, rs.end_time)}</span>
              <span className="perf-run-entries">{rs.entry_count} entries</span>
            </div>
          ))}
        </div>
      </div>
      {selectedRun && (
        <div className="perf-section">
          <h3>Run: {selectedRun}</h3>
          <div className="perf-charts-grid">
            <div className="perf-chart-card">
              <h4>Entries by Agent</h4>
              <div className="perf-bar-chart">
                {Object.entries(agentCounts).map(([agent, count]) => (
                  <div key={agent} className="perf-bar-row">
                    <span className="perf-bar-label">{agent || '(none)'}</span>
                    <div className="perf-bar-track">
                      <div
                        className="perf-bar perf-bar-blue"
                        style={{ width: `${Math.min(100, (count / logs.length) * 100)}%` }}
                      />
                    </div>
                    <span className="perf-bar-value">{count}</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="perf-chart-card">
              <h4>Duration by Agent</h4>
              <div className="perf-bar-chart">
                {Object.entries(durationByAgent).map(([agent, dur]) => (
                  <div key={agent} className="perf-bar-row">
                    <span className="perf-bar-label">{agent || '(none)'}</span>
                    <div className="perf-bar-track">
                      <div
                        className="perf-bar perf-bar-green"
                        style={{ width: `${Math.min(100, (dur / 60000) * 100)}%` }}
                      />
                    </div>
                    <span className="perf-bar-value">{formatDuration(0, dur)}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function countByAgent(entries: LogEntry[]): Record<string, number> {
  const counts: Record<string, number> = {}
  for (const e of entries) {
    const agent = e.agent_id || '(none)'
    counts[agent] = (counts[agent] || 0) + 1
  }
  return counts
}

function computeDurationByAgent(entries: LogEntry[]): Record<string, number> {
  const byAgent: Record<string, number[]> = {}
  for (const e of entries) {
    if (!e.agent_id) continue
    if (!byAgent[e.agent_id]) byAgent[e.agent_id] = []
    byAgent[e.agent_id].push(e.timestamp)
  }
  const result: Record<string, number> = {}
  for (const [agent, timestamps] of Object.entries(byAgent)) {
    const min = Math.min(...timestamps)
    const max = Math.max(...timestamps)
    result[agent] = max - min
  }
  return result
}
