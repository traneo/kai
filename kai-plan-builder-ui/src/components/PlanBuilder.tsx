import { useState, useCallback } from 'react'
import { SpecEditor } from './SpecEditor'
import { ChatPanel } from './ChatPanel'
import { chat, generatePipeline } from '../api'
import type { ChatMessage } from '../types'

interface Props {
  onContinue: (conversationId: string, spec: string, yaml: string) => void
}

export function PlanBuilder({ onContinue }: Props) {
  const [conversationId, setConversationId] = useState<string | null>(null)
  const [spec, setSpec] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [suggestContinue, setSuggestContinue] = useState(false)

  const handleSend = useCallback(async (message: string) => {
    setLoading(true)
    setError(null)
    setMessages(prev => [...prev, { role: 'user', content: message }])

    try {
      const res = await chat(conversationId, message)

      if (!conversationId) {
        setConversationId(res.conversation_id)
      }

      setMessages(prev => [...prev, { role: 'assistant', content: res.reply }])

      if (res.spec_updated) {
        setSpec(res.spec)
      }

      setSuggestContinue(res.suggest_continue)
    } catch (e) {
      const errMsg = e instanceof Error ? e.message : String(e)
      setError(errMsg)
      setMessages(prev => prev.slice(0, -1))
    } finally {
      setLoading(false)
    }
  }, [conversationId])

  const handleContinue = useCallback(async () => {
    if (!conversationId) {
      setError('No conversation started. Chat with the LLM first.')
      return
    }
    if (!spec.trim()) {
      setError('No spec to generate from. Build a spec through conversation first.')
      return
    }

    setGenerating(true)
    setError(null)

    try {
      const res = await generatePipeline(conversationId)
      onContinue(conversationId, spec, res.yaml)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setGenerating(false)
    }
  }, [conversationId, spec, onContinue])

  return (
    <>
      <div className="split-panel">
        <div className="panel-editor">
          <div className="panel-header">Project Spec</div>
          <div className="panel-body">
            <SpecEditor value={spec} onChange={setSpec} />
          </div>
        </div>
        <div className="panel-chat">
          <ChatPanel
            messages={messages}
            onSend={handleSend}
            loading={loading}
          />
        </div>
      </div>
      {generating && (
        <div className="modal-overlay">
          <div style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--radius-lg)',
            padding: 40,
            textAlign: 'center',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 16,
          }}>
            <span className="spinner" style={{ width: 24, height: 24, borderWidth: 3 }} />
            <span style={{ fontSize: 14, color: 'var(--text-secondary)' }}>
              Generating pipeline YAML...
            </span>
          </div>
        </div>
      )}
      {error && (
        <p className="error" style={{ marginTop: 8, paddingLeft: 4 }}>{error}</p>
      )}
      <div className="bottom-bar">
        <div style={{ flex: 1 }}>
          {suggestContinue && (
            <span style={{
              fontSize: 12,
              color: 'var(--accent-green)',
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
            }}>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
              Spec looks complete — ready to generate pipeline
            </span>
          )}
        </div>
        <button
          className="btn"
          onClick={handleContinue}
          disabled={generating || loading || !conversationId || !spec.trim()}
        >
          {generating ? (
            <><span className="spinner" /> Generating Pipeline...</>
          ) : (
            <>Continue &rarr;</>
          )}
        </button>
      </div>
    </>
  )
}
