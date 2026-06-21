import { useRefresh, INTERVALS } from '../RefreshContext'

const LABELS: Record<number, string> = {
  1000: '1s',
  5000: '5s',
  10000: '10s',
  15000: '15s',
  30000: '30s',
}

export function RefreshBar() {
  const { refreshInterval, refreshEnabled, setRefreshInterval, setRefreshEnabled } = useRefresh()

  return (
    <div className="refresh-bar">
      <span className="refresh-bar-label">
        Auto-refresh:
      </span>
      <select
        className="refresh-bar-select"
        value={refreshInterval}
        onChange={(e) => setRefreshInterval(Number(e.target.value))}
      >
        {INTERVALS.map((ms) => (
          <option key={ms} value={ms}>
            {LABELS[ms]}
          </option>
        ))}
      </select>
      <span className="refresh-bar-status">
        {refreshEnabled ? '●' : '○'}
      </span>
      <button
        className="refresh-bar-toggle"
        onClick={() => setRefreshEnabled(!refreshEnabled)}
        title={refreshEnabled ? 'Pause auto-refresh' : 'Resume auto-refresh'}
      >
        {refreshEnabled ? '⏸' : '▶'}
      </button>
    </div>
  )
}
