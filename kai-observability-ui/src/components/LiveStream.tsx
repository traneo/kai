import { useState, useEffect, useRef, useCallback } from 'react'
import type { LogEntry, QueryFilter } from '../types'
import { subscribeLogStream } from '../api'
import { formatTime, levelClass } from '../utils'
import { FilterBar } from './FilterBar'

export function LiveStream() {
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [paused, setPaused] = useState(false)
  const [filter, setFilter] = useState<QueryFilter>({})
  const bottomRef = useRef<HTMLDivElement>(null)
  const maxEntries = 500

  const onEntry = useCallback((entry: LogEntry) => {
    setEntries((prev) => {
      const next = [...prev, entry]
      return next.length > maxEntries ? next.slice(next.length - maxEntries) : next
    })
  }, [])

  useEffect(() => {
    const unsub = subscribeLogStream(onEntry, filter)
    return unsub
  }, [onEntry, filter])

  useEffect(() => {
    if (!paused) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [entries, paused])

  function clear() {
    setEntries([])
  }

  return (
    <div className="live-stream">
      <div className="live-stream-header">
        <FilterBar filter={filter} onChange={setFilter} />
        <div className="live-stream-controls">
          <button className="btn" onClick={() => setPaused(!paused)}>
            {paused ? '▶ Resume' : '❚❚ Pause'}
          </button>
          <span className="live-count">{entries.length} entries</span>
          <button className="btn" onClick={clear}>Clear</button>
        </div>
      </div>
      <div className="live-stream-entries">
        {entries.map((e) => (
          <div key={e.id} className={`live-entry ${levelClass(e.level)}`}>
            <span className="live-time">{formatTime(e.timestamp)}</span>
            <span className={`level-badge ${levelClass(e.level)}`}>{e.level}</span>
            <span className="live-service">{e.service}</span>
            <span className="live-msg">{e.message}</span>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
