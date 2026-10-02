import type { PlanSummary } from './api'
import { chipKind, DOT_CLASS, KIND_ORDER, StatusChip } from './status'

// PlanDependencyGraph — a lightweight DAG of the plans by status. The backend
// surfaces no explicit dependency edges (the "Depends on:" field is free-text
// in the briefing markdown), so this renders the plan set as a status-ordered
// pipeline (active → blocked → pending → shipped) with progress — the
// structural overview the spec asks for, without inventing edges the backend
// doesn't provide. Each node is a button: clicking it selects that plan.
export function PlanDependencyGraph({
  plans,
  selected,
  onOpen,
}: {
  plans: PlanSummary[]
  selected?: string | null
  onOpen: (name: string) => void
}) {
  if (plans.length === 0) {
    return <p className="text-[13px] text-ink-5">No plans yet.</p>
  }
  const ordered = [...plans].sort(
    (a, b) => KIND_ORDER[chipKind(a.status)] - KIND_ORDER[chipKind(b.status)] || b.name.localeCompare(a.name),
  )
  return (
    <ol className="flex flex-col" data-testid="plan-pipeline">
      {ordered.map((p, i) => {
        const kind = chipKind(p.status)
        const active = selected === p.name
        const pct = Math.round(p.progress * 100)
        return (
          <li key={p.name} className="relative flex items-stretch gap-3">
            {i < ordered.length - 1 && (
              <span className="absolute left-[6px] top-5 h-full w-px bg-edge" aria-hidden />
            )}
            <span
              className={`relative z-10 mt-[13px] h-[13px] w-[13px] shrink-0 rounded-full border-2 ${DOT_CLASS[kind]}`}
              aria-hidden
            />
            <button
              type="button"
              onClick={() => onOpen(p.name)}
              aria-current={active ? 'true' : undefined}
              className={[
                'mb-1 flex min-w-0 flex-1 flex-col gap-1.5 rounded-[4px] border px-3 py-2 text-left font-mono text-[12px] transition-colors',
                active ? 'border-accent/60 bg-accent/10' : 'border-edge bg-card hover:bg-edge/40',
              ].join(' ')}
            >
              <span className="flex items-center justify-between gap-2">
                <span className={`truncate ${active ? 'text-accent' : 'text-ink-2'}`}>{p.name}</span>
                <StatusChip status={p.status} />
              </span>
              <span className="flex items-center gap-2">
                <span className="h-1 flex-1 overflow-hidden rounded-full bg-inset">
                  <span className="block h-full bg-accent" style={{ width: `${pct}%` }} />
                </span>
                <span className="shrink-0 tabular-nums text-[11px] text-ink-6">
                  {p.tasksTotal > 0 ? `${p.tasksDone}/${p.tasksTotal}` : '—'}
                </span>
              </span>
            </button>
          </li>
        )
      })}
    </ol>
  )
}
