import { useState, useEffect, useCallback } from 'react'
import { fetchActiveConfig, fetchConfigVersions, createConfigDraft, updateConfigDraft, publishConfigVersion, activateConfigVersion, rollbackConfigVersion } from '../api'
import type { SystemConfig, ConfigVersion, ReloadResult, Page } from '../types'
import { NavBar } from './NavBar'

interface Props {
  onNavigate: (page: Page) => void
}

export function PlatformConfigPage({ onNavigate }: Props) {
  const [versions, setVersions] = useState<ConfigVersion[]>([])
  const [activeConfig, setActiveConfig] = useState<SystemConfig | null>(null)
  const [selectedVersion, setSelectedVersion] = useState<ConfigVersion | null>(null)
  const [draftVersion, setDraftVersion] = useState<ConfigVersion | null>(null)
  const [editing, setEditing] = useState(false)
  const [draftConfig, setDraftConfig] = useState<SystemConfig | null>(null)
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [serviceAvailable, setServiceAvailable] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [active, vers] = await Promise.all([
        fetchActiveConfig(),
        fetchConfigVersions()
      ])
      setActiveConfig(active)
      setVersions(vers)
      setServiceAvailable(true)
      if (vers.length > 0) {
        setSelectedVersion(vers[0])
      }
    } catch (e) {
      setServiceAvailable(false)
      setError('Config service not running. Start it with: scripts/start-dev.sh')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  function startDraft() {
    if (!activeConfig) return
    setDraftConfig(JSON.parse(JSON.stringify(activeConfig)))
    setMessage('')
    setEditing(true)
    setSuccess(null)
  }

  async function saveDraft(): Promise<string | null> {
    if (!draftConfig) return null
    setError(null)
    setSuccess(null)
    try {
      let draft = draftVersion
      if (!draft) {
        draft = await createConfigDraft()
        if (!draft) throw new Error('failed to create draft')
        setDraftVersion(draft)
      }
      const updated = await updateConfigDraft(draft.id, draftConfig, message)
      setDraftVersion(updated)
      setSuccess('Draft saved')
      await load()
      return updated.id
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return null
    }
  }

  async function publish() {
    setError(null)
    setSuccess(null)
    try {
      const id = await saveDraft()
      if (!id) return
      await publishConfigVersion(id)
      const result = await activateConfigVersion(id)
      setDraftVersion(null)
      setEditing(false)
      const msg = result.hot_reloaded?.length
        ? `Activated! Applied: ${result.hot_reloaded.join(', ')}`
        : 'Published & activated'
      if (result.requires_restart?.length) {
        setSuccess(`${msg}. Restart needed: ${result.requires_restart.join(', ')}`)
      } else {
        setSuccess(msg)
      }
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  async function activate(id: string) {
    setError(null)
    setSuccess(null)
    try {
      const result: ReloadResult = await activateConfigVersion(id)
      const msg = result.hot_reloaded?.length
        ? `Activated! Applied: ${result.hot_reloaded.join(', ')}`
        : 'Activated'
      if (result.requires_restart?.length) {
        setSuccess(`${msg}. Restart needed: ${result.requires_restart.join(', ')}`)
      } else {
        setSuccess(msg)
      }
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  async function rollback(id: string) {
    const v = versions.find(x => x.id === id)
    if (!window.confirm(`Rollback to v${v?.version}? This will create and activate a new draft based on it.`)) return
    setError(null)
    setSuccess(null)
    try {
      await rollbackConfigVersion(id)
      setSuccess('Rollback complete')
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  function updatePlatformField(section: string, field: string, value: unknown) {
    if (!draftConfig) return
    const updated = JSON.parse(JSON.stringify(draftConfig))
    const parts = section.split('.')
    let obj: Record<string, any> = updated.platform
    for (const p of parts) {
      obj = obj[p]
    }
    obj[field] = value
    setDraftConfig(updated)
  }

  function selectVersion(v: ConfigVersion) {
    setSelectedVersion(v)
    setEditing(false)
    setDraftVersion(null)
    setDraftConfig(null)
    setError(null)
    setSuccess(null)
  }

  const displayConfig = draftConfig || selectedVersion?.config || activeConfig
  const isDraft = editing && draftConfig
  const statusColors: Record<string, string> = { active: 'var(--accent-green)', published: 'var(--accent-primary)', draft: 'var(--accent-amber)' }

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
        <NavBar current="platform-config" onNavigate={onNavigate} />
      </header>

      <div className="config-page">
        <div className="section-header">
          <h2>Platform Configuration</h2>
          {serviceAvailable && !editing && <button className="btn-primary" onClick={startDraft} style={{ marginLeft: 'auto' }}>+ New Draft</button>}
        </div>

        {error && <p className="error">{error}</p>}
        {success && <p className="config-success">{success}</p>}

        {!serviceAvailable && (
          <div className="config-service-down">
            <h3>Config Service Required</h3>
            <p>The platform config page requires the <strong>kai-config-service</strong> to be running.</p>
            <p>Start it with:</p>
            <pre>scripts/start-dev.sh</pre>
            <p>Or manually:</p>
            <pre>cd kai-config-service && go run ./cmd/server \
  --port 8081 \
  --data-dir ./data/config \
  --orchestrator-url http://localhost:8080</pre>
          </div>
        )}

        {serviceAvailable && versions.length > 0 && (
          <div className="config-version-bar">
            {versions.map(v => (
              <button
                key={v.id}
                className={`config-version-pill ${selectedVersion?.id === v.id ? 'active' : ''} ${v.status}`}
                onClick={() => selectVersion(v)}
              >
                <span className="config-version-num">v{v.version}</span>
                <span className="config-version-badge" style={{ background: statusColors[v.status] || 'var(--text-muted)' }}>{v.status}</span>
              </button>
            ))}
          </div>
        )}

        {serviceAvailable && selectedVersion && (
          <div className="config-actions">
            {selectedVersion.status === 'active' && !editing && <button className="btn-primary" onClick={startDraft}>Edit (Create Draft)</button>}
            {selectedVersion.status === 'published' && !editing && <button className="btn-primary" onClick={() => activate(selectedVersion.id)}>Activate</button>}
            {selectedVersion.status === 'published' && !editing && <button className="btn-secondary" onClick={() => rollback(selectedVersion.id)}>Rollback Here</button>}
            {isDraft && <button className="btn-primary" onClick={saveDraft}>Save Draft</button>}
            {isDraft && <button className="btn-primary" onClick={publish} style={{ borderColor: 'var(--accent-green)', color: 'var(--accent-green)' }}>Publish & Activate</button>}
            {isDraft && <button className="btn-secondary" onClick={() => { setEditing(false); setDraftVersion(null); setDraftConfig(null); setError(null); setSuccess(null) }}>Cancel</button>}
          </div>
        )}

        {displayConfig && (
          <div className="config-sections">
            <div className="config-section">
              <div className="config-section-header">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2" /><rect x="2" y="14" width="20" height="8" rx="2" ry="2" /><line x1="6" y1="6" x2="6.01" y2="6" /><line x1="6" y1="18" x2="6.01" y2="18" /></svg>
                Server</div>
              <div className="config-section-grid">
                <div className="config-field"><label>HTTP Port</label><span className="config-value">{displayConfig.platform.server.http_port}</span></div>
                <div className="config-field"><label>gRPC Port</label><span className="config-value">{displayConfig.platform.server.grpc_port}</span></div>
                <div className="config-field"><label>Version</label><span className="config-value">{displayConfig.platform.server.version}</span></div>
              </div>
            </div>

            <div className="config-section">
              <div className="config-section-header">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
                Authentication <span className="config-restart-note">(restart required)</span></div>
              <div className="config-section-grid">
                <div className="config-field">
                  <label>Mode</label>
                  {isDraft ? (
                    <select className="config-select" value={displayConfig.platform.auth.mode} onChange={e => updatePlatformField('auth', 'mode', e.target.value)}>
                      <option value="insecure">Insecure</option>
                      <option value="token">Token</option>
                      <option value="mtls">mTLS</option>
                    </select>
                  ) : (
                    <span className="config-value">{displayConfig.platform.auth.mode}</span>
                  )}
                </div>
                <div className="config-field">
                  <label>Token</label>
                  {isDraft ? (
                    <input className="config-input" type="password" value={displayConfig.platform.auth.pre_shared_token || ''} onChange={e => updatePlatformField('auth', 'pre_shared_token', e.target.value)} placeholder="bearer token" />
                  ) : (
                    <span className="config-value">{displayConfig.platform.auth.pre_shared_token ? '••••••••' : '(not set)'}</span>
                  )}
                </div>
                <div className="config-field"><label>TLS Cert</label><span className="config-value">{displayConfig.platform.auth.tls_cert_file || '(not set)'}</span></div>
                <div className="config-field"><label>TLS Key</label><span className="config-value">{displayConfig.platform.auth.tls_key_file || '(not set)'}</span></div>
                <div className="config-field"><label>TLS CA</label><span className="config-value">{displayConfig.platform.auth.tls_ca_cert_file || '(not set)'}</span></div>
              </div>
            </div>

            <div className="config-section">
              <div className="config-section-header">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>
                Agent Pool</div>
              <div className="config-section-grid">
                {(['heartbeat_timeout', 'health_interval'] as const).map(f => (
                  <div className="config-field" key={f}>
                    <label>{f === 'heartbeat_timeout' ? 'Heartbeat Timeout' : 'Health Interval'}</label>
                    {isDraft ? (
                      <input className="config-input config-input-narrow" value={(displayConfig.platform.pool as any)[f]} onChange={e => updatePlatformField('pool', f, e.target.value)} />
                    ) : (
                      <span className="config-value">{(displayConfig.platform.pool as any)[f]}</span>
                    )}
                  </div>
                ))}
              </div>
            </div>

            <div className="config-section">
              <div className="config-section-header">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" /><polyline points="3.27 6.96 12 12.01 20.73 6.96" /><line x1="12" y1="22.08" x2="12" y2="12" /></svg>
                Backend Status</div>
              <div className="config-section-grid">
                {(['secrets', 'audit', 'archive'] as const).map(f => (
                  <div className="config-field" key={f}>
                    <label>{f.charAt(0).toUpperCase() + f.slice(1)}</label>
                    <span className="config-value config-status-ok">{(displayConfig.platform.backends as any)[f]}</span>
                  </div>
                ))}
                <div className="config-field">
                  <label>Plugin Dir</label>
                  <span className="config-value">{displayConfig.platform.backends.plugin_dir}</span>
                </div>
                <div className="config-field">
                  <label>Database</label>
                  <span className="config-value">{displayConfig.platform.backends.database_url ? 'PostgreSQL' : '(none)'}</span>
                </div>
              </div>
            </div>

            {isDraft && (
              <div className="config-section">
                <div className="config-section-header">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" /><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" /></svg>
                  Change Description</div>
                <textarea className="config-textarea" value={message} onChange={e => setMessage(e.target.value)} placeholder="Describe what changed in this version..." rows={3} />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
