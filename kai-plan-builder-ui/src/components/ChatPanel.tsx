import { useState, useRef, useEffect } from 'react'
import type { ChatMessage } from '../types'

interface Props {
  messages: ChatMessage[]
  onSend: (message: string) => Promise<void>
  loading: boolean
}

export function ChatPanel({ messages, onSend, loading }: Props) {
  const [input, setInput] = useState('')
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  async function handleSend() {
    const msg = input.trim()
    if (!msg || loading) return
    setInput('')
    await onSend(msg)
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <>
      <div className="panel-header">LLM Chat</div>
      <div className="chat-messages">
        {messages.length === 0 && (
          <div className="chat-empty">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
            </svg>
            <span>Chat with the LLM to build your project spec. It can edit the spec on the left.</span>
          </div>
        )}
        {messages.map((msg, i) => (
          <div key={i} className={`chat-message ${msg.role}`}>
            {msg.content}
          </div>
        ))}
        {loading && (
          <div className="chat-message assistant loading">
            <span className="spinner" style={{ verticalAlign: 'middle', marginRight: 6 }} />
            Thinking...
          </div>
        )}
        <div ref={endRef} />
      </div>
      <div className="chat-input-area">
        <input
          className="chat-input"
          type="text"
          placeholder="Describe your pipeline..."
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKey}
          disabled={loading}
        />
        <button
          className="chat-send-btn"
          onClick={handleSend}
          disabled={!input.trim() || loading}
        >
          Send
        </button>
      </div>
    </>
  )
}
