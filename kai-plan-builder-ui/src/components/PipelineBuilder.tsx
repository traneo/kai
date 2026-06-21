import { useState, useMemo, useEffect } from 'react'
import type { BuilderStep, PipelineConfig } from '../types'
import { parseYaml, serializeYaml, newStep, layoutDAG } from '../utils/pipelineYaml'
import { StepModal } from './StepModal'

interface Props {
  initialYaml?: string
  onYamlChange?: (yaml: string) => void
}

export function PipelineBuilder({ initialYaml, onYamlChange }: Props) {
  const [config, setConfig] = useState<PipelineConfig>({
    project: '',
    repoURL: '',
    repoBaseBranch: 'main',
    repoProvider: '',
    repoTokenRef: '',
    outputType: 'pr',
    branchPrefix: 'feat/',
    steps: [],
  })
  const [error, setError] = useState<string | null>(null)
  const yaml = useMemo(() => serializeYaml(config), [config])

  useEffect(() => {
    if (onYamlChange) {
      onYamlChange(yaml)
    }
  }, [yaml, onYamlChange])
  const [nextStepNum, setNextStepNum] = useState(1)
  const [modalStep, setModalStep] = useState<BuilderStep | null>(null)
  const [modalMode, setModalMode] = useState<'add' | 'edit'>('add')
  const [showImport, setShowImport] = useState(false)
  const [importYaml, setImportYaml] = useState('')
  const [importError, setImportError] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)

  useEffect(() => {
    if (initialYaml && !initialized) {
      try {
        const parsed = parseYaml(initialYaml)
        if (parsed.project.trim()) {
          setConfig(parsed)
          const maxNum = parsed.steps.reduce((max, s) => {
            const match = s.id.match(/step-(\d+)/)
            return match ? Math.max(max, parseInt(match[1]) + 1) : max
          }, 1)
          setNextStepNum(maxNum)
        } else {
          setError('Generated YAML is missing project name. Check the Raw YAML tab to review and fix.')
        }
      } catch {
        setError('Could not load generated YAML into visual editor. The format may not match expectations. Switch to the Raw YAML tab to review and edit manually, or use Import YAML.')
      }
      setInitialized(true)
    }
  }, [initialYaml, initialized])

  function updateConfig(updates: Partial<PipelineConfig>) {
    setConfig(prev => ({ ...prev, ...updates }))
  }

  function openAddModal() {
    const id = `step-${nextStepNum}`
    setModalStep(newStep(id))
    setModalMode('add')
  }

  function openEditModal(step: BuilderStep) {
    setModalStep({ ...step })
    setModalMode('edit')
  }

  function closeModal() {
    setModalStep(null)
  }

  function saveModalStep(updated: BuilderStep) {
    if (modalMode === 'add') {
      setConfig(prev => ({ ...prev, steps: [...prev.steps, updated] }))
      setNextStepNum(n => n + 1)
    } else {
      setConfig(prev => ({
        ...prev,
        steps: prev.steps.map(s =>
          s.id === modalStep?.id ? {
            ...updated,
            dependsOn: updated.dependsOn.filter(d => d !== updated.id),
          } : s
        ),
      }))
    }
    closeModal()
  }

  function removeStep(id: string) {
    setConfig(prev => ({
      ...prev,
      steps: prev.steps.filter(s => s.id !== id).map(s => ({
        ...s,
        dependsOn: s.dependsOn.filter(d => d !== id),
      })),
    }))
    closeModal()
  }

  function handleImport() {
    setImportError(null)
    try {
      const parsed = parseYaml(importYaml)
      if (!parsed.project.trim()) {
        setImportError('YAML must include a project name')
        return
      }
      if (parsed.steps.length === 0) {
        setImportError('YAML must include at least one step')
        return
      }
      setConfig(parsed)
      const maxNum = parsed.steps.reduce((max, s) => {
        const match = s.id.match(/step-(\d+)/)
        return match ? Math.max(max, parseInt(match[1]) + 1) : max
      }, 1)
      setNextStepNum(maxNum)
      setShowImport(false)
      setImportYaml('')
    } catch (e) {
      setImportError(e instanceof Error ? e.message : 'Invalid YAML')
    }
  }

  function closeImport() {
    setShowImport(false)
    setImportYaml('')
    setImportError(null)
  }

  const dagLevels = useMemo(() => layoutDAG(config.steps), [config.steps])

  return (
    <div className="pipeline-builder">
      <div className="builder-header">
        <div className="builder-header-left">
          <h2 style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>Create Pipeline</h2>
        </div>
        <div className="builder-header-right">
          <button className="builder-yaml-btn" onClick={() => setShowImport(true)}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" /><line x1="12" y1="15" x2="12" y2="3" />
            </svg>
            Import YAML
          </button>
        </div>
      </div>

      <div className="builder-content">
        <div className="builder-card">
          <div className="builder-card-title">Project Settings</div>
          <div className="builder-settings-grid">
            <div className="builder-field">
              <label className="builder-label">Project Name</label>
              <input
                className="builder-input"
                type="text"
                placeholder="my-project"
                value={config.project}
                onChange={e => updateConfig({ project: e.target.value })}
              />
            </div>
            <div className="builder-field">
              <label className="builder-label">Output Type</label>
              <select
                className="builder-select"
                value={config.outputType}
                onChange={e => updateConfig({ outputType: e.target.value })}
              >
                <option value="pr">Pull Request</option>
                <option value="branch">Branch</option>
                <option value="direct">Direct Commit</option>
              </select>
            </div>
            <div className="builder-field">
              <label className="builder-label">Branch Prefix</label>
              <input
                className="builder-input"
                type="text"
                placeholder="feat/"
                value={config.branchPrefix}
                onChange={e => updateConfig({ branchPrefix: e.target.value })}
              />
            </div>
            <div className="builder-field">
              <label className="builder-label">Repo URL</label>
              <input
                className="builder-input"
                type="text"
                placeholder="https://github.com/org/repo"
                value={config.repoURL}
                onChange={e => updateConfig({ repoURL: e.target.value })}
              />
            </div>
            <div className="builder-field">
              <label className="builder-label">Base Branch</label>
              <input
                className="builder-input"
                type="text"
                placeholder="main"
                value={config.repoBaseBranch}
                onChange={e => updateConfig({ repoBaseBranch: e.target.value })}
              />
            </div>
            <div className="builder-field">
              <label className="builder-label">Repo Provider</label>
              <input
                className="builder-input"
                type="text"
                placeholder="forgejo, github, gitlab..."
                value={config.repoProvider}
                onChange={e => updateConfig({ repoProvider: e.target.value })}
              />
            </div>
            <div className="builder-field">
              <label className="builder-label">Token Ref</label>
              <input
                className="builder-input"
                type="text"
                placeholder="forgejo-token"
                value={config.repoTokenRef}
                onChange={e => updateConfig({ repoTokenRef: e.target.value })}
              />
            </div>
          </div>
        </div>

        <div className="builder-card">
          <div className="builder-card-row">
            <div className="builder-card-title">Pipeline Graph</div>
            <button className="builder-add-node-btn" onClick={openAddModal}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              Add Step
            </button>
          </div>

          {config.steps.length === 0 ? (
            <div className="builder-graph-empty" onClick={openAddModal}>
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              <span>Click to add your first step</span>
            </div>
          ) : (
            <div className="builder-graph-canvas">
              <div
                className="builder-dag-grid"
                style={{ gridTemplateColumns: `repeat(${dagLevels.length}, 1fr)` }}
              >
                {dagLevels.map((col, ci) => (
                  <div key={ci} className="builder-dag-col">
                    {col.map(step => (
                      <div
                        key={step.id}
                        className="builder-dag-node"
                        onClick={() => openEditModal(step)}
                        title="Click to edit"
                      >
                        <span className="builder-dag-node-id">{step.id}</span>
                        {step.validation.length > 0 && (
                          <span className="builder-dag-badge" title="Has validation gates">
                            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                              <polyline points="20 6 9 17 4 12" />
                            </svg>
                          </span>
                        )}
                        {step.approval === 'required' && (
                          <span className="builder-dag-badge approval" title="Requires approval">
                            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                              <path d="M12 9v4" /><path d="M12 17h.01" />
                              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                            </svg>
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {error && (
          <p className="error" style={{ marginTop: 8 }}>{error}</p>
        )}
      </div>

      {showImport && (
        <div className="modal-overlay" onClick={closeImport} onKeyDown={e => { if (e.key === 'Escape') closeImport() }} tabIndex={0}>
          <div className="modal import-modal" onClick={e => e.stopPropagation()}>
            <div className="import-modal-header">
              <h3 style={{ margin: 0, fontSize: 15 }}>Import YAML</h3>
              <button className="builder-modal-close" onClick={closeImport}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
            <div className="import-modal-body">
              <p style={{ fontSize: 12, color: 'var(--text-tertiary)', marginBottom: 10 }}>
                Paste a pipeline YAML configuration below to load it into the editor.
              </p>
              <textarea
                className="import-textarea"
                value={importYaml}
                onChange={e => setImportYaml(e.target.value)}
                placeholder={`version: 1\nproject: my-project\nrepo:\n  url: https://github.com/org/repo\n  base_branch: main\noutput:\n  type: pr\n  branch_prefix: feat/\nsteps:\n  - id: step-1\n    prompt: |\n      Implement the feature\n    depends_on:\n      - step-0\n    validation:\n      - lint\n    approval: optional`}
                spellCheck={false}
              />
              {importError && (
                <p style={{ color: 'var(--accent-red)', fontSize: 12, marginTop: 8 }}>
                  {importError}
                </p>
              )}
            </div>
            <div className="import-modal-actions">
              <button className="btn-small" onClick={closeImport}>Cancel</button>
              <button
                className="btn-small btn-approve"
                onClick={handleImport}
                disabled={!importYaml.trim()}
                style={{ opacity: importYaml.trim() ? 1 : 0.4, cursor: importYaml.trim() ? 'pointer' : 'not-allowed' }}
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 4 }}>
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                Load Pipeline
              </button>
            </div>
          </div>
        </div>
      )}

      {modalStep && (
        <StepModal
          step={modalStep}
          allSteps={config.steps}
          mode={modalMode}
          onSave={saveModalStep}
          onDelete={modalMode === 'edit' ? () => removeStep(modalStep.id) : undefined}
          onClose={closeModal}
        />
      )}
    </div>
  )
}
