import { useEffect, useMemo, useRef, useState } from 'react'
import {
  fetchActivityEvents,
  fetchActivityStats,
  fetchSessionActivity,
  SessionsError,
  type ActivityEvent,
  type ActivityStats,
} from './api'
import { StatsBar } from './StatsBar'
import { EventCard } from './EventCard'
import { Filters, EMPTY_FILTERS, type ActivityFilters } from './Filters'
import { AgentHealth } from './AgentHealth'
import { AlertBanner } from './AlertBanner'

// ActivityPane — the live activity feed, the default tab of the unified
// Sessions view. With a `sessionId` it renders that session's own feed (server-
// scoped via /sessions/{id}/activity); without one it renders the global feed.
// Filters, stats, alerts and agent-health carry over from the former /activity
// page unchanged.
export function ActivityPane({
  sessionId,
  live,
  registerActivity,
}: {
  sessionId?: string | null
  live: boolean
  // The parent's single SSE stream calls this with its event handler; the pane
  // registers its live handler (which scopes by the current session) so the
  // parent never needs to re-subscribe when the selection changes.
  registerActivity: (h: ((e: ActivityEvent) => void) | null) => void
}) {
  const [events, setEvents] = useState<ActivityEvent[]>([])
  const [error, setError] = useState('')
  const [filters, setFilters] = useState<ActivityFilters>(EMPTY_FILTERS)
  const [dismissed, setDismissed] = useState<Set<number>>(new Set())
  const [stats, setStats] = useState<ActivityStats | null>(null)
  // Debounce the per-event stats refresh (review A10): coalesce to one refetch
  // per 5s so a burst of SSE events doesn't storm the stats endpoint.
  const statsTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const refreshStats = () => {
    if (statsTimer.current) return
    statsTimer.current = setTimeout(() => {
      statsTimer.current = null
      fetchActivityStats().then(setStats).catch(() => {})
    }, 5000)
  }

  // Initial load: events (scoped or global) + global stats. Re-runs when the
  // selected session changes.
  useEffect(() => {
    let cancelled = false
    setError('')
    const load = sessionId
      ? fetchSessionActivity(sessionId, 200)
      : fetchActivityEvents(200)
    Promise.all([load, fetchActivityStats()])
      .then(([ev, st]) => {
        if (cancelled) return
        setEvents(ev.events ?? [])
        setStats(st)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof SessionsError ? e.message : 'failed to load activity')
      })
    return () => {
      cancelled = true
      if (statsTimer.current) clearTimeout(statsTimer.current)
    }
  }, [sessionId])

  // Live activity events: the parent (SessionsPage) owns the single SSE
  // subscription and pushes events here, so there is exactly one EventSource on
  // /activity/stream. The ref always holds the freshest handler (current
  // sessionId); we register it with the parent once so the parent's stable
  // callback calls the latest handler without re-subscribing on selection.
  const onActivityRef = useRef<(e: ActivityEvent) => void>(() => {})
  onActivityRef.current = (e) => {
    if (sessionId && e.sessionId !== sessionId) return
    setEvents((prev) => {
      if (prev.some((p) => p.id === e.id)) return prev
      return [e, ...prev].slice(0, 200)
    })
    refreshStats()
  }
  useEffect(() => {
    registerActivity((e) => onActivityRef.current(e))
    return () => registerActivity(null)
  }, [registerActivity])

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
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-6">
      <div className="flex items-center gap-2 text-xs text-zinc-500">
        <span className={live ? 'text-emerald-500' : ''}>
          {live ? '● live' : '○ offline'}
        </span>
        <span>{sessionId ? 'session-scoped feed' : 'project-wide feed'}</span>
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
        <div
          className="flex flex-col"
          style={{ gap: 'var(--density-row)' }}
          role="log"
          aria-live="polite"
          aria-label="Live activity feed"
        >
          {filtered.map((e) => (
            <EventCard key={e.id} event={e} />
          ))}
        </div>
      )}
    </div>
  )
}
