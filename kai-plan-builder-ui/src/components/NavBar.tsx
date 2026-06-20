interface Props {
  current: string
  onGoBack?: () => void
}

export function NavBar({ current, onGoBack }: Props) {
  return (
    <nav className="nav-tabs" style={{ display: 'flex', gap: 8, marginLeft: 16, alignItems: 'center' }}>
      <span style={{
        fontSize: 11,
        color: 'var(--text-muted)',
        textTransform: 'uppercase',
        letterSpacing: 0.5,
        background: 'var(--bg-elevated)',
        padding: '3px 8px',
        borderRadius: 'var(--radius-sm)',
      }}>
        {current === 'builder' ? 'Spec Builder' : 'Pipeline Review'}
      </span>
      {onGoBack && (
        <button
          className="btn btn-secondary"
          onClick={onGoBack}
          style={{ padding: '4px 12px', fontSize: 12 }}
        >
          &larr; Back
        </button>
      )}
    </nav>
  )
}
