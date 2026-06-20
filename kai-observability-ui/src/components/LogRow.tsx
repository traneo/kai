import type { LogEntry } from '../types'
import { formatTime, levelClass } from '../utils'

interface Props {
  entry: LogEntry
  onClick?: (entry: LogEntry) => void
}

export function LogRow({ entry, onClick }: Props) {
  return (
    <tr
      className={`log-row ${levelClass(entry.level)}`}
      onClick={() => onClick?.(entry)}
    >
      <td className="log-cell-time">{formatTime(entry.timestamp)}</td>
      <td className="log-cell-level">
        <span className={`level-badge ${levelClass(entry.level)}`}>
          {entry.level}
        </span>
      </td>
      <td className="log-cell-service">{entry.service}</td>
      <td className="log-cell-msg">{entry.message}</td>
      {entry.run_id && <td className="log-cell-run">{entry.run_id}</td>}
    </tr>
  )
}
