import type { ActivityEvent } from './api'

const SEVERITY_BORDER: Record<string, string> = {
  high: 'border-l-red-500',
  medium: 'border-l-amber-500',
  info: 'border-l-zinc-600',
}

const TYPE_ICON: Record<string, string> = {
  decision: '◆',
  architecture: '⬡',
  bugfix: '⚑',
  bug: '⚑',
  discovery: '◎',
  pattern: '✦',
  config: '⚙',
  learning: '✎',
  preference: '♡',
  convention: '§',
}

function timeAgo(ts: string): string {
  const t = new Date(ts).getTime()
  if (Number.isNaN(t)) return ts
  const s = Math.floor((Date.now() - t) / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return `${Math.floor(s / 86400)}d ago`
}

// EventCard — one compact activity row: type icon, timestamp, summary,
// severity left-border, source/actor badges.
export function EventCard({ event }: { event: ActivityEvent }) {
  const border = SEVERITY_BORDER[event.severity] ?? SEVERITY_BORDER.info
  const icon = TYPE_ICON[event.type] ?? '•'
  return (
    <div className={`rounded-md border border-edge ${border} border-l-4 bg-card px-3 py-2`}>
      <div className="flex items-start gap-2.5">
        <span className="mt-0.5 text-sm text-accent" aria-hidden>
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate text-sm font-medium text-zinc-100">{event.summary}</span>
            <span className="shrink-0 text-[11px] tabular-nums text-zinc-500">
              {timeAgo(event.ts)}
            </span>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[11px]">
            <span className="rounded bg-edge/60 px-1.5 py-0.5 text-zinc-300">{event.type}</span>
            {event.actor && (
              <span className="rounded bg-edge/40 px-1.5 py-0.5 text-zinc-400">
                {event.actor}
              </span>
            )}
            <span className="rounded bg-edge/40 px-1.5 py-0.5 text-zinc-500">
              {event.source}
            </span>
            {event.topicKey && (
              <span className="truncate text-zinc-600" title={event.topicKey}>
                {event.topicKey}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
