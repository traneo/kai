import { useEffect, useState, useCallback, useMemo } from 'react'
import { fetchAgents, fetchPipelines, fetchStats, fetchPipelineDetail, fetchQueue, subscribeEvents } from './api'
import type { Agent, PipelineRun, Stats, PipelineDetail, Page, QueueEntry } from './types'
import { PipelineDetailView } from './components/PipelineDetailView'
import { LandingPage } from './components/LandingPage'
import { NewPipeline } from './components/NewPipeline'
import { AuditLog } from './components/AuditLog'
import { SecretsPage } from './components/SecretsPage'
import { AgentDetailPanel } from './components/AgentDetailPanel'
import { PlatformConfigPage } from './components/PlatformConfigPage'
import { NavBar } from './components/NavBar'
import './App.css'

const STATUS_ORDER: Record<string, number> = {
  running: 0,
  blocked: 1,
  queued: 2,
  pending: 3,
  completed: 4,
  passed: 4,
  failed: 5,
}

function getDisplayStatus(p: PipelineRun): string {
  if (p.has_blocked) return 'blocked'
  if (p.has_queued) return 'queued'
  return p.status === 'completed' ? 'passed' : p.status
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const sec = Math.floor(diff / 1000)
  if (sec < 60) return `${sec}s ago`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`
  return `${Math.floor(hr / 24)}d ago`
}

function debounce(fn: () => void, ms: number) {
  let timer: ReturnType<typeof setTimeout>
  return () => {
    clearTimeout(timer)
    timer = setTimeout(fn, ms)
  }
}

export default function App() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [pipelines, setPipelines] = useState<PipelineRun[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [page, setPage] = useState<Page>('landing')
  const [selectedPipeline, setSelectedPipeline] = useState<string | null>(null)
  const [pipelineDetail, setPipelineDetail] = useState<PipelineDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)
  const [queue, setQueue] = useState<QueueEntry[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string | null>(null)

  const loadPipelines = useCallback(async () => {
    const data = await fetchPipelines().catch(() => [])
    setPipelines(data)
  }, [])

  const loadQueue = useCallback(async () => {
    const data = await fetchQueue().catch(() => [])
    setQueue(data)
  }, [])

  const loadAgentsAndStats = useCallback(async () => {
    const [agentsData, statsData] = await Promise.all([
      fetchAgents().catch(() => []),
      fetchStats().catch(() => null),
    ])
    setAgents(agentsData)
    setStats(statsData)
  }, [])

  const loadAll = useCallback(async () => {
    await Promise.all([loadPipelines(), loadAgentsAndStats(), loadQueue()])
  }, [loadPipelines, loadAgentsAndStats, loadQueue])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    const refresh = debounce(() => {
      loadPipelines()
      loadQueue()
    }, 400)
    const unsub = subscribeEvents(refresh)
    return unsub
  }, [loadPipelines, loadQueue])

  useEffect(() => {
    const interval = setInterval(() => loadAgentsAndStats(), 15000)
    return () => clearInterval(interval)
  }, [loadAgentsAndStats])

  const sortedPipelines = useMemo(() => {
    return [...pipelines].sort((a, b) => {
      const pa = STATUS_ORDER[getDisplayStatus(a)] ?? 99
      const pb = STATUS_ORDER[getDisplayStatus(b)] ?? 99
      if (pa !== pb) return pa - pb
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    })
  }, [pipelines])

  const filteredPipelines = useMemo(() => {
    return sortedPipelines.filter(p => {
      if (statusFilter && getDisplayStatus(p) !== statusFilter) return false
      if (searchQuery && !p.project.toLowerCase().includes(searchQuery.toLowerCase())) return false
      return true
    })
  }, [sortedPipelines, searchQuery, statusFilter])

  async function openDetail(id: string) {
    setPage('detail')
    setSelectedPipeline(id)
    setDetailLoading(true)
    setError(null)
    try {
      const detail = await fetchPipelineDetail(id)
      setPipelineDetail(detail)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setDetailLoading(false)
    }
  }

  function navigateTo(p: Page) {
    setPage(p)
    if (p !== 'detail') {
      setSelectedPipeline(null)
      setPipelineDetail(null)
    }
    if (p === 'dashboard') loadPipelines()
  }

  const refreshDetail = useCallback(async () => {
    if (!selectedPipeline) return
    try {
      const detail = await fetchPipelineDetail(selectedPipeline)
      setPipelineDetail(detail)
    } catch (e) {
      setError(String(e))
    }
  }, [selectedPipeline])

  function handlePipelineCreated(_runId: string) {
  }

  if (page === 'landing') {
    return <LandingPage onEnter={() => navigateTo('dashboard')} onNavigate={navigateTo} />
  }

  if (page === 'detail') {
    return (
      <div className="app">
        <header className="header">
          <div className="header-brand">
            <div className="header-logo">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2" />
                <line x1="12" y1="22" x2="12" y2="15.5" />
                <polyline points="22 8.5 12 15.5 2 8.5" />
              </svg>
            </div>
            <h1><span>kai</span> Platform</h1>
          </div>
          <NavBar current={page} onNavigate={navigateTo} />
        </header>
        <header className="header" style={{ borderTop: '1px solid var(--border)', paddingTop: 0, paddingBottom: 0 }}>
          <button className="btn-back" onClick={() => navigateTo('dashboard')}>&larr; Back</button>
          <h1>{pipelineDetail?.project || 'Pipeline'} <span className="badge">{selectedPipeline}</span></h1>
        </header>
        {error && <p className="error">{error}</p>}
        {detailLoading && <p className="muted">Loading...</p>}
        {!error && !detailLoading && !pipelineDetail && <p className="muted">Unable to load pipeline details</p>}
        {pipelineDetail && <PipelineDetailView detail={pipelineDetail} onUpdated={refreshDetail} />}
      </div>
    )
  }

  if (page === 'new') {
    return (
      <div className="app">
        <header className="header">
          <div className="header-brand">
            <div className="header-logo">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2" />
                <line x1="12" y1="22" x2="12" y2="15.5" />
                <polyline points="22 8.5 12 15.5 2 8.5" />
              </svg>
            </div>
            <h1><span>kai</span> Platform</h1>
          </div>
          <NavBar current={page} onNavigate={navigateTo} />
        </header>
        <NewPipeline onCreated={handlePipelineCreated} />
      </div>
    )
  }

  if (page === 'secrets') {
    return <SecretsPage onNavigate={(p) => navigateTo(p)} />
  }

  if (page === 'platform-config') {
    return <PlatformConfigPage onNavigate={(p) => navigateTo(p)} />
  }

  if (page === 'audit') {
    return (
      <div className="app">
        <header className="header">
          <div className="header-brand">
            <div className="header-logo">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2" />
                <line x1="12" y1="22" x2="12" y2="15.5" />
                <polyline points="22 8.5 12 15.5 2 8.5" />
              </svg>
            </div>
            <h1><span>kai</span> Platform</h1>
          </div>
          <NavBar current={page} onNavigate={navigateTo} />
        </header>
        <AuditLog />
      </div>
    )
  }

  return (
      <div className="app">
        <header className="header">
        <div className="header-brand">
          <div className="header-logo">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2" />
              <line x1="12" y1="22" x2="12" y2="15.5" />
              <polyline points="22 8.5 12 15.5 2 8.5" />
            </svg>
          </div>
          <h1><span>kai</span> Platform</h1>
        </div>
        <NavBar current={page} onNavigate={navigateTo} />
      </header>

      {stats && <StatsBar stats={stats} agents={agents.length} />}

      <div className="dashboard-layout">
        <div className="dashboard-main">
          <div className="section-header">
            <h2>Pipelines</h2>
            <span className="section-count">{pipelines.length}</span>
            <div className="pipeline-toolbar">
              <input
                type="text"
                className="pipeline-search"
                placeholder="Search pipelines..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
              />
              <div className="filter-pills">
                {['running', 'queued', 'blocked', 'passed', 'failed'].map(s => (
                  <button
                    key={s}
                    className={`filter-pill ${s}${statusFilter === s ? ' active' : ''}`}
                    onClick={() => setStatusFilter(statusFilter === s ? null : s)}
                  >
                    {s === 'passed' ? 'completed' : s}
                  </button>
                ))}
              </div>
              <button className="btn-refresh" onClick={loadPipelines} title="Refresh">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="23 4 23 10 17 10" />
                  <polyline points="1 20 1 14 7 14" />
                  <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
                </svg>
              </button>
            </div>
          </div>
          {filteredPipelines.length === 0 && <p className="muted">No pipelines yet</p>}
          <div className="pipeline-list">
            {filteredPipelines.map(p => (
              <PipelineCard key={p.id} pipeline={p} onClick={() => openDetail(p.id)} />
            ))}
          </div>
        </div>

        <aside className="dashboard-sidebar">
          <div className="section-header">
            <h2>Agents</h2>
            <span className="section-count">{agents.length}</span>
          </div>
          <div className="agent-list">
            {agents.length === 0 && <p className="muted">No agents connected</p>}
            {agents.map(a => (
              <AgentCard key={a.id} agent={a} onClick={() => setSelectedAgent(a)} />
            ))}
          </div>

          {queue.length > 0 && (
            <>
              <div className="section-header" style={{ marginTop: 20 }}>
                <h2>Queue</h2>
                <span className="section-count">{queue.length}</span>
              </div>
              <div className="queue-list">
                {queue.map(entry => (
                  <QueueCard key={`${entry.run_id}-${entry.step_id}`} entry={entry} />
                ))}
              </div>
            </>
          )}
        </aside>
      </div>

      {selectedAgent && (
        <AgentDetailPanel agent={selectedAgent} onClose={() => setSelectedAgent(null)} />
      )}
    </div>
  )
}

function StatsBar({ stats, agents }: { stats: Stats; agents: number }) {
  const items = [
    { label: 'Pipelines', value: stats.pipelines, icon: 'M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5' },
    { label: 'Steps', value: stats.steps, icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2' },
    { label: 'Tokens Used', value: stats.tokens.total, icon: 'M13 2L3 14h9l-1 8 10-12h-9l1-8z' },
    { label: 'Active Agents', value: agents, icon: 'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z' },
    { label: 'Uptime', value: Math.floor(stats.duration_ms / 1000), suffix: 's', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z' },
  ]

  return (
    <div className="stats-bar">
      {items.map(item => (
        <div key={item.label} className="stat-card">
          <div className="stat-icon">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d={item.icon} />
            </svg>
          </div>
          <div className="stat-body">
            <span className="stat-value">{item.value.toLocaleString()}{item.suffix || ''}</span>
            <span className="stat-label">{item.label}</span>
          </div>
        </div>
      ))}
    </div>
  )
}

function PipelineCard({ pipeline, onClick }: { pipeline: PipelineRun; onClick: () => void }) {
  const displayStatus = pipeline.has_blocked ? 'blocked'
    : pipeline.has_queued ? 'queued'
    : pipeline.status === 'completed' ? 'passed'
    : pipeline.status === 'failed' ? 'failed'
    : pipeline.status === 'running' ? 'running' : 'pending'

  const displayLabel = pipeline.has_blocked ? 'waiting approval'
    : pipeline.has_queued ? 'queued'
    : pipeline.status

  const showProgress = pipeline.status === 'running'
  const progress = pipeline.steps > 0 ? Math.round((pipeline.passed / pipeline.steps) * 100) : 0

  return (
    <div
      className={`pipeline-card clickable ${displayStatus}`}
      onClick={onClick}
    >
      <div className="pipeline-header">
        <h3>{pipeline.project}</h3>
        <span className={`status-badge ${displayStatus}`}>
          <span className="status-dot" />
          {displayLabel}
        </span>
      </div>
      <div className="pipeline-meta">
        <span>Steps: {pipeline.passed}/{pipeline.steps}</span>
        {pipeline.failed > 0 && <span className="meta-failed">Failed: {pipeline.failed}</span>}
        <span className="pipeline-time">{timeAgo(pipeline.created_at)}</span>
      </div>
      {showProgress && (
        <div className="progress-track">
          <div className="progress-fill" style={{ width: `${progress}%` }} />
        </div>
      )}
    </div>
  )
}

function QueueCard({ entry }: { entry: QueueEntry }) {
  return (
    <div className="queue-card">
      <div className="queue-card-position">#{entry.position + 1}</div>
      <div className="queue-card-body">
        <div className="queue-card-step">{entry.step_id}</div>
        <div className="queue-card-run">{entry.run_id}</div>
        <div className="queue-card-prompt">{entry.prompt.length > 60 ? entry.prompt.slice(0, 60) + '...' : entry.prompt}</div>
      </div>
    </div>
  )
}

function AgentCard({ agent, onClick }: { agent: Agent; onClick?: () => void }) {
  const isBusy = agent.state === 'busy'
  return (
    <div className={`agent-card ${isBusy ? 'busy' : 'idle'}${onClick ? ' clickable' : ''}`} onClick={onClick}>
      <div className="agent-info">
        <span className={`agent-dot ${isBusy ? 'busy' : 'idle'}`}>
          {!isBusy && <span className="agent-ping" />}
        </span>
        <span className="agent-name">{agent.id}</span>
      </div>
      <div className="agent-status-row">
        {isBusy && <div className="agent-spinner" />}
        <span className="agent-status">{agent.state}</span>
      </div>
    </div>
  )
}
