import type { UnifiedTask } from './types'

const PRIORITY_BADGE: Record<string, string> = {
  high: 'text-danger border-danger/40',
  medium: 'text-warn border-warn/40',
  low: 'text-ink-5 border-ink-6/40',
}

export function TaskCard({
  task,
  onClick,
  dragging,
  milestoneTitles,
}: {
  task: UnifiedTask
  onClick: () => void
  dragging?: boolean
  milestoneTitles?: Record<string, string>
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={[
        'w-full rounded border border-edge bg-card p-3 text-left font-mono transition-colors',
        'hover:border-accent/50 hover:bg-edge/20',
        dragging ? 'opacity-40 ring-1 ring-accent' : '',
      ].join(' ')}
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-[12px] text-ink-5">{task.id}</span>
        {task.priority && (
          <span
            className={[
              'rounded border px-1.5 py-0.5 text-[10px] uppercase tracking-[0.06em]',
              PRIORITY_BADGE[task.priority] ?? 'text-ink-5 border-ink-6/40',
            ].join(' ')}
          >
            {task.priority}
          </span>
        )}
      </div>
      <p className="mt-1.5 line-clamp-2 text-[13px] leading-snug text-ink-2">{task.title}</p>
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {task.type && (
          <span className="rounded bg-edge/60 px-1.5 py-0.5 text-[10px] text-ink-4">
            {task.type}
          </span>
        )}
        {task.milestone && (
          <span className="rounded bg-accent/15 px-1.5 py-0.5 text-[10px] text-accent">
            {milestoneTitles?.[task.milestone] || task.milestone}
          </span>
        )}
        {task.labels?.slice(0, 3).map((l) => (
          <span
            key={l}
            className="rounded bg-edge/40 px-1.5 py-0.5 text-[10px] text-ink-5"
          >
            {l}
          </span>
        ))}
      </div>
      <div className="mt-2 flex items-center justify-between text-[11px] text-ink-5">
        <span>
          {task.assignees?.length ? task.assignees.join(', ') : 'unassigned'}
        </span>
        {task.ac_total ? (
          <span>
            AC {task.ac_completed ?? 0}/{task.ac_total}
          </span>
        ) : null}
      </div>
      {task.dependencies && task.dependencies.length > 0 && (
        <div className="mt-1.5 flex items-center gap-1 text-[10px] text-ink-6">
          <span>deps:</span>
          {task.dependencies.slice(0, 3).map((d) => (
            <span key={d}>
              {d}
            </span>
          ))}
          {task.dependencies.length > 3 && (
            <span>+{task.dependencies.length - 3}</span>
          )}
        </div>
      )}
    </button>
  )
}
