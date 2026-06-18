import { useState, useEffect, useCallback } from 'react'
import { fetchSecrets, setSecret, deleteSecret } from '../api'
import type { SecretMeta, Page } from '../types'
import { NavBar } from './NavBar'

interface Props {
  onNavigate: (page: Page) => void
}

export function SecretsPage({ onNavigate }: Props) {
  const [secrets, setSecrets] = useState<SecretMeta[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [formName, setFormName] = useState('')
  const [formValue, setFormValue] = useState('')
  const [formDesc, setFormDesc] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchSecrets()
      setSecrets(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError(null)
    try {
      await setSecret(formName, formValue, formDesc || undefined)
      setFormName('')
      setFormValue('')
      setFormDesc('')
      setShowForm(false)
      await load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : String(e))
    }
  }

  async function handleDelete(name: string) {
    if (!window.confirm(`Delete secret "${name}"? This cannot be undone.`)) return
    try {
      await deleteSecret(name)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
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
        <NavBar current="secrets" onNavigate={onNavigate} />
      </header>

      <div className="secrets-page">
        <div className="section-header">
          <h2>Secrets</h2>
          <span className="section-count">{secrets.length}</span>
          <button className="btn-primary" onClick={() => setShowForm(true)} style={{ marginLeft: 'auto' }}>
            + Add Secret
          </button>
        </div>

        {error && <p className="error">{error}</p>}

        {showForm && (
          <form className="secret-form" onSubmit={handleCreate}>
            <div className="secret-form-fields">
              <input
                className="pipeline-search"
                placeholder="Secret name (e.g. forgejo-token)"
                value={formName}
                onChange={e => setFormName(e.target.value)}
                required
                style={{ maxWidth: 250 }}
              />
              <input
                className="pipeline-search"
                type="password"
                placeholder="Token value"
                value={formValue}
                onChange={e => setFormValue(e.target.value)}
                required
                style={{ maxWidth: 350, fontFamily: 'var(--font-mono)' }}
              />
              <input
                className="pipeline-search"
                placeholder="Description (optional)"
                value={formDesc}
                onChange={e => setFormDesc(e.target.value)}
                style={{ maxWidth: 250 }}
              />
            </div>
            {formError && <p className="error" style={{ marginTop: 8 }}>{formError}</p>}
            <div className="secret-form-actions">
              <button type="submit" className="btn-primary">Save</button>
              <button type="button" className="btn-secondary" onClick={() => { setShowForm(false); setFormError(null) }}>Cancel</button>
            </div>
          </form>
        )}

        {loading && <p className="muted">Loading...</p>}

        {!loading && secrets.length === 0 && !showForm && (
          <p className="muted">No secrets yet. Add your first git token to get started.</p>
        )}

        <div className="secrets-list">
          {secrets.map(s => (
            <div key={s.name} className="secret-card">
              <div className="secret-card-body">
                <div className="secret-card-name">{s.name}</div>
                {s.description && <div className="secret-card-desc">{s.description}</div>}
                <div className="secret-card-meta">
                  Created {timeAgo(s.created_at)} &middot; Updated {timeAgo(s.updated_at)}
                </div>
              </div>
              <div className="secret-card-actions">
                <button className="btn-danger" onClick={() => handleDelete(s.name)} title="Delete secret">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
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
