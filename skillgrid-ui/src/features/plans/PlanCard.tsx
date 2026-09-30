import type { PlanSummary } from './api'

const STATUS_COLOR: Record<string, string> = {
  done: 'bg-accent/10 text-accent border-accent-ink',
  planning: 'bg-info/10 text-info border-info/30',
  draft: 'bg-edge/60 text-ink-4 border-edge-soft',
  in_progress: 'bg-warn/10 text-warn border-warn-ink',
  paused: 'bg-edge/60 text-ink-5 border-edge-soft',
}

// PlanCard — a single plan in the grid: status badge, progress bar, task count.
export function PlanCard({ plan, onOpen }: { plan: PlanSummary; onOpen: (name: string) => void }) {
  const badge = STATUS_COLOR[plan.status] ?? 'bg-edge/60 text-ink-4 border-edge-soft'
  const pct = Math.round(plan.progress * 100)
  return (
    <button
      type="button"
      onClick={() => onOpen(plan.name)}
      className="group flex flex-col gap-3 rounded-md border border-edge bg-card p-4 text-left transition-colors hover:border-accent/60"
    >
      <div className="flex items-start justify-between gap-2">
        <h3 className="min-w-0 flex-1 truncate text-[13px] font-semibold text-ink group-hover:text-accent" title={plan.name}>
          {plan.name}
        </h3>
        <span className={`shrink-0 rounded border px-2 py-0.5 text-[11px] font-medium ${badge}`}>
          {plan.status || 'unknown'}
        </span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-edge">
        <div className="h-full rounded-full bg-accent" style={{ width: `${pct}%` }} />
      </div>
      <div className="flex items-center justify-between text-[11px] text-ink-5">
        <span>
          {plan.tasksDone}/{plan.tasksTotal} tasks · {pct}%
        </span>
        {plan.hasLedger && <span className="text-ink-6">SDD ledger</span>}
      </div>
    </button>
  )
}
