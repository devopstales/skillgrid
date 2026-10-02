import type { PlanSummary } from './api'
import { StatusChip } from './status'

// Column template shared by the header and the rows so they stay aligned.
// Mirrors prototype 002 variant A `.ch`: name · phase · progress · chip.
export const CHANGE_COLS = 'grid-cols-[minmax(0,1fr)_110px_96px_72px]'

// ChangeRow — one change in the dense list. The whole row is a button: click
// selects the plan and opens its detail pane. `active` marks the selected row.
export function ChangeRow({
  plan,
  active,
  onOpen,
}: {
  plan: PlanSummary
  active: boolean
  onOpen: (name: string) => void
}) {
  const pct = Math.round(plan.progress * 100)
  return (
    <button
      type="button"
      onClick={() => onOpen(plan.name)}
      aria-current={active ? 'true' : undefined}
      data-testid="change-row"
      className={[
        'grid w-full items-center gap-2.5 border-b border-edge px-3 py-[11px] text-left font-mono text-[12px] transition-colors last:border-b-0',
        CHANGE_COLS,
        active ? 'bg-accent/10' : 'hover:bg-edge/40',
      ].join(' ')}
    >
      <span className="flex min-w-0 items-center gap-2">
        <span
          className={`truncate ${active ? 'text-accent' : 'text-ink-2'}`}
          title={plan.name}
        >
          {plan.name}
        </span>
        {plan.hasLedger && (
          <span className="shrink-0 rounded-[3px] border border-edge px-1 text-[10px] leading-[14px] text-ink-4">
            sdd
          </span>
        )}
      </span>
      <span className="truncate text-ink-5" title={plan.status || 'unknown'}>
        {plan.status || '—'}
      </span>
      <span className="text-right tabular-nums text-ink-6">
        {plan.tasksTotal > 0 ? `${plan.tasksDone}/${plan.tasksTotal} · ${pct}%` : '—'}
      </span>
      <span className="text-right">
        <StatusChip status={plan.status} />
      </span>
    </button>
  )
}
