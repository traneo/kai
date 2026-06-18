import { useState, useEffect, useCallback } from 'react'
import type { PipelineDetail, StepDetail, ConversationEntry } from '../types'
import { approveStep, cancelPipeline, fetchPipelineYAML, fetchConversation, subscribeEvents } from '../api'

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

      <DAGView steps={detail.steps} onStepClick={(s) => {
        if (s.status === 'blocked') setApprovalStep(s)
      }} currentStepId={firstRunningOrPending?.id} />

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

function DAGView({ steps, onStepClick, currentStepId }: {
  steps: StepDetail[]
  onStepClick: (s: StepDetail) => void
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

function StepCard({ step, onApprove, onSelect, selected }: {
  step: StepDetail
  onApprove: () => void
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
      </div>
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

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Escape') onClose()
  }

  return (
    <div className="modal-overlay" onClick={onClose} onKeyDown={handleKey} tabIndex={0}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--accent-yellow)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 9v4" /><path d="M12 17h.01" />
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          </svg>
          <h3 style={{ margin: 0 }}>Review Step: {step.id}</h3>
        </div>
        <p className="step-prompt">{step.prompt}</p>
        <textarea
          className="yaml-input"
          placeholder="Optional review message..."
          value={message}
          onChange={e => setMessage(e.target.value)}
          rows={3}
        />
        <div className="modal-actions">
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
  )
}
