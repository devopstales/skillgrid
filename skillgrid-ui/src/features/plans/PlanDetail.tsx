import type { PlanDetail } from './api'
import { MarkdownView } from '../docs/MarkdownView'
import { linkifyTaskRefs, linkifyTaskRefsMarkdown } from './taskLinks'

const STATUS_COLOR: Record<string, string> = {
  COMPLETE: 'text-accent',
  PENDING: 'text-ink-5',
  DONE: 'text-accent',
}

// PlanDetail — the right-hand detail panel for a selected plan: status +
// progress, SDD ledger steps (checklist), file inventory, and a "Read in Docs"
// cross-link. The spec markdown itself is rendered by SpecViewer (parent).
export function PlanDetailPanel({
  plan,
  onOpenSpec,
}: {
  plan: PlanDetail
  onOpenSpec: (path: string) => void
}) {
  const pct = Math.round(plan.progress * 100)
  return (
    <div className="space-y-5">
      <div>
        <div className="flex items-center gap-2">
          <h2 className="text-[14px] font-semibold text-ink">{plan.name}</h2>
          <span className="rounded bg-accent/15 px-2 py-0.5 text-[11px] text-accent">
            {plan.status || 'unknown'}
          </span>
        </div>
        <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-edge">
          <div className="h-full rounded-full bg-accent" style={{ width: `${pct}%` }} />
        </div>
        <p className="mt-1 text-[11px] text-ink-5">
          {plan.tasksDone}/{plan.tasksTotal} tasks · {pct}%
        </p>
      </div>

      {plan.briefing.trim() !== '' && (
        <section>
          <h3 className="mb-2 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
            Briefing
          </h3>
          <MarkdownView body={linkifyTaskRefsMarkdown(plan.briefing)} />
        </section>
      )}

      {(plan.steps?.length ?? 0) > 0 && (
        <section>
          <h3 className="mb-2 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
            SDD ledger steps
          </h3>
          <ul className="space-y-1">
            {plan.steps!.map((s, i) => (
              <li
                key={i}
                className="flex items-start gap-2 rounded border border-edge bg-card px-3 py-1.5 text-[12px]"
              >
                <span className={`shrink-0 ${STATUS_COLOR[s.status] ?? 'text-ink-4'}`}>
                  {s.status === 'COMPLETE' || s.status === 'DONE' ? '✓' : '○'}
                </span>
                <span
                  className="min-w-0 flex-1 text-ink-3"
                  dangerouslySetInnerHTML={{
                    __html: linkifyTaskRefs(s.raw.replace(/^[-*]\s+/, '')),
                  }}
                />
              </li>
            ))}
          </ul>
        </section>
      )}

      <section>
        <h3 className="mb-2 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">Files</h3>
        <ul className="space-y-1">
          {plan.files.map((f) => (
            <li key={f.name}>
              <button
                type="button"
                onClick={() => onOpenSpec(`${plan.name}/${f.name}`)}
                className="w-full truncate rounded border border-edge bg-card px-3 py-1.5 text-left text-[12px] text-ink-3 hover:border-accent/60 hover:text-accent"
                title={`${f.name} (${f.size} bytes)`}
              >
                <span className="mr-2 text-ink-6">{(f.size / 1024).toFixed(1)} KB</span>
                {f.name}
              </button>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}
