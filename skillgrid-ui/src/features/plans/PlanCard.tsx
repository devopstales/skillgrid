import type { PlanSummary } from './api'

const STATUS_COLOR: Record<string, string> = {
  done: 'bg-emerald-500/15 text-emerald-300 border-emerald-800/50',
  planning: 'bg-sky-500/15 text-sky-300 border-sky-800/50',
  draft: 'bg-zinc-500/15 text-zinc-300 border-zinc-700/50',
  in_progress: 'bg-amber-500/15 text-amber-300 border-amber-800/50',
  paused: 'bg-zinc-500/15 text-zinc-400 border-zinc-700/50',
}

// PlanCard — a single plan in the grid: status badge, progress bar, task count.
export function PlanCard({ plan, onOpen }: { plan: PlanSummary; onOpen: (name: string) => void }) {
  const badge = STATUS_COLOR[plan.status] ?? 'bg-zinc-500/15 text-zinc-300 border-zinc-700/50'
  const pct = Math.round(plan.progress * 100)
  return (
    <button
      type="button"
      onClick={() => onOpen(plan.name)}
      className="group flex flex-col gap-3 rounded-lg border border-edge bg-card p-4 text-left transition-colors hover:border-accent/60"
    >
      <div className="flex items-start justify-between gap-2">
        <h3 className="min-w-0 flex-1 truncate text-sm font-semibold text-zinc-100 group-hover:text-accent" title={plan.name}>
          {plan.name}
        </h3>
        <span className={`shrink-0 rounded border px-2 py-0.5 text-[11px] font-medium ${badge}`}>
          {plan.status || 'unknown'}
        </span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-edge">
        <div className="h-full rounded-full bg-accent" style={{ width: `${pct}%` }} />
      </div>
      <div className="flex items-center justify-between text-[11px] text-zinc-500">
        <span>
          {plan.tasksDone}/{plan.tasksTotal} tasks · {pct}%
        </span>
        {plan.hasLedger && <span className="text-zinc-600">SDD ledger</span>}
      </div>
    </button>
  )
}
