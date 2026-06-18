import type { Page } from '../types'

interface Props {
  current: Page
  onNavigate: (page: Page) => void
}

const tabs: { page: Page; label: string }[] = [
  { page: 'dashboard', label: 'Dashboard' },
  { page: 'secrets', label: 'Secrets' },
  { page: 'platform-config', label: 'Config' },
  { page: 'audit', label: 'Audit Log' },
  { page: 'new', label: 'New Pipeline' },
]

export function NavBar({ current, onNavigate }: Props) {
  return (
    <nav className="nav-tabs">
      {tabs.map(t => (
        <button
          key={t.page}
          className={`nav-tab${current === t.page ? ' active' : ''}`}
          onClick={() => onNavigate(t.page)}
        >
          {t.label}
        </button>
      ))}
    </nav>
  )
}
