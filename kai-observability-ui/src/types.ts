export type LogLevel = 'info' | 'warn' | 'error' | 'debug'

export interface LogEntry {
  id: string
  service: string
  level: LogLevel
  message: string
  timestamp: number
  run_id?: string
  step_id?: string
  mission_id?: string
  agent_id?: string
  metadata?: Record<string, unknown>
  received_at: string
}

export interface QueryFilter {
  service?: string
  level?: LogLevel
  from?: string
  to?: string
  search?: string
  run_id?: string
  step_id?: string
  mission_id?: string
  agent_id?: string
  limit?: number
  offset?: number
}

export interface RunSummary {
  run_id: string
  entry_count: number
  start_time: number
  end_time: number
  services: string[]
  steps: string[]
}

export type Page =
  | 'dashboard'
  | 'live'
  | 'logs'
  | 'log-detail'
  | 'runs'
  | 'run-detail'
  | 'errors'
  | 'analytics'
