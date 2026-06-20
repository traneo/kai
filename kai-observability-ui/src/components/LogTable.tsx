import { useState, useEffect, useCallback } from 'react'
import type { LogEntry, QueryFilter } from '../types'
import { fetchLogs } from '../api'
import { LogRow } from './LogRow'
import { LogDetail } from './LogDetail'

interface Props {
  filter: QueryFilter
}

export function LogTable({ filter }: Props) {
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [offset, setOffset] = useState(0)
  const [selected, setSelected] = useState<LogEntry | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchLogs({ ...filter, offset })
      setEntries(data)
    } catch {
      setEntries([])
    } finally {
      setLoading(false)
    }
  }, [filter, offset])

  useEffect(() => {
    setOffset(0)
  }, [filter])

  useEffect(() => {
    load()
  }, [load])

  return (
    <div className="log-table-container">
      {loading && <div className="spinner" />}
      <table className="log-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Level</th>
            <th>Service</th>
            <th>Message</th>
            <th>Run ID</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <LogRow key={e.id} entry={e} onClick={setSelected} />
          ))}
          {entries.length === 0 && !loading && (
            <tr><td colSpan={5} className="log-empty">No log entries found</td></tr>
          )}
        </tbody>
      </table>
      <div className="pagination">
        <button disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - 100))}>
          Previous
        </button>
        <span className="pagination-info">Offset {offset}</span>
        <button disabled={entries.length < 100} onClick={() => setOffset(offset + 100)}>
          Next
        </button>
      </div>
      {selected && <LogDetail entry={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}
