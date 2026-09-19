import { useEffect, useMemo, useRef, useState } from 'react'
import {
  fetchActivityEvents,
  fetchActivityStats,
  openActivityStream,
  ActivityError,
  type ActivityEvent,
} from './api'
import { StatsBar } from './StatsBar'
import { EventCard } from './EventCard'
import { Filters, EMPTY_FILTERS, type ActivityFilters } from './Filters'
import { AgentHealth } from './AgentHealth'
import { AlertBanner } from './AlertBanner'

export function ActivityPage() {
  const [events, setEvents] = useState<ActivityEvent[]>([])
  const [error, setError] = useState('')
  const [live, setLive] = useState(false)
  const [filters, setFilters] = useState<ActivityFilters>(EMPTY_FILTERS)
  const [dismissed, setDismissed] = useState<Set<number>>(new Set())
  const [stats, setStats] = useState<Awaited<ReturnType<typeof fetchActivityStats>> | null>(null)
  const loadedRef = useRef(false)

  // Initial load: events + stats.
  useEffect(() => {
    let cancelled = false
    setError('')
    Promise.all([fetchActivityEvents(100), fetchActivityStats()])
      .then(([ev, st]) => {
        if (cancelled) return
        setEvents(ev.events ?? [])
        setStats(st)
        loadedRef.current = true
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof ActivityError ? e.message : 'failed to load activity')
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Live SSE stream: prepend new events, refresh stats, toggle live flag.
  useEffect(() => {
    const close = openActivityStream({
      onReady: () => setLive(true),
      onActivity: (e) => {
        setEvents((prev) => {
          // de-dup by id (the poller seeds from the newest id, so no overlap,
          // but be defensive)
          if (prev.some((p) => p.id === e.id)) return prev
          return [e, ...prev].slice(0, 200)
        })
        // lightweight stats refresh
        fetchActivityStats()
          .then(setStats)
          .catch(() => {})
      },
      onError: () => setLive(false),
    })
    return close
  }, [])

  const filtered = useMemo(() => {
    return events.filter((e) => {
      if (filters.type && e.type !== filters.type) return false
      if (filters.source && e.source !== filters.source) return false
      if (filters.severity && e.severity !== filters.severity) return false
      if (filters.actor && (e.actor ?? '') !== filters.actor) return false
      if (dismissed.has(e.id) && e.severity === 'high') return false
      return true
    })
  }, [events, filters, dismissed])

  const visibleAlerts = useMemo(
    () => events.filter((e) => e.severity === 'high' && !dismissed.has(e.id)),
    [events, dismissed],
  )

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-8">
      <div>
        <h1 className="text-2xl font-semibold text-zinc-100">Activity</h1>
        <p className="mt-1 text-sm text-zinc-500">
          Live agent event feed · {live ? 'streaming' : 'connected'}
        </p>
      </div>

      <StatsBar stats={stats} />

      <AlertBanner
        events={visibleAlerts}
        onDismiss={(id) => setDismissed((prev) => new Set(prev).add(id))}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <Filters events={events} value={filters} onChange={setFilters} />
        <AgentHealth events={events} stats={stats} live={live} />
      </div>

      {error ? (
        <div className="rounded-md border border-red-900/60 bg-red-950/30 p-4 text-sm text-red-300">
          {error}
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-md border border-edge bg-card p-8 text-center text-sm text-zinc-500">
          {events.length === 0 ? 'No activity recorded yet.' : 'No events match the current filters.'}
        </div>
      ) : (
        <div className="space-y-2">
          {filtered.map((e) => (
            <EventCard key={e.id} event={e} />
          ))}
        </div>
      )}
    </div>
  )
}
