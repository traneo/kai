import { useState, useEffect, useMemo, useCallback } from 'react'
import type { LogEntry } from '../types'
import { fetchLogs } from '../api'
import { levelClass } from '../utils'
import { MetricsCard } from './MetricsCard'
import { LogDetail } from './LogDetail'
import { useRefresh } from '../RefreshContext'
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
  BarChart, Bar, CartesianGrid,
} from 'recharts'

const LEVEL_COLORS: Record<string, string> = {
  error: '#f85149',
  warn: '#d29922',
}

function formatShortTime(ts: number): string {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

export function ErrorDashboard() {
  const { refreshTick } = useRefresh()
  const [errors, setErrors] = useState<LogEntry[]>([])
  const [warnings, setWarnings] = useState<LogEntry[]>([])
  const [selected, setSelected] = useState<LogEntry | null>(null)

  const load = useCallback(() => {
    Promise.all([
      fetchLogs({ level: 'error', limit: 200 }),
      fetchLogs({ level: 'warn', limit: 200 }),
    ])
      .then(([errs, warns]) => {
        setErrors(errs || [])
        setWarnings(warns || [])
      })
      .catch(() => {})
  }, [])

  useEffect(() => { load() }, [load, refreshTick])

  const byService = useMemo(() => {
    const groups: Record<string, LogEntry[]> = {}
    for (const e of errors) {
      if (!groups[e.service]) groups[e.service] = []
      groups[e.service].push(e)
    }
    return groups
  }, [errors])

  const allIssues = useMemo(() => [...errors, ...warnings], [errors, warnings])

  const errorTimeline = useMemo(() => {
    if (allIssues.length === 0) return []
    const ts = allIssues.map((e) => e.timestamp).sort((a, b) => a - b)
    const minT = ts[0]
    const maxT = ts[ts.length - 1]
    if (minT === maxT) return [{ time: minT, errors: errors.length, warnings: warnings.length }]
    const bucketSize = Math.max(5000, Math.ceil((maxT - minT) / 15))
    const errorBuckets: Record<number, { time: number; errors: number; warnings: number }> = {}
    let current = minT
    while (current <= maxT) {
      errorBuckets[current] = { time: current, errors: 0, warnings: 0 }
      current += bucketSize
    }
    for (const e of allIssues) {
      const key = Math.floor((e.timestamp - minT) / bucketSize) * bucketSize + minT
      if (errorBuckets[key]) {
        if (e.level === 'error') errorBuckets[key].errors++
        else errorBuckets[key].warnings++
      }
    }
    return Object.values(errorBuckets).sort((a, b) => a.time - b.time)
  }, [allIssues, errors, warnings])

  const serviceErrorData = useMemo(() => {
    return Object.entries(byService)
      .map(([name, entries]) => ({ name, count: entries.length }))
      .sort((a, b) => b.count - a.count)
  }, [byService])

  const levelDist = useMemo(() => {
    return [
      { name: 'error', value: errors.length },
      { name: 'warn', value: warnings.length },
    ].filter((d) => d.value > 0)
  }, [errors, warnings])

  return (
    <div className="error-dashboard">
      <div className="error-summary-cards">
        <MetricsCard label="Errors" value={errors.length} color="var(--accent-red)" />
        <MetricsCard label="Warnings" value={warnings.length} color="var(--accent-yellow)" />
        <MetricsCard label="Services with Errors" value={Object.keys(byService).length} />
        <MetricsCard label="Total Issues" value={errors.length + warnings.length} color="var(--accent-purple)" />
      </div>

      {allIssues.length > 0 && (
        <div className="chart-grid">
          <div className="chart-card">
            <h3>Issues Over Time</h3>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={errorTimeline}>
                <XAxis dataKey="time" tickFormatter={formatShortTime} stroke="#6e7681" tick={{ fontSize: 10 }} />
                <YAxis stroke="#6e7681" tick={{ fontSize: 10 }} />
                <Tooltip
                  contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                  labelFormatter={formatShortTime as any}
                />
                <Line type="monotone" dataKey="errors" stroke="#f85149" strokeWidth={2} dot={false} name="Errors" />
                <Line type="monotone" dataKey="warnings" stroke="#d29922" strokeWidth={2} dot={false} name="Warnings" />
              </LineChart>
            </ResponsiveContainer>
          </div>

          <div className="chart-card">
            <h3>Errors by Service</h3>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={serviceErrorData} layout="vertical" margin={{ left: 80 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} />
                <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={100} />
                <Tooltip
                  contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                />
                <Bar dataKey="count" fill="#f85149" radius={[0, 3, 3, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          <div className="chart-card">
            <h3>Error vs Warning</h3>
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie
                  data={levelDist}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={70}
                  innerRadius={35}
                  stroke="none"
                >
                  {levelDist.map((entry) => (
                    <Cell key={entry.name} fill={LEVEL_COLORS[entry.name] || '#6e7681'} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                />
              </PieChart>
            </ResponsiveContainer>
            <div className="chart-legend">
              {levelDist.map((entry) => (
                <span key={entry.name} className="legend-item">
                  <span className="legend-dot" style={{ background: LEVEL_COLORS[entry.name] || '#6e7681' }} />
                  {entry.name}: {entry.value}
                </span>
              ))}
            </div>
          </div>
        </div>
      )}

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
                      <span className="error-entry-time">{new Date(e.timestamp).toLocaleTimeString('en-US', { hour12: false })}</span>
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
                <span className="error-entry-time">{new Date(e.timestamp).toLocaleTimeString('en-US', { hour12: false })}</span>
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
