import type { LogLevel } from './types'

export function formatTime(ts: number): string {
  const d = new Date(ts)
  return d.toLocaleTimeString('en-US', { hour12: false })
}

export function formatDuration(startMs: number, endMs: number): string {
  const ms = endMs - startMs
  if (ms <= 0) return '0s'
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ${s % 60}s`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

export function levelClass(level: LogLevel): string {
  switch (level) {
    case 'error': return 'level-error'
    case 'warn': return 'level-warn'
    case 'info': return 'level-info'
    case 'debug': return 'level-debug'
  }
}
