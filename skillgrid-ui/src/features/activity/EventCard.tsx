import { memo } from 'react'
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
// severity left-border, source/actor badges. Memoized (review A10): the parent
// re-renders on every SSE event, but only the new row's `event` identity
// changes, so unchanged rows skip re-render (O(1) instead of O(200) per event).
export const EventCard = memo(function EventCard({ event }: { event: ActivityEvent }) {
  const border = SEVERITY_BORDER[event.severity] ?? SEVERITY_BORDER.info
  const icon = TYPE_ICON[event.type] ?? '•'
  return (
    // animate-stream-in: SSE-prepended items fade/slide in from the top (7.4).
    <div className={`animate-stream-in rounded-md border border-edge ${border} border-l-4 bg-card px-3 py-2`}>
      <div className="flex items-start gap-2.5">
        <span className="mt-0.5 text-sm text-accent" aria-hidden>
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate text-sm font-medium text-zinc-100">{event.summary}</span>
            {/* review B5: zinc-400 (was zinc-500) for ≥4.5:1 contrast at 11px */}
            <span className="shrink-0 text-[11px] tabular-nums text-zinc-400">
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
            <span className="rounded bg-edge/40 px-1.5 py-0.5 text-zinc-400">
              {event.source}
            </span>
            {/* review B5: zinc-500 (was zinc-600, ~4.0:1) for ≥4.5:1 contrast */}
            {event.topicKey && (
              <span className="truncate text-zinc-500" title={event.topicKey}>
                {event.topicKey}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
})
