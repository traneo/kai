import { useState, useEffect, useCallback } from 'react'
import type { PipelineDetail, StepDetail, ConversationEntry } from '../types'
import { approveStep, cancelPipeline, fetchPipelineYAML, fetchConversation, subscribeEvents, retryStep } from '../api'

interface Props {
  detail: PipelineDetail
  onUpdated: () => void
}

export function PipelineDetailView({ detail, onUpdated }: Props) {
  const [approvalStep, setApprovalStep] = useState<StepDetail | null>(null)
  const [cancelling, setCancelling] = useState(false)
  const [yamlCopied, setYamlCopied] = useState(false)
  const [activeStep, setActiveStep] = useState<string | null>(null)
  const [entries, setEntries] = useState<ConversationEntry[]>([])
  const [selectedStep, setSelectedStep] = useState<string | null>(null)
  const [detailTab, setDetailTab] = useState<'steps' | 'output'>('steps')

  const firstRunningOrPending = detail.steps.find(
    s => s.status === 'running' || s.status === 'pending'
  )

  const loadConversation = useCallback(async (stepId: string) => {
    setActiveStep(stepId)
    try {
      const data = await fetchConversation(detail.id, stepId)
      setEntries(data)
    } catch {
      setEntries([])
    }
  }, [detail.id])

  useEffect(() => {
    if (selectedStep) {
      loadConversation(selectedStep)
    }
  }, [selectedStep, loadConversation, detail])

  useEffect(() => {
    const unsub = subscribeEvents((evt: any) => {
      if (evt?.type === 'conversation' && evt.run_id === detail.id) {
        if (evt.step_id === selectedStep || (evt.step_id && !selectedStep)) {
          setEntries(prev => {
            const next = [...prev, {
              mission_id: evt.mission_id,
              run_id: evt.run_id,
              step_id: evt.step_id,
              sequence: evt.sequence,
              timestamp: new Date().toISOString(),
              source: evt.source,
              message: evt.message,
            }]
            if (next.length > 1000) return next.slice(-800)
            return next
          })
        }
      }
      onUpdated()
    })
    return unsub
  }, [selectedStep, detail.id, onUpdated])

  async function handleCancel() {
    if (!confirm('Cancel this pipeline? Running steps will be stopped.')) return
    setCancelling(true)
    try {
      await cancelPipeline(detail.id)
      onUpdated()
    } catch { /* ignore */ }
    finally { setCancelling(false) }
  }

  async function handleCopyYAML() {
    try {
      const yaml = await fetchPipelineYAML(detail.id)
      await navigator.clipboard.writeText(yaml)
      setYamlCopied(true)
      setTimeout(() => setYamlCopied(false), 2000)
    } catch { /* ignore */ }
  }

  async function handleApprove(message?: string) {
    if (!approvalStep) return
    try {
      await approveStep(detail.id, approvalStep.id, 'approve', message)
      setApprovalStep(null)
      onUpdated()
    } catch { /* ignore */ }
  }

  async function handleReject(message?: string) {
    if (!approvalStep) return
    try {
      await approveStep(detail.id, approvalStep.id, 'reject', message)
      setApprovalStep(null)
      onUpdated()
    } catch { /* ignore */ }
  }

  async function handleRetryStep(stepId: string) {
    try {
      await retryStep(detail.id, stepId)
      onUpdated()
    } catch { /* ignore */ }
  }

  const canCancel = (detail.status === 'running' || detail.status === 'blocked') && !cancelling

  return (
    <div className="detail-container">
      <div className="detail-meta">
        <span>Status: <strong className={`status-text ${detail.status}`}>{detail.status}</strong></span>
        {detail.status === 'completed' && detail.output_url && (
          <button className="btn-small btn-output" onClick={() => window.open(detail.output_url, '_blank')}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            View PR
          </button>
        )}
        {detail.status === 'completed' && detail.output_sha && (
          <span className="output-sha" title="Commit SHA">{detail.output_sha.slice(0, 7)}</span>
        )}
        <span>Created: {new Date(detail.created_at).toLocaleString()}</span>
        <span>Updated: {new Date(detail.updated_at).toLocaleString()}</span>
        {canCancel && <button className="btn-small btn-cancel-action" onClick={handleCancel}>Cancel</button>}
        <button className="btn-small" onClick={handleCopyYAML}>{yamlCopied ? 'Copied!' : 'Copy YAML'}</button>
      </div>

      {detail.error && <p className="error">{detail.error}</p>}

      <DAGView steps={detail.steps} currentStepId={firstRunningOrPending?.id}
        onStepClick={(s) => {
          if (s.status === 'blocked') setApprovalStep(s)
        }}
        onRetry={(s) => handleRetryStep(s.id)}
      />

      <div className="detail-tab-bar">
        <button
          className={`detail-tab${detailTab === 'steps' ? ' active' : ''}`}
          onClick={() => setDetailTab('steps')}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="8" y1="6" x2="21" y2="6" /><line x1="8" y1="12" x2="21" y2="12" />
            <line x1="8" y1="18" x2="21" y2="18" /><line x1="3" y1="6" x2="3.01" y2="6" />
            <line x1="3" y1="12" x2="3.01" y2="12" /><line x1="3" y1="18" x2="3.01" y2="18" />
          </svg>
          Steps
          <span className="detail-tab-count">{detail.steps.length}</span>
        </button>
        <button
          className={`detail-tab${detailTab === 'output' ? ' active' : ''}`}
          onClick={() => setDetailTab('output')}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="16 3 21 3 21 8" /><line x1="4" y1="20" x2="21" y2="3" />
            <polyline points="21 16 21 21 16 21" /><line x1="15" y1="15" x2="21" y2="21" />
            <line x1="4" y1="4" x2="9" y2="9" />
          </svg>
          Output
        </button>
      </div>

      {detailTab === 'steps' && (
        <div className="detail-steps">
          <div className="step-list">
            {detail.steps.map(s => (
              <StepCard
                key={s.id}
                step={s}
                onApprove={() => setApprovalStep(s)}
                onRetry={handleRetryStep}
                onSelect={() => setSelectedStep(s.id)}
                selected={selectedStep === s.id}
              />
            ))}
          </div>
        </div>
      )}

      {detailTab === 'output' && (
        <div className="detail-conversation">
          <div className="conv-header">
            <span className="conv-title">Agent Output</span>
            <div className="conv-step-selector">
              {detail.steps.map(s => (
                <button
                  key={s.id}
                  className={`conv-step-btn ${selectedStep === s.id ? 'active' : ''}`}
                  onClick={() => setSelectedStep(s.id)}
                >
                  {activeStep === s.id && entries.length === 0 && <span className="conv-dot" />}
                  {s.id}
                </button>
              ))}
            </div>
            {activeStep && <span className="conv-count">{entries.length} events</span>}
          </div>
          <div className="conv-content">
            {entries.length === 0 && activeStep && (
              <div className="conv-empty">Waiting for agent output...</div>
            )}
            {entries.length === 0 && !activeStep && (
              <div className="conv-empty">Select a step to view its output</div>
            )}
            {entries.map((entry, i) => (
              <div key={i} className={`conv-line ${entry.source === 'stderr' ? 'conv-error' : ''}`}>
                <span className="conv-source">{entry.source}</span>
                <span className="conv-msg">{entry.message}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {approvalStep && (
        <ApprovalDialog
          step={approvalStep}
          onApprove={handleApprove}
          onReject={handleReject}
          onClose={() => setApprovalStep(null)}
        />
      )}
    </div>
  )
}

function DAGView({ steps, onStepClick, onRetry, currentStepId }: {
  steps: StepDetail[]
  onStepClick: (s: StepDetail) => void
  onRetry?: (s: StepDetail) => void
  currentStepId?: string
}) {
  const levels = layoutDAG(steps)

  return (
    <div className="dag-container">
      <div className="dag-grid" style={{ gridTemplateColumns: `repeat(${levels.length}, 1fr)` }}>
        {levels.map((col, ci) => (
          <div key={ci} className="dag-column">
            {col.map((step) => {
              const isCurrent = step.id === currentStepId
              let cls = 'dag-node'
              if (step.status === 'passed' || step.status === 'completed') cls += ' passed'
              else if (step.status === 'failed') cls += ' failed'
              else if (step.status === 'running') cls += ' running'
              else if (step.status === 'blocked') cls += ' blocked clickable-dag'
              else cls += ' pending'
              if (isCurrent) cls += ' current'

              return (
                <div key={step.id} className={cls} onClick={() => {
                  if (step.status === 'blocked') onStepClick(step)
                }}>
                  <span className="dag-node-status">{step.status}</span>
                  <span className="dag-node-id">{step.id}</span>
                  {step.status === 'failed' && onRetry && (
                    <button className="dag-node-retry" onClick={(e) => { e.stopPropagation(); onRetry(step) }} title="Retry step">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3" /></svg>
                    </button>
                  )}
                </div>
              )
            })}
          </div>
        ))}
      </div>
    </div>
  )
}

function layoutDAG(steps: StepDetail[]): StepDetail[][] {
  const stepMap = new Map(steps.map(s => [s.id, s]))
  const depths = new Map<string, number>()

  function getDepth(id: string): number {
    if (depths.has(id)) return depths.get(id)!
    const s = stepMap.get(id)
    if (!s || !s.depends_on || s.depends_on.length === 0) {
      depths.set(id, 0)
      return 0
    }
    let maxDep = 0
    for (const dep of s.depends_on) {
      maxDep = Math.max(maxDep, getDepth(dep) + 1)
    }
    depths.set(id, maxDep)
    return maxDep
  }

  for (const s of steps) getDepth(s.id)
  const maxDepth = Math.max(...Array.from(depths.values()), 0)
  const levels: StepDetail[][] = Array.from({ length: maxDepth + 1 }, () => [])
  for (const s of steps) {
    levels[depths.get(s.id)!].push(s)
  }
  return levels
}

function StepCard({ step, onApprove, onRetry, onSelect, selected }: {
  step: StepDetail
  onApprove: () => void
  onRetry?: (id: string) => void
  onSelect: () => void
  selected: boolean
}) {
  const statusClass = step.status === 'passed' || step.status === 'completed' ? 'passed'
    : step.status === 'failed' ? 'failed'
    : step.status === 'running' ? 'running'
    : step.status === 'blocked' ? 'blocked' : 'pending'

  const gates = step.gate_results || []

  return (
    <div
      className={`step-card ${statusClass} ${selected ? 'selected' : ''}`}
      onClick={onSelect}
    >
      <div className="step-card-header">
        <div className="step-card-title">
          <span className={`step-status-dot ${statusClass}`} />
          <strong>{step.id}</strong>
          {step.assigned_to && <span className="step-assigned">@{step.assigned_to}</span>}
        </div>
        <span className={`status-badge ${statusClass}`}>
          <span className="status-dot" />
          {step.status}
        </span>
      </div>
      <div className="step-card-body">
        <p className="step-prompt">{step.prompt}</p>
        <div className="step-meta">
          {step.depends_on && step.depends_on.length > 0 && <span>Depends: {step.depends_on.join(', ')}</span>}
          {step.validation && step.validation.length > 0 && <span>Validation: {step.validation.join(', ')}</span>}
          {step.approval && step.approval !== 'optional' && <span>Approval: {step.approval}</span>}
          {step.retries > 0 && <span>Retries: {step.retries}</span>}
          {step.started_at && <span>Started: {new Date(step.started_at).toLocaleTimeString()}</span>}
        </div>

        {step.policy && (
          <div className="policy-tags">
            {step.policy.max_retries !== undefined && step.policy.max_retries > 0 && (
              <span className="policy-tag" title="Max retries">retries: {step.policy.max_retries}</span>
            )}
            {step.policy.timeout_seconds !== undefined && step.policy.timeout_seconds > 0 && (
              <span className="policy-tag" title="Timeout">{step.policy.timeout_seconds}s timeout</span>
            )}
            {step.policy.agent && (
              <span className="policy-tag" title="Assigned agent">agent: {step.policy.agent}</span>
            )}
            {step.policy.allowed_tools && step.policy.allowed_tools.length > 0 && (
              <span className="policy-tag" title="Allowed tools">tools: {step.policy.allowed_tools.join(', ')}</span>
            )}
            {step.policy.allowed_dirs && step.policy.allowed_dirs.length > 0 && (
              <span className="policy-tag" title="Allowed directories">dirs: {step.policy.allowed_dirs.join(', ')}</span>
            )}
            {step.policy.allowed_commands && step.policy.allowed_commands.length > 0 && (
              <span className="policy-tag" title="Allowed commands">cmds: {step.policy.allowed_commands.join(', ')}</span>
            )}
          </div>
        )}

        {gates.length > 0 && (
          <div className="gate-results">
            {gates.map(g => (
              <div key={g.gate} className={`gate-result ${g.status}`}>
                <span className="gate-icon">
                  {g.status === 'passed' ? (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                  ) : g.status === 'failed' ? (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
                  ) : (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" /></svg>
                  )}
                </span>
                <span className="gate-name">{g.gate}</span>
                {g.message && <span className="gate-message">{g.message}</span>}
                {g.duration && <span className="gate-duration">{g.duration}</span>}
              </div>
            ))}
          </div>
        )}

        {step.error && <p className="step-error">{step.error}</p>}
      </div>
      <div className="step-card-actions">
        {step.status === 'running' && (
          <span className="agent-spinner" style={{ width: 14, height: 14 }} />
        )}
        {step.status === 'blocked' && (
          <button className="btn-small btn-approve" onClick={(e) => { e.stopPropagation(); onApprove() }}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 20h9" /><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" /></svg>
            Review
          </button>
        )}
        {step.status === 'failed' && onRetry && (
          <button className="btn-small btn-retry" onClick={(e) => { e.stopPropagation(); onRetry(step.id) }}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="1 4 1 10 7 10" /><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10" /></svg>
            Retry
          </button>
        )}
      </div>
    </div>
  )
}

interface DiffLine {
  type: 'add' | 'del' | 'ctx' | 'hdr'
  text: string
  lineNum?: number
}

interface DiffFile {
  file: string
  lines: DiffLine[]
  added: number
  deleted: number
}

interface FileTreeNode {
  name: string
  path: string
  type: 'file' | 'dir'
  children: FileTreeNode[]
  added: number
  deleted: number
}

function buildFileTree(files: DiffFile[]): FileTreeNode {
  const root: FileTreeNode = { name: '', path: '', type: 'dir', children: [], added: 0, deleted: 0 }

  for (const f of files) {
    const parts = f.file.split('/')
    let node = root
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]
      const isFile = i === parts.length - 1
      let child = node.children.find(c => c.name === part)
      if (!child) {
        child = {
          name: part,
          path: isFile ? f.file : parts.slice(0, i + 1).join('/'),
          type: isFile ? 'file' : 'dir',
          children: [],
          added: 0,
          deleted: 0,
        }
        node.children.push(child)
      }
      child.added += f.added
      child.deleted += f.deleted
      node = child
    }
  }

  return root
}

function parseDiffFiles(diff: string): DiffFile[] {
  const files: DiffFile[] = []
  let current: DiffFile | null = null
  let oldLine = 0
  let newLine = 0

  for (const line of diff.split('\n')) {
    if (line.startsWith('diff --git a/')) {
      const m = line.match(/diff --git a\/(.*?) b\//)
      if (current) files.push(current)
      current = { file: m ? m[1] : line, lines: [], added: 0, deleted: 0 }
      current.lines.push({ type: 'hdr', text: line })
    } else if (!current) {
      continue
    } else if (line.startsWith('@@')) {
      const mh = line.match(/@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
      if (mh) newLine = parseInt(mh[1], 10)
      current.lines.push({ type: 'hdr', text: line })
    } else if (line.startsWith('+++ ') || line.startsWith('--- ') || line.startsWith('index ') || line.startsWith('new file') || line.startsWith('deleted file')) {
      current.lines.push({ type: 'hdr', text: line })
    } else if (line.startsWith('+')) {
      current.lines.push({ type: 'add', text: line, lineNum: newLine })
      current.added++
      newLine++
    } else if (line.startsWith('-')) {
      current.lines.push({ type: 'del', text: line })
      current.deleted++
    } else {
      current.lines.push({ type: 'ctx', text: line, lineNum: newLine })
      newLine++
    }
  }
  if (current) files.push(current)
  return files
}

function FileTreeItem({ node, depth, selectedFile, onSelect, expandedDirs, onToggle }: {
  node: FileTreeNode
  depth: number
  selectedFile: string | null
  onSelect: (path: string) => void
  expandedDirs: Set<string>
  onToggle: (path: string) => void
}) {
  const isExpanded = expandedDirs.has(node.path)

  function handleClick() {
    if (node.type === 'dir') {
      onToggle(node.path)
    } else {
      onSelect(node.path)
    }
  }

  const total = node.added + node.deleted

  return (
    <div>
      <div
        className={`approval-filetree-item ${node.type === 'file' && selectedFile === node.path ? 'selected' : ''}`}
        style={{ paddingLeft: 12 + depth * 16 }}
        onClick={handleClick}
      >
        {node.type === 'dir' ? (
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0, marginRight: 4 }}>
            {isExpanded
              ? <><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l3 3h7a2 2 0 0 1 2 2v1" /><path d="M5 15h14l-3 3 3 3" /></>
              : <><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l3 3h7a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2H5z" /></>
            }
          </svg>
        ) : (
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0, marginRight: 4 }}>
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><polyline points="14 2 14 8 20 8" />
          </svg>
        )}
        <span className="approval-filetree-name">{node.name}</span>
        <span className="approval-filetree-stats">
          {node.added > 0 && <span className="stat-added">+{node.added}</span>}
          {node.deleted > 0 && <span className="stat-deleted">-{node.deleted}</span>}
          <span className="stat-total">{total}</span>
        </span>
      </div>
      {node.type === 'dir' && isExpanded && (
        <div>
          {node.children.map(child => (
            <FileTreeItem
              key={child.path}
              node={child}
              depth={depth + 1}
              selectedFile={selectedFile}
              onSelect={onSelect}
              expandedDirs={expandedDirs}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ApprovalDialog({ step, onApprove, onReject, onClose }: {
  step: StepDetail
  onApprove: (msg?: string) => void
  onReject: (msg?: string) => void
  onClose: () => void
}) {
  const [message, setMessage] = useState('')
  const [selectedFile, setSelectedFile] = useState<string | null>(null)

  const diffFiles = step.diff ? parseDiffFiles(step.diff) : []
  const gates = step.gate_results || []

  const fileTree = diffFiles.length > 0 ? buildFileTree(diffFiles) : null

  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(() => {
    if (!fileTree) return new Set()
    const s = new Set<string>()
    function collectDirs(n: FileTreeNode) {
      if (n.type === 'dir') {
        s.add(n.path)
        n.children.forEach(collectDirs)
      }
    }
    fileTree.children.forEach(collectDirs)
    return s
  })

  function toggleDir(path: string) {
    setExpandedDirs(prev => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })
  }

  useEffect(() => {
    if (!selectedFile && diffFiles.length > 0) {
      setSelectedFile(diffFiles[0].file)
    }
  }, [selectedFile, diffFiles])

  const activeFile = diffFiles.find(f => f.file === selectedFile)

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Escape') onClose()
  }

  function gateIcon(status: string) {
    if (status === 'passed') {
      return <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
    }
    if (status === 'failed') {
      return <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
    }
    return <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" /></svg>
  }

  return (
    <div className="modal-overlay" onClick={onClose} onKeyDown={handleKey} tabIndex={0}>
      <div className="approval-modal-inner" onClick={e => e.stopPropagation()}>
        {/* Top bar: header + badges */}
        <div className="approval-topbar">
          <div className="approval-topbar-left">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--accent-yellow)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 9v4" /><path d="M12 17h.01" />
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            </svg>
            <span className="approval-topbar-step">Review: <strong>{step.id}</strong></span>
            <span className="approval-topbar-prompt">{step.prompt}</span>
          </div>
          <div className="approval-topbar-badges">
            {gates.map(g => (
              <span key={g.gate} className={`gate-badge ${g.status}`}>
                {gateIcon(g.status)}
                {g.gate}
              </span>
            ))}
          </div>
        </div>

        {/* Body: file tree + diff */}
        <div className="approval-body">
          <div className="approval-filetree">
            <div className="approval-filetree-title">Files</div>
            <div className="approval-filetree-list">
              {fileTree && fileTree.children.map(child => (
                <FileTreeItem
                  key={child.path}
                  node={child}
                  depth={0}
                  selectedFile={selectedFile}
                  onSelect={setSelectedFile}
                  expandedDirs={expandedDirs}
                  onToggle={toggleDir}
                />
              ))}
              {!fileTree && <div className="approval-filetree-empty">No changes</div>}
            </div>
          </div>

          <div className="approval-diff">
            {activeFile ? (
              <div className="diff-view">
                <div className="diff-view-header">{activeFile.file}</div>
                <div className="diff-view-content">
                  {activeFile.lines.map((l, li) => {
                    if (l.type === 'hdr') return null
                    return (
                      <div key={li} className={`diff-line diff-${l.type}`}>
                        {l.lineNum !== undefined && <span className="diff-ln">{l.lineNum}</span>}
                        {l.lineNum === undefined && <span className="diff-ln diff-ln-empty" />}
                        {l.type !== 'ctx' && <span className="diff-prefix">{l.text[0]}</span>}
                        {l.type === 'ctx' && <span className="diff-prefix diff-prefix-ctx" />}
                        <span className="diff-text">{l.type === 'ctx' || l.type === 'add' || l.type === 'del' ? l.text.slice(1) : l.text}</span>
                      </div>
                    )
                  })}
                </div>
              </div>
            ) : (
              <div className="diff-empty">No file selected</div>
            )}
          </div>
        </div>

        {/* Bottom bar: message + actions */}
        <div className="approval-bottombar">
          <textarea
            className="approval-msg"
            placeholder="Optional review message..."
            value={message}
            onChange={e => setMessage(e.target.value)}
            rows={1}
          />
          <div className="approval-actions">
            <button className="btn btn-approve-action" onClick={() => onApprove(message)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 4 }}><polyline points="20 6 9 17 4 12" /></svg>
              Approve
            </button>
            <button className="btn btn-reject" onClick={() => onReject(message)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 4 }}><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
              Reject
            </button>
            <button className="btn btn-cancel" onClick={onClose}>Cancel</button>
          </div>
        </div>
      </div>
    </div>
  )
}
