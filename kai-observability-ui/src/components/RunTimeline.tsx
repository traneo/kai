import { useState, useEffect, useMemo, useCallback } from 'react'
import type { LogEntry } from '../types'
import { fetchLogs } from '../api'
import { formatDuration } from '../utils'
import { LogTable } from './LogTable'
import { useRefresh } from '../RefreshContext'
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts'

interface Props {
  runId: string
}

function formatShortTime(ts: number): string {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

export function RunTimeline({ runId }: Props) {
  const { refreshTick } = useRefresh()
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [expandedStep, setExpandedStep] = useState<string | null>(null)

  const load = useCallback(() => {
    fetchLogs({ run_id: runId, limit: 5000 })
      .then((d) => setEntries(d || []))
      .catch(() => setEntries([]))
  }, [runId])

  useEffect(() => { load() }, [load, refreshTick])

  const steps = useMemo(() => groupByStep(entries), [entries])
  const runStart = entries.length > 0 ? Math.min(...entries.map((e) => e.timestamp)) : 0
  const runEnd = entries.length > 0 ? Math.max(...entries.map((e) => e.timestamp)) : 0
  const runDuration = runEnd - runStart

  const stepDurationChart = useMemo(() => {
    return Object.entries(steps).map(([stepId, stepEntries]) => {
      const stepStart = Math.min(...stepEntries.map((e) => e.timestamp))
      const stepEnd = Math.max(...stepEntries.map((e) => e.timestamp))
      return {
        name: stepId,
        duration: ((stepEnd - stepStart) / 1000).toFixed(1),
        durationMs: stepEnd - stepStart,
        entries: stepEntries.length,
        start: stepStart,
        end: stepEnd,
      }
    }).sort((a, b) => a.start - b.start)
  }, [steps])

  const stepVolumeChart = useMemo(() => {
    return Object.entries(steps).map(([stepId, stepEntries]) => ({
      name: stepId,
      entries: stepEntries.length,
    })).sort((a, b) => b.entries - a.entries)
  }, [steps])

  return (
    <div className="run-timeline">
      <div className="run-timeline-header">
        <h2>Run: {runId}</h2>
        <span>Duration: {formatDuration(runStart, runEnd)}</span>
        <span className="timeline-total-entries">{entries.length} entries</span>
      </div>

      {stepDurationChart.length > 0 && (
        <div className="chart-grid">
          <div className="chart-card">
            <h3>Step Timeline (Gantt)</h3>
            <div className="timeline-gantt">
              <div className="timeline-gantt-header">
                <span className="timeline-gantt-label">Step</span>
                <div className="timeline-gantt-track">
                  {stepDurationChart.map((step) => {
                    const left = runDuration > 0 ? ((step.start - runStart) / runDuration) * 100 : 0
                    const width = runDuration > 0 ? ((step.end - step.start) / runDuration) * 100 : 0
                    return (
                      <div
                        key={step.name}
                        className="timeline-gantt-block"
                        style={{ left: `${left}%`, width: `${Math.max(width, 1)}%` }}
                        title={`${step.name}: ${step.duration}s, ${step.entries} entries`}
                      />
                    )
                  })}
                </div>
                <span className="timeline-gantt-dur">Duration</span>
              </div>
              {stepDurationChart.map((step) => {
                const left = runDuration > 0 ? ((step.start - runStart) / runDuration) * 100 : 0
                const width = runDuration > 0 ? ((step.end - step.start) / runDuration) * 100 : 0
                return (
                  <div
                    key={step.name}
                    className="timeline-gantt-row"
                    onClick={() => setExpandedStep(expandedStep === step.name ? null : step.name)}
                  >
                    <span className="timeline-gantt-label">{step.name}</span>
                    <div className="timeline-gantt-track">
                      <div
                        className="timeline-gantt-bar"
                        style={{ left: `${left}%`, width: `${Math.max(width, 1)}%` }}
                      />
                    </div>
                    <span className="timeline-gantt-dur">{step.duration}s</span>
                  </div>
                )
              })}
            </div>
          </div>

          <div className="chart-card">
            <h3>Step Duration</h3>
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={stepDurationChart} layout="vertical" margin={{ left: 100 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} label={{ value: 'seconds', position: 'insideBottom', style: { fill: '#6e7681', fontSize: 10 } }} />
                <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={140} />
                <Tooltip
                  contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                  formatter={(value: any) => [`${(Number(value) / 1000).toFixed(1)}s`, 'Duration']}
                />
                <Bar dataKey="durationMs" fill="#b926ff" radius={[0, 3, 3, 0]} name="Duration" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      <div className="chart-grid">
        <div className="chart-card">
          <h3>Entries per Step</h3>
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={stepVolumeChart} layout="vertical" margin={{ left: 100 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis type="number" stroke="#6e7681" tick={{ fontSize: 10 }} />
              <YAxis type="category" dataKey="name" stroke="#6e7681" tick={{ fontSize: 10 }} width={140} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
              />
              <Bar dataKey="entries" fill="#3fb950" radius={[0, 3, 3, 0]} name="Entries" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-card">
          <h3>Duration vs Entries</h3>
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={stepDurationChart} margin={{ left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis dataKey="name" stroke="#6e7681" tick={{ fontSize: 9 }} angle={-15} textAnchor="end" height={50} />
              <YAxis yAxisId="left" stroke="#6e7681" tick={{ fontSize: 10 }} label={{ value: 's', angle: -90, position: 'insideLeft', style: { fill: '#6e7681', fontSize: 10 } }} />
              <YAxis yAxisId="right" orientation="right" stroke="#6e7681" tick={{ fontSize: 10 }} label={{ value: '#', angle: 90, position: 'insideRight', style: { fill: '#6e7681', fontSize: 10 } }} />
              <Tooltip
                contentStyle={{ background: '#1e1e1e', border: '1px solid #333', borderRadius: 4, fontSize: 12 }}
                formatter={(value: any, name: any) => [
                  name === 'durationMs' ? `${(Number(value) / 1000).toFixed(1)}s` : value,
                  name === 'durationMs' ? 'Duration' : 'Entries'
                ]}
              />
              <Bar yAxisId="left" dataKey="durationMs" fill="#d29922" radius={[3, 3, 0, 0]} name="durationMs" />
              <Bar yAxisId="right" dataKey="entries" fill="#3fb950" radius={[3, 3, 0, 0]} name="entries" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="timeline-bars">
        {stepDurationChart.map((step) => (
          <div key={step.name} className="timeline-step">
            <div
              className="timeline-step-label"
              onClick={() => setExpandedStep(expandedStep === step.name ? null : step.name)}
            >
              <span className="timeline-step-name">{step.name}</span>
              <span className="timeline-step-stats">
                {step.entries} entries · {step.duration}s
              </span>
            </div>
            <div className="timeline-bar-track">
              <div
                className="timeline-bar"
                style={{
                  left: `${runDuration > 0 ? ((step.start - runStart) / runDuration) * 100 : 0}%`,
                  width: `${Math.max(runDuration > 0 ? ((step.end - step.start) / runDuration) * 100 : 0, 1)}%`
                }}
              />
            </div>
            {expandedStep === step.name && (
              <div className="timeline-step-detail">
                <LogTable filter={{ run_id: runId, step_id: step.name, limit: 100 }} />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

function groupByStep(entries: LogEntry[]): Record<string, LogEntry[]> {
  const groups: Record<string, LogEntry[]> = {}
  for (const e of entries) {
    const key = e.step_id || '(no step)'
    if (!groups[key]) groups[key] = []
    groups[key].push(e)
  }
  return groups
}
