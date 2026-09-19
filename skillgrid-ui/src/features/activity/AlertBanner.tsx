import type { ActivityEvent } from './api'

// AlertBanner — surfaces the most recent high-severity (bugfix/bug) events as
// a dismissible banner above the feed. Returns null when there are none.
export function AlertBanner({
  events,
  onDismiss,
}: {
  events: ActivityEvent[]
  onDismiss: (id: number) => void
}) {
  const recent = events.filter((e) => e.severity === 'high').slice(0, 3)
  if (recent.length === 0) return null
  return (
    <div className="space-y-1.5">
      {recent.map((e) => (
        <div
          key={e.id}
          className="flex items-center gap-2 rounded-md border border-red-900/60 bg-red-950/40 px-3 py-2 text-xs"
        >
          <span className="text-red-400" aria-hidden>
            ⚑
          </span>
          <span className="min-w-0 flex-1 truncate text-red-200">{e.summary}</span>
          <span className="shrink-0 text-red-400/70">{e.type}</span>
          <button
            type="button"
            onClick={() => onDismiss(e.id)}
            className="shrink-0 text-red-400/60 hover:text-red-200"
            aria-label="dismiss alert"
          >
            ✕
          </button>
        </div>
      ))}
    </div>
  )
}
