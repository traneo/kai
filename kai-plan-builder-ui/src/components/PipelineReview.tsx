import { useState, useMemo, useCallback } from 'react'
import { createPipeline } from '../api'
import { PipelineBuilder } from './PipelineBuilder'

interface Props {
  yaml: string
  onGoBack: () => void
  onCancel: () => void
}

export function PipelineReview({ yaml, onGoBack, onCancel }: Props) {
  const [tab, setTab] = useState<'visual' | 'yaml'>('visual')
  const [running, setRunning] = useState(false)
  const [success, setSuccess] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [liveYaml, setLiveYaml] = useState(yaml)

  const showVisual = tab === 'visual'
  const showRawYaml = tab === 'yaml'

  function handleYamlChange(updated: string) {
    setLiveYaml(updated)
  }

  const handleDownload = useCallback(() => {
    const blob = new Blob([liveYaml], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'pipeline.yaml'
    a.click()
    URL.revokeObjectURL(url)
  }, [liveYaml])

  const handleRun = useCallback(async () => {
    setRunning(true)
    setError(null)
    try {
      const result = await createPipeline(liveYaml)
      setSuccess(`Pipeline submitted successfully! Run ID: ${result.id}`)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setRunning(false)
    }
  }, [liveYaml])

  const yamlLines = useMemo(() => liveYaml.split('\n').length, [liveYaml])

  return (
    <div className="review-page">
      {success && (
        <div className="toast">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
          <span>{success}</span>
          <button className="toast-close" onClick={() => setSuccess(null)}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      )}

      <div className="tab-bar">
        <button
          className={`tab-btn${showVisual ? ' active' : ''}`}
          onClick={() => setTab('visual')}
        >
          Visual Editor
        </button>
        <button
          className={`tab-btn${showRawYaml ? ' active' : ''}`}
          onClick={() => setTab('yaml')}
        >
          Raw YAML
          <span style={{
            marginLeft: 6,
            fontSize: 11,
            color: 'var(--text-muted)',
            background: 'var(--bg-elevated)',
            padding: '1px 6px',
            borderRadius: 'var(--radius-sm)',
          }}>
            {yamlLines} lines
          </span>
        </button>
      </div>

      {showVisual && (
        <PipelineBuilder
          initialYaml={yaml}
          onYamlChange={handleYamlChange}
        />
      )}

      {showRawYaml && (
        <pre className="review-yaml-output">{liveYaml}</pre>
      )}

      {error && (
        <p className="error">{error}</p>
      )}

      <div className="review-actions">
        <button className="btn btn-danger" onClick={onCancel}>
          Cancel &amp; Reset
        </button>
        <button className="btn btn-secondary" onClick={onGoBack}>
          Go Back
        </button>
        <div className="spacer" />
        <button className="btn btn-secondary" onClick={handleDownload}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" /><line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          Download YAML
        </button>
        <button className="btn" onClick={handleRun} disabled={running}>
          {running ? (
            <><span className="spinner" /> Running...</>
          ) : (
            <>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="5 3 19 12 5 21 5 3" />
              </svg>
              Run Pipeline
            </>
          )}
        </button>
      </div>
    </div>
  )
}
