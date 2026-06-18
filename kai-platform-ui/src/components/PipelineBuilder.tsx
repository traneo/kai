import { useState, useMemo, useCallback } from 'react'
import { createPipeline } from '../api'

interface StepPolicy {
  allowedDirs: string[]
  agent: string
  allowedTools: string[]
  allowedCommands: string[]
  maxRetries: number
  timeoutSeconds: number
  retryDelaySeconds: number
  retryBackoff: string
  saveState: boolean
}

interface BuilderStep {
  id: string
  prompt: string
  dependsOn: string[]
  validation: string[]
  approval: string
  policy: StepPolicy
}

interface PipelineConfig {
  project: string
  repoURL: string
  repoBaseBranch: string
  outputType: string
  branchPrefix: string
  steps: BuilderStep[]
}

function defaultPolicy(): StepPolicy {
  return {
    allowedDirs: [],
    agent: '',
    allowedTools: [],
    allowedCommands: [],
    maxRetries: 0,
    timeoutSeconds: 0,
    retryDelaySeconds: 0,
    retryBackoff: 'linear',
    saveState: false,
  }
}

function newStep(id: string): BuilderStep {
  return {
    id,
    prompt: '',
    dependsOn: [],
    validation: [],
    approval: 'optional',
    policy: defaultPolicy(),
  }
}

const VALIDATION_GATES = ['exit_zero', 'lint', 'typecheck', 'tests', 'diff_review']

function parseYaml(yaml: string): PipelineConfig {
  const lines = yaml.split('\n')
  const config: PipelineConfig = {
    project: '',
    repoURL: '',
    repoBaseBranch: 'main',
    outputType: 'pr',
    branchPrefix: 'feat/',
    steps: [],
  }

  let i = 0
  let currentStep: Partial<BuilderStep> | null = null
  let inPrompt = false
  let promptLines: string[] = []
  let promptIndent = 0

  function getIndent(line: string): number {
    return line.search(/\S|$/)
  }

  function finishPrompt() {
    if (currentStep && inPrompt) {
      currentStep.prompt = promptLines.join('\n')
      promptLines = []
      inPrompt = false
    }
  }

  function finishStep() {
    if (currentStep && currentStep.id) {
      const policy = currentStep.policy || defaultPolicy()
      config.steps.push({
        id: currentStep.id || '',
        prompt: currentStep.prompt || '',
        dependsOn: currentStep.dependsOn || [],
        validation: currentStep.validation || [],
        approval: currentStep.approval || 'optional',
        policy: {
          allowedDirs: policy.allowedDirs || [],
          agent: policy.agent || '',
          allowedTools: policy.allowedTools || [],
          allowedCommands: policy.allowedCommands || [],
          maxRetries: policy.maxRetries || 0,
          timeoutSeconds: policy.timeoutSeconds || 0,
          retryDelaySeconds: policy.retryDelaySeconds || 0,
          retryBackoff: policy.retryBackoff || 'linear',
          saveState: policy.saveState || false,
        },
      })
    }
    currentStep = null
    inPrompt = false
    promptLines = []
  }

  for (; i < lines.length; i++) {
    const raw = lines[i]
    const line = raw.replace(/#.*$/, '').trimEnd()

    if (!line.trim() || line.trim().startsWith('#')) {
      if (inPrompt && raw.includes(promptLines[promptLines.length - 1] || '')) {
        promptLines.push('')
      }
      continue
    }

    if (inPrompt) {
      const indent = getIndent(raw)
      if (indent >= promptIndent) {
        promptLines.push(raw.slice(promptIndent))
        continue
      } else {
        finishPrompt()
      }
    }

    const indent = getIndent(raw)
    const trimmed = line.trim()

    if (trimmed === 'version: 1') continue
    if (trimmed === 'version: "1"') continue
    if (trimmed === "version: '1'") continue

    const colonIdx = trimmed.indexOf(':')
    if (colonIdx === -1) continue

    let key = trimmed.slice(0, colonIdx).trim()
    let val = trimmed.slice(colonIdx + 1).trim()

    if (key.startsWith('- ')) {
      key = key.slice(2).trim()
    }

    if (key === 'project' && indent === 0) {
      config.project = val
    } else if (key === 'url' && indent === 2) {
      config.repoURL = val
    } else if (key === 'base_branch' && indent === 2) {
      config.repoBaseBranch = val
    } else if (key === 'provider' && indent === 2) {
    } else if (key === 'token_ref' && indent === 2) {
    } else if (key === 'type' && indent === 2) {
      config.outputType = val
    } else if (key === 'branch_prefix' && indent === 2) {
      config.branchPrefix = val
    } else if (key === 'id' && indent === 2) {
      finishStep()
      currentStep = { id: val, policy: defaultPolicy() }
    } else if (key === 'prompt' && indent === 4) {
      if (val === '|') {
        inPrompt = true
        promptLines = []
        promptIndent = indent + 2
      } else if (val) {
        if (currentStep) currentStep.prompt = val
      }
    } else if (key === 'depends_on') {
      if (val.startsWith('[') && val.endsWith(']')) {
        if (currentStep) currentStep.dependsOn = val.slice(1, -1).split(',').map(s => s.trim()).filter(Boolean)
      } else {
        const deps: string[] = []
        i++
        while (i < lines.length) {
          const depRaw = lines[i].trim()
          if (!depRaw || depRaw.startsWith('#')) { i++; continue }
          const depIndent = getIndent(lines[i])
          if (depIndent <= 4) break
          if (depRaw.startsWith('- ')) {
            deps.push(depRaw.slice(2).trim())
          }
          i++
        }
        i--
        if (currentStep) currentStep.dependsOn = deps
      }
    } else if (key === 'validation') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const vRaw = lines[i].trim()
        if (!vRaw || vRaw.startsWith('#')) { i++; continue }
        const vIndent = getIndent(lines[i])
        if (vIndent <= 4) break
        if (vRaw.startsWith('- ')) {
          vals.push(vRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) currentStep.validation = vals
    } else if (key === 'approval' && indent === 3) {
      if (currentStep) currentStep.approval = val
    } else if (key === 'max_retries') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.maxRetries = parseInt(val) || 0
      }
    } else if (key === 'timeout_seconds') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.timeoutSeconds = parseInt(val) || 0
      }
    } else if (key === 'retry_delay_seconds') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.retryDelaySeconds = parseInt(val) || 0
      }
    } else if (key === 'retry_backoff') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.retryBackoff = val
      }
    } else if (key === 'save_state') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.saveState = val === 'true'
      }
    } else if (key === 'allowed_dirs') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const dRaw = lines[i].trim()
        if (!dRaw || dRaw.startsWith('#')) { i++; continue }
        const dIndent = getIndent(lines[i])
        if (dIndent <= 5) break
        if (dRaw.startsWith('- ')) {
          vals.push(dRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.allowedDirs = vals
      }
    } else if (key === 'agent') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.agent = lines[i].substring(key.length + 1).trim()
      }
    } else if (key === 'allowed_tools') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const tRaw = lines[i].trim()
        if (!tRaw || tRaw.startsWith('#')) { i++; continue }
        const tIndent = getIndent(lines[i])
        if (tIndent <= 5) break
        if (tRaw.startsWith('- ')) {
          vals.push(tRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.allowedTools = vals
      }
    } else if (key === 'allowed_commands') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const cRaw = lines[i].trim()
        if (!cRaw || cRaw.startsWith('#')) { i++; continue }
        const cIndent = getIndent(lines[i])
        if (cIndent <= 5) break
        if (cRaw.startsWith('- ')) {
          vals.push(cRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.allowedCommands = vals
      }
    }
  }

  finishPrompt()
  finishStep()

  return config
}

interface Props {
  onCreated: (runId: string) => void
}

function serializeYaml(config: PipelineConfig): string {
  const indent = (n: number) => '  '.repeat(n)

  let yaml = `version: 1\n`
  yaml += `project: ${config.project}\n`

  if (config.repoURL) {
    yaml += `repo:\n`
    yaml += `${indent(1)}url: ${config.repoURL}\n`
    if (config.repoBaseBranch) {
      yaml += `${indent(1)}base_branch: ${config.repoBaseBranch}\n`
    }
  }

  yaml += `output:\n`
  yaml += `${indent(1)}type: ${config.outputType}\n`
  if (config.branchPrefix) {
    yaml += `${indent(1)}branch_prefix: ${config.branchPrefix}\n`
  }

  yaml += `steps:\n`
  for (const step of config.steps) {
    yaml += `${indent(1)}- id: ${step.id}\n`
    yaml += `${indent(2)}prompt: |\n`
    for (const line of step.prompt.split('\n')) {
      yaml += `${indent(3)}${line || ''}\n`
    }
    if (step.dependsOn.length > 0) {
      yaml += `${indent(2)}depends_on:\n`
      for (const dep of step.dependsOn) {
        yaml += `${indent(3)}- ${dep}\n`
      }
    }
    if (step.validation.length > 0) {
      yaml += `${indent(2)}validation:\n`
      for (const v of step.validation) {
        yaml += `${indent(3)}- ${v}\n`
      }
    }
    if (step.approval) {
      yaml += `${indent(2)}approval: ${step.approval}\n`
    }

    const p = step.policy
    const hasPolicy = p.maxRetries > 0 || p.timeoutSeconds > 0 || p.retryDelaySeconds > 0
      || p.allowedDirs.length > 0 || p.agent !== '' || p.allowedTools.length > 0
      || p.allowedCommands.length > 0 || p.retryBackoff !== 'linear' || p.saveState

    if (hasPolicy) {
      yaml += `${indent(2)}policy:\n`
      for (const dir of p.allowedDirs) yaml += `${indent(3)}allowed_dirs:\n${indent(4)}- ${dir}\n`
      if (p.agent) yaml += `${indent(3)}agent: ${p.agent}\n`
      for (const tool of p.allowedTools) yaml += `${indent(3)}allowed_tools:\n${indent(4)}- ${tool}\n`
      for (const cmd of p.allowedCommands) yaml += `${indent(3)}allowed_commands:\n${indent(4)}- ${cmd}\n`
      if (p.maxRetries > 0) yaml += `${indent(3)}max_retries: ${p.maxRetries}\n`
      if (p.timeoutSeconds > 0) yaml += `${indent(3)}timeout_seconds: ${p.timeoutSeconds}\n`
      if (p.retryDelaySeconds > 0) yaml += `${indent(3)}retry_delay_seconds: ${p.retryDelaySeconds}\n`
      if (p.retryBackoff !== 'linear') yaml += `${indent(3)}retry_backoff: ${p.retryBackoff}\n`
      if (p.saveState) yaml += `${indent(3)}save_state: true\n`
    }
  }

  return yaml
}

function layoutDAG(steps: BuilderStep[]): BuilderStep[][] {
  const depths = new Map<string, number>()

  function getDepth(id: string): number {
    if (depths.has(id)) return depths.get(id)!
    const s = steps.find(x => x.id === id)
    if (!s || s.dependsOn.length === 0) {
      depths.set(id, 0)
      return 0
    }
    let maxDep = 0
    for (const dep of s.dependsOn) {
      maxDep = Math.max(maxDep, getDepth(dep) + 1)
    }
    depths.set(id, maxDep)
    return maxDep
  }

  for (const s of steps) getDepth(s.id)
  const maxDepth = Math.max(...Array.from(depths.values()), 0)
  const levels: BuilderStep[][] = Array.from({ length: maxDepth + 1 }, () => [])
  for (const s of steps) {
    levels[depths.get(s.id)!].push(s)
  }
  return levels
}

export function PipelineBuilder({ onCreated }: Props) {
  const [config, setConfig] = useState<PipelineConfig>({
    project: '',
    repoURL: '',
    repoBaseBranch: 'main',
    outputType: 'pr',
    branchPrefix: 'feat/',
    steps: [],
  })
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showYaml, setShowYaml] = useState(false)
  const [nextStepNum, setNextStepNum] = useState(1)
  const [modalStep, setModalStep] = useState<BuilderStep | null>(null)
  const [modalMode, setModalMode] = useState<'add' | 'edit'>('add')
  const [showImport, setShowImport] = useState(false)
  const [importYaml, setImportYaml] = useState('')
  const [importError, setImportError] = useState<string | null>(null)

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

  const yaml = useMemo(() => serializeYaml(config), [config])
  const dagLevels = useMemo(() => layoutDAG(config.steps), [config.steps])
  const yamlLines = useMemo(() => yaml.split('\n'), [yaml])

  const handleSubmit = useCallback(async () => {
    if (!config.project.trim()) {
      setError('Project name is required')
      return
    }
    if (config.steps.length === 0) {
      setError('Add at least one step')
      return
    }
    for (const s of config.steps) {
      if (!s.prompt.trim()) {
        setError(`Step "${s.id}" has no prompt`)
        return
      }
    }

    setCreating(true)
    setError(null)
    try {
      const result = await createPipeline(yaml)
      setConfig({
        project: '',
        repoURL: '',
        repoBaseBranch: 'main',
        outputType: 'pr',
        branchPrefix: 'feat/',
        steps: [],
      })
      setNextStepNum(1)
      onCreated(result.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setCreating(false)
    }
  }, [config, yaml, onCreated])

  return (
    <div className="pipeline-builder">
      {/* Header */}
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
          <button
            className={`builder-yaml-btn${showYaml ? ' active' : ''}`}
            onClick={() => setShowYaml(!showYaml)}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="16 18 22 12 16 6" /><polyline points="8 6 2 12 8 18" />
            </svg>
            YAML Preview
            {!showYaml && <span className="section-count">{yamlLines.length - 1} lines</span>}
          </button>
          {error && <p className="error" style={{ margin: 0, fontSize: 11 }}>{error}</p>}
          <button className="btn" onClick={handleSubmit} disabled={creating} style={{ whiteSpace: 'nowrap' }}>
            {creating ? (
              <><span className="spinner" style={{ verticalAlign: 'middle', marginRight: 6 }} />Creating...</>
            ) : (
              <>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 6 }}>
                  <polygon points="5 3 19 12 5 21 5 3" />
                </svg>
                Run Pipeline
              </>
            )}
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="builder-content">
        {/* Project Settings */}
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
          </div>
        </div>

        {/* Pipeline Graph */}
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

        {/* YAML Preview (secondary) */}
        {showYaml && (
          <div className="builder-card">
            <div className="builder-card-title">YAML Preview</div>
            <pre className="builder-yaml-output">{yaml}</pre>
          </div>
        )}
      </div>

      {/* Import Modal */}
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

      {/* Step Modal */}
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

function StepModal({ step, allSteps, mode, onSave, onDelete, onClose }: {
  step: BuilderStep
  allSteps: BuilderStep[]
  mode: 'add' | 'edit'
  onSave: (s: BuilderStep) => void
  onDelete?: () => void
  onClose: () => void
}) {
  const [local, setLocal] = useState<BuilderStep>({ ...step })
  const [showPolicy, setShowPolicy] = useState(false)

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Escape') onClose()
  }

  function canSave() {
    return local.id.trim() && local.prompt.trim()
  }

  return (
    <div className="modal-overlay" onClick={onClose} onKeyDown={handleKey} tabIndex={0}>
      <div className="modal builder-modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 560 }}>
        <div className="builder-modal-header">
          <h3 style={{ margin: 0, fontSize: 15 }}>
            {mode === 'add' ? 'Add Step' : 'Edit Step'}
          </h3>
          <button className="builder-modal-close" onClick={onClose}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="builder-modal-body">
          <div className="builder-field">
            <label className="builder-label">Step ID</label>
            <input
              className="builder-input"
              type="text"
              value={local.id}
              onChange={e => setLocal(prev => ({ ...prev, id: e.target.value }))}
              placeholder="unique-step-id"
            />
          </div>

          <div className="builder-field">
            <label className="builder-label">Prompt</label>
            <textarea
              className="builder-textarea"
              rows={5}
              value={local.prompt}
              onChange={e => setLocal(prev => ({ ...prev, prompt: e.target.value }))}
              placeholder="Instructions sent to the agent..."
            />
          </div>

          <div className="builder-modal-row">
            <div className="builder-field">
              <label className="builder-label">Depends On</label>
              <div className="builder-depends-list">
                {allSteps
                  .filter(s => s.id !== local.id)
                  .map(s => (
                    <label key={s.id} className="builder-depends-chip">
                      <input
                        type="checkbox"
                        checked={local.dependsOn.includes(s.id)}
                        onChange={() => {
                          const next = local.dependsOn.includes(s.id)
                            ? local.dependsOn.filter(d => d !== s.id)
                            : [...local.dependsOn, s.id]
                          setLocal(prev => ({ ...prev, dependsOn: next }))
                        }}
                      />
                      {s.id}
                    </label>
                  ))}
                {allSteps.filter(s => s.id !== local.id).length === 0 && (
                  <span className="builder-depends-none">No other steps yet</span>
                )}
              </div>
            </div>
          </div>

          <div className="builder-modal-row">
            <div className="builder-field">
              <label className="builder-label">Validation Gates</label>
              <div className="builder-gate-list">
                {VALIDATION_GATES.map(g => (
                  <button
                    key={g}
                    className={`builder-gate-chip${local.validation.includes(g) ? ' active' : ''}`}
                    onClick={() => {
                      const next = local.validation.includes(g)
                        ? local.validation.filter(x => x !== g)
                        : [...local.validation, g]
                      setLocal(prev => ({ ...prev, validation: next }))
                    }}
                  >
                    {g === 'exit_zero' ? 'Exit Code' : g.charAt(0).toUpperCase() + g.slice(1)}
                  </button>
                ))}
              </div>
            </div>
            <div className="builder-field">
              <label className="builder-label">Approval</label>
              <div className="builder-approval-toggle">
                <button
                  className={`builder-toggle-btn${local.approval === 'optional' ? ' active' : ''}`}
                  onClick={() => setLocal(prev => ({ ...prev, approval: 'optional' }))}
                >
                  Optional
                </button>
                <button
                  className={`builder-toggle-btn${local.approval === 'required' ? ' active' : ''}`}
                  onClick={() => setLocal(prev => ({ ...prev, approval: 'required' }))}
                >
                  Required
                </button>
              </div>
            </div>
          </div>

          <div className="builder-policy-section">
            <button className="builder-policy-toggle" onClick={() => setShowPolicy(!showPolicy)}>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                style={{ transform: `rotate(${showPolicy ? 90 : 0}deg)`, transition: 'transform 0.15s' }}>
                <polyline points="9 18 15 12 9 6" />
              </svg>
              Policy & Constraints
            </button>
            {showPolicy && (
              <div className="builder-policy-body" style={{ marginTop: 10 }}>
                <div className="builder-policy-grid">
                  <div className="builder-field">
                    <label className="builder-label">Max Retries</label>
                    <input className="builder-input" type="number" min={0}
                      value={local.policy.maxRetries || ''}
                      onChange={e => setLocal(prev => ({ ...prev, policy: { ...prev.policy, maxRetries: parseInt(e.target.value) || 0 } }))} />
                  </div>
                  <div className="builder-field">
                    <label className="builder-label">Timeout (s)</label>
                    <input className="builder-input" type="number" min={0}
                      value={local.policy.timeoutSeconds || ''}
                      onChange={e => setLocal(prev => ({ ...prev, policy: { ...prev.policy, timeoutSeconds: parseInt(e.target.value) || 0 } }))} />
                  </div>
                  <div className="builder-field">
                    <label className="builder-label">Retry Delay (s)</label>
                    <input className="builder-input" type="number" min={0}
                      value={local.policy.retryDelaySeconds || ''}
                      onChange={e => setLocal(prev => ({ ...prev, policy: { ...prev.policy, retryDelaySeconds: parseInt(e.target.value) || 0 } }))} />
                  </div>
                  <div className="builder-field">
                    <label className="builder-label">Retry Backoff</label>
                    <select className="builder-select"
                      value={local.policy.retryBackoff}
                      onChange={e => setLocal(prev => ({ ...prev, policy: { ...prev.policy, retryBackoff: e.target.value } }))}>
                      <option value="linear">Linear</option>
                      <option value="exponential">Exponential</option>
                    </select>
                  </div>
                  <div className="builder-field builder-field-full">
                    <label className="builder-label">Allowed Directories</label>
                    <TagInput values={local.policy.allowedDirs}
                      onChange={dirs => setLocal(prev => ({ ...prev, policy: { ...prev.policy, allowedDirs: dirs } }))}
                      placeholder="e.g. src/" />
                  </div>
                  <div className="builder-field builder-field-full">
                    <label className="builder-label">Agent</label>
                    <input type="text" className="builder-input"
                      value={local.policy.agent}
                      onChange={e => setLocal(prev => ({ ...prev, policy: { ...prev.policy, agent: e.target.value } }))}
                      placeholder="e.g. agent-1 (empty = any agent)" />
                  </div>
                  <div className="builder-field builder-field-full">
                    <label className="builder-label">Allowed Tools</label>
                    <TagInput values={local.policy.allowedTools}
                      onChange={tools => setLocal(prev => ({ ...prev, policy: { ...prev.policy, allowedTools: tools } }))}
                      placeholder="e.g. write" />
                  </div>
                  <div className="builder-field builder-field-full">
                    <label className="builder-label">Allowed Commands</label>
                    <TagInput values={local.policy.allowedCommands}
                      onChange={cmds => setLocal(prev => ({ ...prev, policy: { ...prev.policy, allowedCommands: cmds } }))}
                      placeholder="e.g. npm test" />
                  </div>
                  <div className="builder-field">
                    <label className="builder-label">Save State</label>
                    <label className="builder-toggle-switch">
                      <input type="checkbox" checked={local.policy.saveState}
                        onChange={e => setLocal(prev => ({ ...prev, policy: { ...prev.policy, saveState: e.target.checked } }))} />
                      <span className="builder-toggle-track"><span className="builder-toggle-knob" /></span>
                    </label>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="builder-modal-actions">
          {onDelete && (
            <button className="btn-small btn-cancel-action" onClick={onDelete}>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 4 }}>
                <polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
              </svg>
              Remove Step
            </button>
          )}
          <div style={{ flex: 1 }} />
          <button className="btn-small" onClick={onClose}>Cancel</button>
          <button className="btn-small btn-approve" onClick={() => onSave(local)} disabled={!canSave()}
            style={{ opacity: canSave() ? 1 : 0.4, cursor: canSave() ? 'pointer' : 'not-allowed' }}>
            {mode === 'add' ? 'Add Step' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  )
}

function TagInput({ values, onChange, placeholder }: {
  values: string[]
  onChange: (v: string[]) => void
  placeholder: string
}) {
  const [input, setInput] = useState('')

  function addTag() {
    const trimmed = input.trim()
    if (trimmed && !values.includes(trimmed)) {
      onChange([...values, trimmed])
    }
    setInput('')
  }

  function removeTag(tag: string) {
    onChange(values.filter(v => v !== tag))
  }

  return (
    <div className="tag-input">
      <div className="tag-list">
        {values.map(tag => (
          <span key={tag} className="tag-item">
            {tag}
            <button className="tag-remove" onClick={() => removeTag(tag)}>
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </span>
        ))}
      </div>
      <div className="tag-input-row">
        <input className="builder-input tag-text-input" type="text" placeholder={placeholder}
          value={input} onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addTag() } }} />
        <button className="tag-add-btn" onClick={addTag} disabled={!input.trim()}>Add</button>
      </div>
    </div>
  )
}
