import type { Page } from '../types'

interface Props {
  current: Page
  onNavigate: (page: Page) => void
}

const navItems: { page: Page; label: string; icon: string }[] = [
  { page: 'dashboard', label: 'Dashboard', icon: '⊞' },
  { page: 'live', label: 'Live Stream', icon: '▶' },
  { page: 'logs', label: 'Logs', icon: '☰' },
  { page: 'runs', label: 'Runs', icon: '⏵' },
  { page: 'errors', label: 'Errors', icon: '⚠' },
  { page: 'analytics', label: 'Analytics', icon: '⬆' },
]

export function NavBar({ current, onNavigate }: Props) {
  return (
    <nav className="navbar">
      <div className="navbar-brand">
        <span className="navbar-logo">KAI</span>
        <span className="navbar-subtitle">Observability</span>
      </div>
      <div className="navbar-items">
        {navItems.map(({ page, label, icon }) => (
          <button
            key={page}
            className={`navbar-item${current === page ? ' active' : ''}`}
            onClick={() => onNavigate(page)}
          >
            <span className="navbar-icon">{icon}</span>
            <span>{label}</span>
          </button>
        ))}
      </div>
    </nav>
  )
}
