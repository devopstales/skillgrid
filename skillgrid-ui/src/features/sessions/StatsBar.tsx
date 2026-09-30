import type { ActivityStats } from './api'

// StatsBar — the activity counters (total + per-type + active sessions).
export function StatsBar({ stats }: { stats: ActivityStats | null }) {
  if (!stats) return null
  const types = Object.entries(stats.byType).sort((a, b) => b[1] - a[1])
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Stat label="events" value={stats.total} />
      <Stat label="active sessions" value={stats.activeSessions} />
      {types.map(([type, n]) => (
        <Stat key={type} label={type} value={n} subtle />
      ))}
    </div>
  )
}

function Stat({ label, value, subtle }: { label: string; value: number; subtle?: boolean }) {
  return (
    <div
      className={
        'rounded border border-edge px-3 py-1.5 font-mono text-[12px] ' +
        (subtle ? 'bg-card text-ink-4' : 'bg-accent/10 text-accent')
      }
    >
      <span className="font-semibold tabular-nums">{value}</span>{' '}
      <span className="uppercase tracking-[0.1em] text-ink-6">{label}</span>
    </div>
  )
}
