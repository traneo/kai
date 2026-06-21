import { useState } from 'react'
import type { BuilderStep } from '../types'
import { VALIDATION_GATES } from '../types'
import { TagInput } from './TagInput'

interface Props {
  step: BuilderStep
  allSteps: BuilderStep[]
  mode: 'add' | 'edit'
  onSave: (s: BuilderStep) => void
  onDelete?: () => void
  onClose: () => void
}

export function StepModal({ step, allSteps, mode, onSave, onDelete, onClose }: Props) {
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
