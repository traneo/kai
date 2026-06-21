import { useState } from 'react'

interface Props {
  values: string[]
  onChange: (v: string[]) => void
  placeholder: string
}

export function TagInput({ values, onChange, placeholder }: Props) {
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
