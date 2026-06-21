import { useState } from 'react'
import type { Page } from './types'
import { Layout } from './components/Layout'
import { Dashboard } from './components/Dashboard'
import { LiveStream } from './components/LiveStream'
import { LogTable } from './components/LogTable'
import { FilterBar } from './components/FilterBar'
import { RunExplorer } from './components/RunExplorer'
import { RunTimeline } from './components/RunTimeline'
import { ErrorDashboard } from './components/ErrorDashboard'
import { PerformanceCharts } from './components/PerformanceCharts'
import { LogDetail } from './components/LogDetail'
import type { QueryFilter, LogEntry } from './types'
import { fetchLog } from './api'
import { RefreshProvider } from './RefreshContext'

export default function App() {
  const [page, setPage] = useState<Page>('dashboard')
  const [logFilter, setLogFilter] = useState<QueryFilter>({})
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)
  const [selectedLogId, setSelectedLogId] = useState<string | null>(null)
  const [detailEntry, setDetailEntry] = useState<LogEntry | null>(null)

  function navigateTo(p: Page) {
    setPage(p)
  }

  function handleSelectRun(runId: string) {
    setSelectedRunId(runId)
    setPage('run-detail')
  }

  async function handleSelectLog(id: string) {
    setSelectedLogId(id)
    const entry = await fetchLog(id)
    setDetailEntry(entry)
    setPage('log-detail')
  }

  function renderPage() {
    switch (page) {
      case 'dashboard':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Dashboard">
            <Dashboard />
          </Layout>
        )
      case 'live':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Live Stream">
            <LiveStream />
          </Layout>
        )
      case 'logs':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Log Explorer">
            <FilterBar filter={logFilter} onChange={setLogFilter} />
            <LogTable filter={logFilter} />
          </Layout>
        )
      case 'log-detail':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Log Detail">
            {detailEntry ? (
              <div style={{ position: 'relative', minHeight: 0 }}>
                <LogDetail entry={detailEntry} onClose={() => navigateTo('logs')} />
              </div>
            ) : (
              <p>Log entry not found</p>
            )}
          </Layout>
        )
      case 'runs':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Run Explorer">
            <RunExplorer onSelectRun={handleSelectRun} />
          </Layout>
        )
      case 'run-detail':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Run Timeline">
            {selectedRunId && <RunTimeline runId={selectedRunId} />}
          </Layout>
        )
      case 'errors':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Error Dashboard">
            <ErrorDashboard />
          </Layout>
        )
      case 'analytics':
        return (
          <Layout current={page} onNavigate={navigateTo} title="Analytics">
            <PerformanceCharts />
          </Layout>
        )
    }
  }

  return (
    <RefreshProvider>
      {renderPage()}
    </RefreshProvider>
  )
}
