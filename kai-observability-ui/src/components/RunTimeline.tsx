import { useState, useEffect } from 'react'
import type { LogEntry } from '../types'
import { fetchLogs } from '../api'
import { formatDuration, formatTime } from '../utils'
import { LogTable } from './LogTable'

interface Props {
  runId: string
}

export function RunTimeline({ runId }: Props) {
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [expandedStep, setExpandedStep] = useState<string | null>(null)

  useEffect(() => {
    fetchLogs({ run_id: runId, limit: 5000 })
      .then(setEntries)
      .catch(() => setEntries([]))
  }, [runId])

  const steps = groupByStep(entries)
  const runStart = entries.length > 0 ? Math.min(...entries.map((e) => e.timestamp)) : 0
  const runEnd = entries.length > 0 ? Math.max(...entries.map((e) => e.timestamp)) : 0
  const runDuration = runEnd - runStart

  return (
    <div className="run-timeline">
      <div className="run-timeline-header">
        <h2>Run: {runId}</h2>
        <span>Duration: {formatDuration(runStart, runEnd)}</span>
      </div>
      <div className="timeline-bars">
        {Object.entries(steps).map(([stepId, stepEntries]) => {
          const stepStart = Math.min(...stepEntries.map((e) => e.timestamp))
          const stepEnd = Math.max(...stepEntries.map((e) => e.timestamp))
          const left = runDuration > 0 ? ((stepStart - runStart) / runDuration) * 100 : 0
          const width = runDuration > 0 ? ((stepEnd - stepStart) / runDuration) * 100 : 0

          return (
            <div key={stepId} className="timeline-step">
              <div className="timeline-step-label" onClick={() => setExpandedStep(expandedStep === stepId ? null : stepId)}>
                <span className="timeline-step-name">{stepId}</span>
                <span className="timeline-step-stats">
                  {stepEntries.length} entries · {formatDuration(stepStart, stepEnd)}
                </span>
              </div>
              <div className="timeline-bar-track">
                <div
                  className="timeline-bar"
                  style={{ left: `${left}%`, width: `${Math.max(width, 1)}%` }}
                />
              </div>
              {expandedStep === stepId && (
                <div className="timeline-step-detail">
                  <LogTable filter={{ run_id: runId, step_id: stepId, limit: 100 }} />
                </div>
              )}
            </div>
          )
        })}
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
