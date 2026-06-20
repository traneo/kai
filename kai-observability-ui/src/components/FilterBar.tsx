import { useState, useEffect } from 'react'
import type { LogLevel, QueryFilter } from '../types'

interface Props {
  filter: QueryFilter
  onChange: (filter: QueryFilter) => void
}

const levels: LogLevel[] = ['info', 'warn', 'error', 'debug']

export function FilterBar({ filter, onChange }: Props) {
  const [search, setSearch] = useState(filter.search || '')
  const [debounce, setDebounce] = useState<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    setSearch(filter.search || '')
  }, [filter.search])

  function update(part: Partial<QueryFilter>) {
    onChange({ ...filter, ...part })
  }

  function handleSearch(val: string) {
    setSearch(val)
    if (debounce) clearTimeout(debounce)
    setDebounce(setTimeout(() => update({ search: val || undefined }), 300))
  }

  return (
    <div className="filter-bar">
      <input
        className="filter-input"
        placeholder="Search messages..."
        value={search}
        onChange={(e) => handleSearch(e.target.value)}
      />
      <select
        className="filter-select"
        value={filter.service || ''}
        onChange={(e) => update({ service: e.target.value || undefined })}
      >
        <option value="">All Services</option>
        <option value="kai-cli">kai-cli</option>
        <option value="kai-code">kai-code</option>
        <option value="kai-platform">kai-platform</option>
        <option value="kai-config">kai-config</option>
      </select>
      <select
        className="filter-select"
        value={filter.level || ''}
        onChange={(e) => update({ level: (e.target.value || undefined) as LogLevel | undefined })}
      >
        <option value="">All Levels</option>
        {levels.map((l) => (
          <option key={l} value={l}>{l}</option>
        ))}
      </select>
      <input
        className="filter-input filter-input-narrow"
        type="text"
        placeholder="run_id"
        value={filter.run_id || ''}
        onChange={(e) => update({ run_id: e.target.value || undefined })}
      />
      <input
        className="filter-input filter-input-narrow"
        type="text"
        placeholder="step_id"
        value={filter.step_id || ''}
        onChange={(e) => update({ step_id: e.target.value || undefined })}
      />
      <input
        className="filter-input filter-input-narrow"
        type="text"
        placeholder="agent_id"
        value={filter.agent_id || ''}
        onChange={(e) => update({ agent_id: e.target.value || undefined })}
      />
    </div>
  )
}
