import { useState, useEffect, useMemo, useCallback } from 'react'
import type { RunSummary, LogEntry } from '../types'
import { fetchSummaries, fetchLogs } from '../api'
import { formatDuration, levelClass } from '../utils'
import { MetricsCard } from './MetricsCard'
import { LogDetail } from './LogDetail'
import { useRefresh } from '../RefreshContext'
import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
  BarChart, Bar, CartesianGrid,
} from 'recharts'

const LEVEL_COLORS: Record<string, string> = {
  error: '#f85149',
  warn: '#d29922',
  info: '#b926ff',
  debug: '#6e7681',
}

function formatShortTime(ts: number): string {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

export function Dashboard() {
  const { refreshTick } = useRefresh()
  const [summaries, setSummaries] = useState<RunSummary[]>([])
  const [allLogs, setAllLogs] = useState<LogEntry[]>([])
  const [recentErrors, setRecentErrors] = useState<LogEntry[]>([])
  const [selected, setSelected] = useState<LogEntry | null>(null)

  const load = useCallback(() => {
    fetchSummaries().then((d) => setSummaries(d || [])).catch(() => setSummaries([]))
    Promise.all([
      fetchLogs({ limit: 500 }),
      fetchLogs({ level: 'error', limit: 20 }),
    ])
      .then(([logs, errs]) => {
        setAllLogs(logs || [])
        setRecentErrors(errs || [])
      })
      .catch(() => {})
  }, [])

  useEffect(() => { load() }, [load, refreshTick])

  const activeRuns = summaries.filter((rs) => rs.end_time === 0)
  const totalEntries = summaries.reduce((s, r) => s + r.entry_count, 0)
  const avgDuration = summaries.length > 0
    ? summaries.reduce((s, r) => s + (r.end_time - r.start_time), 0) / summaries.length
    : 0

  const volumeOverTime = useMemo(() => {
    if (!allLogs || allLogs.length === 0) return []
    const ts = allLogs.map((e) => e.timestamp).sort((a, b) => a - b)
    const minT = ts[0]
    const maxT = ts[ts.length - 1]
    const bucketSize = Math.max(5000, Math.ceil((maxT - minT) / 20))
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
  }, [allLogs])

  const levelDistribution = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of allLogs) {
      counts[e.level] = (counts[e.level] || 0) + 1
    }
    return Object.entries(counts).map(([name, value]) => ({ name, value }))
  }, [allLogs])

  const serviceBreakdown = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of allLogs) {
      counts[e.service] = (counts[e.service] || 0) + 1
    }
    return Object.entries(counts)
      .map(([name, value]) => ({ name, value }))
      .sort((a, b) => b.value - a.value)
  }, [allLogs])

  const stepBreakdown = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of allLogs) {
      const step = e.step_id || '(no step)'
      counts[step] = (counts[step] || 0) + 1
    }
    return Object.entries(counts)
      .map(([name, value]) => ({ name, value }))
      .sort((a, b) => b.value - a.value)
  }, [allLogs])

  const runOverview = useMemo(() => {
    if (summaries.length === 0) return []
    const allStart = Math.min(...summaries.map((r) => r.start_time))
    const allEnd = Math.max(...summaries.map((r) => r.end_time || Date.now()))
    const totalDuration = allEnd - allStart || 1
    return summaries.map((r) => ({
      id: r.run_id,
      startOffset: ((r.start_time - allStart) / totalDuration) * 100,
      width: Math.max(((Math.max(r.end_time || Date.now(), r.start_time) - r.start_time) / totalDuration) * 100, 1),
      entries: r.entry_count,
      steps: (r.steps || []).length,
      services: (r.services || []).join(', '),
    }))
  }, [summaries])

  return (
    <div className="dashboard">
      <div className="dashboard-cards">
        <MetricsCard label="Total Runs" value={summaries.length} />
        <MetricsCard label="Active Runs" value={activeRuns.length} color="var(--accent-green)" />
        <MetricsCard label="Total Log Entries" value={totalEntries} />
        <MetricsCard label="Avg Run Duration" value={formatDuration(0, avgDuration)} />
      </div>

      <div className="chart-grid">
        <div className="chart-card">
          <h3>Log Volume Over Time</h3>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={volumeOverTime}>
              <defs>
                <linearGradient id="volGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#b926ff" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#b926ff" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="time" tickFormatter={formatShortTime} stroke="#6e7681" tick={{ fontSize: 10 }} />
              <YAxis stroke="#6e7681" tick={{ fontSize: 10 }} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                labelFormatter={formatShortTime as any}
                formatter={(value: any) => [value, 'Entries']}
              />
              <Area type="monotone" dataKey="count" stroke="#b926ff" fill="url(#volGrad)" strokeWidth={2} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card">
          <h3>Level Distribution</h3>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie
                data={levelDistribution}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                outerRadius={70}
                innerRadius={35}
                stroke="none"
              >
                {levelDistribution.map((entry) => (
                  <Cell key={entry.name} fill={LEVEL_COLORS[entry.name] || '#6e7681'} />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
              />
            </PieChart>
          </ResponsiveContainer>
          <div className="chart-legend">
            {levelDistribution.map((entry) => (
              <span key={entry.name} className="legend-item">
                <span className="legend-dot" style={{ background: LEVEL_COLORS[entry.name] || '#6e7681' }} />
                {entry.name}: {entry.value}
              </span>
            ))}
          </div>
        </div>

        <div className="chart-card">
          <h3>Entries by Service</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={serviceBreakdown} layout="vertical" margin={{ left: 60 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} />
              <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={80} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
              />
              <Bar dataKey="value" fill="#bc8cff" radius={[0, 3, 3, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card">
          <h3>Entries by Step</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={stepBreakdown} layout="vertical" margin={{ left: 100 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} />
              <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={140} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
              />
              <Bar dataKey="value" fill="#3fb950" radius={[0, 3, 3, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="dashboard-sections">
        <div className="dashboard-section">
          <h3>Run Overview</h3>
          <div className="run-overview-chart">
            {runOverview.map((r) => (
              <div key={r.id} className="run-overview-row" title={`${r.id} — ${r.entries} entries, ${r.steps} steps`}>
                <span className="run-overview-label">{r.id.slice(0, 20)}…</span>
                <div className="run-overview-track">
                  <div
                    className="run-overview-bar"
                    style={{ left: `${r.startOffset}%`, width: `${r.width}%` }}
                  />
                </div>
                <span className="run-overview-stats">{r.entries}</span>
              </div>
            ))}
            {runOverview.length === 0 && (
              <div className="dashboard-empty">No runs</div>
            )}
          </div>
        </div>

        <div className="dashboard-section">
          <h3>Recent Errors</h3>
          <div className="dashboard-error-list">
            {recentErrors.map((e) => (
              <div
                key={e.id}
                className={`dashboard-error ${levelClass(e.level)}`}
                onClick={() => setSelected(e)}
              >
                <span className="dashboard-error-time">{new Date(e.timestamp).toLocaleTimeString('en-US', { hour12: false })}</span>
                <span className="dashboard-error-svc">{e.service}</span>
                <span className="dashboard-error-msg">{e.message}</span>
              </div>
            ))}
            {recentErrors.length === 0 && (
              <div className="dashboard-empty">No recent errors</div>
            )}
          </div>
        </div>
      </div>

      {selected && <LogDetail entry={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}
