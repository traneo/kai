import { useState, useEffect, useCallback } from 'react'
import type { RunSummary } from '../types'
import { fetchSummaries } from '../api'
import { MetricsCard } from './MetricsCard'
import { formatDuration, formatTime } from '../utils'
import { useRefresh } from '../RefreshContext'

interface Props {
  onSelectRun: (runId: string) => void
}

export function RunExplorer({ onSelectRun }: Props) {
  const { refreshTick } = useRefresh()
  const [summaries, setSummaries] = useState<RunSummary[]>([])

  const load = useCallback(() => {
    fetchSummaries().then((d) => setSummaries(d || [])).catch(() => setSummaries([]))
  }, [])

  useEffect(() => { load() }, [load, refreshTick])

  if (summaries.length === 0) {
    return <div className="run-empty">No pipeline runs found</div>
  }

  return (
    <div className="run-explorer">
      {summaries.map((rs) => (
        <div
          key={rs.run_id}
          className="run-card"
          onClick={() => onSelectRun(rs.run_id)}
        >
          <div className="run-card-header">
            <span className="run-card-id">{rs.run_id}</span>
            <span className="run-card-duration">{formatDuration(rs.start_time, rs.end_time)}</span>
          </div>
          <div className="run-card-stats">
            <MetricsCard label="Entries" value={rs.entry_count} />
            <MetricsCard label="Services" value={(rs.services || []).length} />
            <MetricsCard label="Steps" value={(rs.steps || []).length} />
            <MetricsCard label="Started" value={formatTime(rs.start_time)} />
          </div>
        </div>
      ))}
    </div>
  )
}
