import type { LogEntry, QueryFilter, RunSummary } from './types'

const API = '/api/v1'

export async function fetchLogs(filter: QueryFilter): Promise<LogEntry[]> {
  const params = new URLSearchParams()
  if (filter.service) params.set('service', filter.service)
  if (filter.level) params.set('level', filter.level)
  if (filter.from) params.set('from', filter.from)
  if (filter.to) params.set('to', filter.to)
  if (filter.search) params.set('search', filter.search)
  if (filter.run_id) params.set('run_id', filter.run_id)
  if (filter.step_id) params.set('step_id', filter.step_id)
  if (filter.mission_id) params.set('mission_id', filter.mission_id)
  if (filter.agent_id) params.set('agent_id', filter.agent_id)
  if (filter.limit) params.set('limit', String(filter.limit))
  if (filter.offset) params.set('offset', String(filter.offset))

  const res = await fetch(`${API}/logs?${params}`)
  if (!res.ok) throw new Error(`fetchLogs: ${res.status}`)
  return res.json()
}

export async function fetchLog(id: string): Promise<LogEntry | null> {
  const res = await fetch(`${API}/logs/${id}`)
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`fetchLog: ${res.status}`)
  return res.json()
}

export async function fetchSummaries(): Promise<RunSummary[]> {
  const res = await fetch(`${API}/logs/summaries`)
  if (!res.ok) throw new Error(`fetchSummaries: ${res.status}`)
  return res.json()
}

export function subscribeLogStream(
  onEntry: (entry: LogEntry) => void,
  filter?: QueryFilter
): () => void {
  const params = new URLSearchParams()
  if (filter?.service) params.set('service', filter.service)
  if (filter?.level) params.set('level', filter.level)

  const url = params.toString()
    ? `${API}/logs/stream?${params}`
    : `${API}/logs/stream`

  const es = new EventSource(url)
  es.onmessage = (e) => {
    try {
      const entry: LogEntry = JSON.parse(e.data)
      onEntry(entry)
    } catch { /* skip malformed */ }
  }
  es.onerror = () => {} // auto-reconnects
  return () => es.close()
}
