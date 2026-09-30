import type { ActivityEvent, ActivityStats } from './api'

// AgentHealth — a lightweight "is the agent alive?" panel derived from the
// feed: last-event age + active session count. No backend health endpoint, so
// this is an approximation from the activity stream.
export function AgentHealth({
  events,
  stats,
  live,
}: {
  events: ActivityEvent[]
  stats: ActivityStats | null
  live: boolean
}) {
  const lastTs = events[0]?.ts
  const age = lastTs ? Math.floor((Date.now() - new Date(lastTs).getTime()) / 1000) : null
  // Heuristic: "active" if a live stream is open AND there's recent activity,
  // or active sessions > 0.
  const active = live || (stats?.activeSessions ?? 0) > 0 || (age !== null && age < 300)

  const dot = active ? 'bg-accent' : 'bg-ink-6'
  const label = active ? (age !== null && age < 60 ? 'active now' : 'recently active') : 'idle'

  return (
    <div className="flex items-center gap-2 rounded border border-edge bg-card px-3 py-2 font-mono text-[12px]">
      <span className={`h-2 w-2 rounded-full ${dot}`} aria-hidden />
      <span className="font-medium text-ink-2">Agent</span>
      <span className="text-ink-5">{label}</span>
      {age !== null && (
        <span className="text-ink-6">· last event {age}s ago</span>
      )}
      {stats && (
        <span className="ml-auto text-ink-5">
          {stats.activeSessions} active session{stats.activeSessions === 1 ? '' : 's'}
        </span>
      )}
    </div>
  )
}
