export interface Agent {
  id: string
  addr: string
  state: string
  mission_id: string
  mission_status: string
  missions_completed: number
  healthy: boolean
  uptime_ms: number
  connected_at: string
  last_heartbeat: string
}

export interface PipelineRun {
  id: string
  project: string
  status: string
  steps: number
  passed: number
  failed: number
  has_blocked: boolean
  has_queued: boolean
  created_at: string
}

export interface GateResult {
  gate: string
  status: string
  message: string
  duration: string
}

export interface StepPolicy {
  allowed_dirs?: string[]
  allowed_tools?: string[]
  allowed_commands?: string[]
  agent?: string
  max_retries?: number
  timeout_seconds?: number
}

export interface StepDetail {
  id: string
  prompt: string
  status: string
  depends_on: string[]
  validation: string[]
  approval: string
  retries: number
  max_retries: number
  next_retry_at?: string
  assigned_to: string
  error: string
  started_at: string | null
  policy?: StepPolicy
  gate_results?: GateResult[]
  diff?: string
}

export interface PipelineDetail {
  id: string
  project: string
  status: string
  steps: StepDetail[]
  created_at: string
  updated_at: string
  error?: string
  output_url?: string
  output_sha?: string
}

export interface AuditEvent {
  id: number
  time: string
  type: string
  run_id: string
  step_id: string
  agent_id: string
  message: string
  payload: unknown
}

export interface ServerEvent {
  type: string
  [key: string]: unknown
}

export interface Stats {
  agents: { total: number; idle: number; busy: number }
  queue_depth: number
  pipelines: number
  steps: number
  tokens: { total: number; prompt: number; completion: number }
  duration_ms: number
  runs?: { run_id: string; project: string; total_tokens: number }[]
}

export interface QueueEntry {
  run_id: string
  step_id: string
  prompt: string
  agent_id: string
  position: number
}

export interface ConversationEntry {
  mission_id: string
  run_id: string
  step_id: string
  sequence: number
  timestamp: string
  source: string
  message: string
}

export interface SecretMeta {
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface ServerConfig {
  http_port: string
  grpc_port: string
  version: string
}

export interface AuthConfig {
  mode: string
  pre_shared_token?: string
  tls_cert_file?: string
  tls_key_file?: string
  tls_ca_cert_file?: string
}

export interface PoolConfig {
  heartbeat_timeout: string
  health_interval: string
}

export interface BackendStatus {
  secrets: string
  audit: string
  archive: string
  plugin_dir: string
  database_url?: string
}

export interface PlatformConfig {
  server: ServerConfig
  auth: AuthConfig
  pool: PoolConfig
  backends: BackendStatus
}

export interface SystemConfig {
  platform: PlatformConfig
  blob?: { runner: string; data?: unknown }
}

export interface ConfigVersion {
  id: string
  version: number
  status: string
  config: SystemConfig
  message: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface ReloadResult {
  status: string
  applied_at: string
  hot_reloaded: string[]
  requires_restart: string[]
  errors?: string[]
}

export type Page = 'landing' | 'dashboard' | 'detail' | 'new' | 'audit' | 'secrets' | 'platform-config'
