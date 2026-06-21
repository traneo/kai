import { useState, useEffect, useMemo, useCallback } from 'react'
import type { RunSummary, LogEntry } from '../types'
import { fetchSummaries, fetchLogs } from '../api'
import { MetricsCard } from './MetricsCard'
import { useRefresh } from '../RefreshContext'
import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer,
  BarChart, Bar, CartesianGrid, Legend,
} from 'recharts'

function formatShortTime(ts: number): string {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

export function PerformanceCharts() {
  const { refreshTick } = useRefresh()
  const [summaries, setSummaries] = useState<RunSummary[]>([])
  const [runLogs, setRunLogs] = useState<Map<string, LogEntry[]>>(new Map())
  const [selectedRun, setSelectedRun] = useState<string | null>(null)
  const [allLogs, setAllLogs] = useState<LogEntry[]>([])

  const load = useCallback(() => {
    fetchSummaries()
      .then((data) => {
        const d = data || []
        setSummaries(d)
        if (d.length > 0 && !selectedRun) {
          setSelectedRun(d[0].run_id)
        }
      })
      .catch(() => setSummaries([]))
    fetchLogs({ limit: 500 }).then((d) => setAllLogs(d || [])).catch(() => setAllLogs([]))
  }, [])

  useEffect(() => { load() }, [load, refreshTick])

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

  const runDurations = useMemo(() => {
    return summaries
      .map((r) => ({
        id: r.run_id.slice(0, 24),
        duration: r.end_time > 0 ? (r.end_time - r.start_time) / 1000 : 0,
        entries: r.entry_count,
        steps: (r.steps || []).length,
      }))
      .sort((a, b) => a.duration - b.duration)
  }, [summaries])

  const stepDurationData = useMemo(() => {
    if (logs.length === 0) return []
    const byStep: Record<string, number[]> = {}
    for (const e of logs) {
      const step = e.step_id || '(no step)'
      if (!byStep[step]) byStep[step] = []
      byStep[step].push(e.timestamp)
    }
    return Object.entries(byStep)
      .map(([name, timestamps]) => {
        const min = Math.min(...timestamps)
        const max = Math.max(...timestamps)
        return { name, duration: (max - min) / 1000, entries: timestamps.length }
      })
      .sort((a, b) => b.duration - a.duration)
  }, [logs])

  const stepVolumeData = useMemo(() => {
    if (logs.length === 0) return []
    const byStep: Record<string, number> = {}
    for (const e of logs) {
      const step = e.step_id || '(no step)'
      byStep[step] = (byStep[step] || 0) + 1
    }
    return Object.entries(byStep)
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count)
  }, [logs])

  const activityTimeline = useMemo(() => {
    if (logs.length === 0) return []
    const ts = logs.map((e) => e.timestamp).sort((a, b) => a - b)
    const minT = ts[0]
    const maxT = ts[ts.length - 1]
    const bucketSize = Math.max(3000, Math.ceil((maxT - minT) / 25))
    const buckets: { time: number; count: number }[] = []
    let current = minT
    while (current <= maxT) {
      buckets.push({ time: current, count: 0 })
      current += bucketSize
    }
    for (const t of ts) {
      const idx = Math.min(buckets.length - 1, Math.floor((t - minT) / bucketSize))
      buckets[idx].count++
    }
    return buckets
  }, [logs])

  const totalEntries = summaries.reduce((s, r) => s + r.entry_count, 0)

  return (
    <div className="performance-charts">
      <div className="perf-summary-cards">
        <MetricsCard label="Total Runs" value={summaries.length} />
        <MetricsCard label="Total Entries" value={totalEntries} />
        <MetricsCard label="Runs with Data" value={summaries.filter((r) => r.entry_count > 0).length} />
      </div>

      <div className="chart-grid">
        <div className="chart-card">
          <h3>Run Durations</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={runDurations} margin={{ left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis dataKey="id" stroke="#6e7681" tick={{ fontSize: 9 }} angle={-20} textAnchor="end" height={50} />
              <YAxis stroke="#6e7681" tick={{ fontSize: 10 }} label={{ value: 'seconds', angle: -90, position: 'insideLeft', style: { fill: '#6e7681', fontSize: 10 } }} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                formatter={(value: any, name: any) => [name === 'duration' ? `${Number(value).toFixed(1)}s` : value, name]}
              />
              <Bar dataKey="duration" fill="#b926ff" radius={[3, 3, 0, 0]} name="Duration" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card">
          <h3>Entries per Run</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={runDurations} margin={{ left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis dataKey="id" stroke="#6e7681" tick={{ fontSize: 9 }} angle={-20} textAnchor="end" height={50} />
              <YAxis stroke="#6e7681" tick={{ fontSize: 10 }} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
              />
              <Bar dataKey="entries" fill="#3fb950" radius={[3, 3, 0, 0]} name="Entries" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="perf-section">
        <h3>Select Run</h3>
        <div className="perf-run-list">
          {summaries.map((rs) => (
            <div
              key={rs.run_id}
              className={`perf-run-item${selectedRun === rs.run_id ? ' selected' : ''}`}
              onClick={() => selectRun(rs.run_id)}
            >
              <span className="perf-run-id">{rs.run_id}</span>
              <span className="perf-run-duration">
                {rs.end_time > 0 ? `${((rs.end_time - rs.start_time) / 1000).toFixed(1)}s` : 'running'}
              </span>
              <span className="perf-run-entries">{rs.entry_count} entries · {(rs.steps || []).length} steps</span>
            </div>
          ))}
        </div>
      </div>

      {selectedRun && logs.length > 0 && (
        <>
          <div className="chart-grid">
            <div className="chart-card">
              <h3>Activity Timeline — {selectedRun.slice(0, 24)}…</h3>
              <ResponsiveContainer width="100%" height={200}>
                <AreaChart data={activityTimeline}>
                  <defs>
                    <linearGradient id="actGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#bc8cff" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#bc8cff" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="time" tickFormatter={formatShortTime} stroke="#6e7681" tick={{ fontSize: 10 }} />
                  <YAxis stroke="#6e7681" tick={{ fontSize: 10 }} />
                  <Tooltip
                    contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                  labelFormatter={formatShortTime as any}
                />
                <Area type="monotone" dataKey="count" stroke="#bc8cff" fill="url(#actGrad)" strokeWidth={2} name="Entries" />
                </AreaChart>
              </ResponsiveContainer>
            </div>

            <div className="chart-card">
              <h3>Steps by Duration</h3>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={stepDurationData} layout="vertical" margin={{ left: 100 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} label={{ value: 'seconds', position: 'insideBottom', style: { fill: '#6e7681', fontSize: 10 } }} />
                  <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={140} />
                  <Tooltip
                    contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                    formatter={(value: any) => [`${Number(value).toFixed(1)}s`, 'Duration']}
                  />
                  <Bar dataKey="duration" fill="#d29922" radius={[0, 3, 3, 0]} name="Duration" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="chart-grid">
            <div className="chart-card">
              <h3>Entries per Step</h3>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={stepVolumeData} layout="vertical" margin={{ left: 100 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} />
                  <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={140} />
                  <Tooltip
                    contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                  />
                  <Bar dataKey="count" fill="#3fb950" radius={[0, 3, 3, 0]} name="Entries" />
                </BarChart>
              </ResponsiveContainer>
            </div>

            <div className="chart-card">
              <h3>Duration · Entries Overlay</h3>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={stepDurationData} margin={{ left: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis dataKey="name" stroke="#6e7681" tick={{ fontSize: 9 }} angle={-15} textAnchor="end" height={50} width={120} />
                  <YAxis yAxisId="left" stroke="#6e7681" tick={{ fontSize: 10 }} />
                  <YAxis yAxisId="right" orientation="right" stroke="#6e7681" tick={{ fontSize: 10 }} />
                  <Tooltip
                    contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                  />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Bar yAxisId="left" dataKey="duration" fill="#d29922" radius={[3, 3, 0, 0]} name="Duration (s)" />
                  <Bar yAxisId="right" dataKey="entries" fill="#3fb950" radius={[3, 3, 0, 0]} name="Entries" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
