import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'

interface RefreshContextValue {
  refreshInterval: number
  refreshEnabled: boolean
  refreshTick: number
  setRefreshInterval: (ms: number) => void
  setRefreshEnabled: (enabled: boolean) => void
}

const INTERVALS = [1000, 5000, 10000, 15000, 30000] as const

const RefreshContext = createContext<RefreshContextValue | null>(null)

export function RefreshProvider({ children }: { children: ReactNode }) {
  const [refreshInterval, setRefreshInterval] = useState(10000)
  const [refreshEnabled, setRefreshEnabled] = useState(true)
  const [refreshTick, setRefreshTick] = useState(0)

  useEffect(() => {
    if (!refreshEnabled) return
    const id = setInterval(() => {
      setRefreshTick((t) => t + 1)
    }, refreshInterval)
    return () => clearInterval(id)
  }, [refreshInterval, refreshEnabled])

  const setIntervalSafe = useCallback((ms: number) => {
    const clamped = INTERVALS.includes(ms as typeof INTERVALS[number]) ? ms : 10000
    setRefreshInterval(clamped)
  }, [])

  return (
    <RefreshContext.Provider
      value={{
        refreshInterval,
        refreshEnabled,
        refreshTick,
        setRefreshInterval: setIntervalSafe,
        setRefreshEnabled,
      }}
    >
      {children}
    </RefreshContext.Provider>
  )
}

export function useRefresh() {
  const ctx = useContext(RefreshContext)
  if (!ctx) throw new Error('useRefresh must be used within RefreshProvider')
  return ctx
}

export { INTERVALS }
