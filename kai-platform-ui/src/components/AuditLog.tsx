import { useEffect, useState } from 'react'
import { fetchAuditLog, subscribeEvents } from '../api'
import type { AuditEvent } from '../types'

export function AuditLog() {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [filter, setFilter] = useState('')
  const [runFilter, setRunFilter] = useState('')
  const [loading, setLoading] = useState(true)

  async function load() {
    setLoading(true)
    try {
      const data = await fetchAuditLog(100, runFilter || undefined)
      setEvents(data)
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }

  useEffect(() => {
    load()
    const unsub = subscribeEvents(() => load())
    return unsub
  }, [])

  useEffect(() => {
    load()
  }, [runFilter])

  const filtered = filter
    ? events.filter(e => e.type.includes(filter) || e.run_id.includes(filter) || e.step_id.includes(filter) || (e.message || '').includes(filter))
    : events

  function eventColor(type: string): string {
    if (type.includes('failed') || type.includes('rejected')) return 'event-failed'
    if (type.includes('completed') || type.includes('passed') || type.includes('approved') || type.includes('granted')) return 'event-passed'
    if (type.includes('started') || type.includes('assigned')) return 'event-started'
    if (type.includes('blocked') || type.includes('cancelled')) return 'event-warn'
    return 'event-info'
  }

  return (
    <div className="audit-log">
      <div className="audit-toolbar">
        <input
          className="audit-search"
          type="text"
          placeholder="Filter events..."
          value={filter}
          onChange={e => setFilter(e.target.value)}
        />
        <input
          className="audit-search"
          type="text"
          placeholder="Run ID..."
          value={runFilter}
          onChange={e => setRunFilter(e.target.value)}
        />
        <button className="btn-small" onClick={load}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 3 }}>
            <polyline points="23 4 23 10 17 10" /><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
          </svg>
          Refresh
        </button>
      </div>

      {loading && <p className="muted">Loading...</p>}
      {!loading && filtered.length === 0 && <p className="muted">No events found</p>}

      <div className="audit-list">
        {filtered.map(e => (
          <div key={e.id} className={`audit-event ${eventColor(e.type)}`}>
            <div className="audit-event-header">
              <span className="audit-event-type">{e.type}</span>
              <span className="audit-event-time">{new Date(e.time).toLocaleString()}</span>
            </div>
            <div className="audit-event-detail">
              {e.run_id && <span className="audit-tag">run: {e.run_id}</span>}
              {e.step_id && <span className="audit-tag">step: {e.step_id}</span>}
              {e.agent_id && <span className="audit-tag">agent: {e.agent_id}</span>}
            </div>
            {e.message && <div className="audit-event-message">{e.message}</div>}
          </div>
        ))}
      </div>
    </div>
  )
}
