import type { PlanSummary } from './api'

const STATUS_ORDER: Record<string, number> = {
  done: 0,
  in_progress: 1,
  planning: 2,
  draft: 3,
  paused: 4,
  archived: 5,
}

const STATUS_COLOR: Record<string, string> = {
  done: 'border-accent-ink bg-accent/15 text-accent',
  in_progress: 'border-warn-ink bg-warn/10 text-warn',
  planning: 'border-info/50 bg-surface-2/60 text-info',
  draft: 'border-edge-soft bg-surface-2/60 text-ink-3',
  paused: 'border-edge-soft bg-surface-2/60 text-ink-5',
}

// PlanDependencyGraph — a lightweight DAG of the plans by status. The backend
// surfaces no explicit dependency edges (the "Depends on:" field is free-text
// in the briefing markdown), so this renders the plan set as a status-ordered
// pipeline (done → in-progress → planning → draft) with progress — the
// structural overview the spec asks for, without inventing edges the backend
// doesn't provide.
export function PlanDependencyGraph({ plans }: { plans: PlanSummary[] }) {
  if (plans.length === 0) {
    return <p className="text-[13px] text-ink-5">No plans yet.</p>
  }
  const ordered = [...plans].sort(
    (a, b) =>
      (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9) ||
      b.name.localeCompare(a.name),
  )
  return (
    <div className="flex flex-col gap-2">
      {ordered.map((p, i) => {
        const color = STATUS_COLOR[p.status] ?? 'border-edge-soft bg-surface-2/60 text-ink-4'
        return (
          <div key={p.name} className="relative flex items-center gap-3">
            {i < ordered.length - 1 && (
              <span className="absolute left-[7px] top-6 h-full w-px bg-edge" aria-hidden />
            )}
            <span
              className={`relative z-10 h-3.5 w-3.5 shrink-0 rounded-full border-2 ${color.split(' ')[0]} ${color.split(' ')[1]}`}
              aria-hidden
            />
            <div className={`flex-1 rounded border px-3 py-2 text-[12px] ${color}`}>
              <div className="flex items-center justify-between gap-2">
                <span className="truncate font-medium">{p.name}</span>
                <span className="shrink-0 opacity-70">{p.status}</span>
              </div>
              <div className="mt-1 h-1 w-full overflow-hidden rounded-full bg-black/30">
                <div className="h-full bg-current opacity-60" style={{ width: `${Math.round(p.progress * 100)}%` }} />
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
