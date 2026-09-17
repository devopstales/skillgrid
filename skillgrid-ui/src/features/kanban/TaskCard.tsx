import type { UnifiedTask } from './types'

const PRIORITY_BADGE: Record<string, string> = {
  high: 'text-red-400 border-red-500/40',
  medium: 'text-amber-400 border-amber-500/40',
  low: 'text-zinc-500 border-zinc-600/40',
}

export function TaskCard({
  task,
  onClick,
  dragging,
}: {
  task: UnifiedTask
  onClick: () => void
  dragging?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={[
        'w-full rounded-md border border-edge bg-card p-3 text-left transition-colors',
        'hover:border-accent/50 hover:bg-edge/30',
        dragging ? 'opacity-40 ring-1 ring-accent' : '',
      ].join(' ')}
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-xs font-mono text-zinc-500">{task.id}</span>
        {task.priority && (
          <span
            className={[
              'rounded border px-1.5 py-0.5 text-[10px] uppercase',
              PRIORITY_BADGE[task.priority] ?? 'text-zinc-500 border-zinc-600/40',
            ].join(' ')}
          >
            {task.priority}
          </span>
        )}
      </div>
      <p className="mt-1.5 line-clamp-2 text-sm text-zinc-200">{task.title}</p>
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {task.type && (
          <span className="rounded bg-edge/50 px-1.5 py-0.5 text-[10px] text-zinc-400">
            {task.type}
          </span>
        )}
        {task.milestone && (
          <span className="rounded bg-accent/15 px-1.5 py-0.5 text-[10px] text-accent">
            {task.milestone}
          </span>
        )}
        {task.labels?.slice(0, 3).map((l) => (
          <span
            key={l}
            className="rounded bg-edge/40 px-1.5 py-0.5 text-[10px] text-zinc-500"
          >
            {l}
          </span>
        ))}
      </div>
      <div className="mt-2 flex items-center justify-between text-[11px] text-zinc-500">
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
        <div className="mt-1.5 flex items-center gap-1 text-[10px] text-zinc-600">
          <span>deps:</span>
          {task.dependencies.slice(0, 3).map((d) => (
            <span key={d} className="font-mono">
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
