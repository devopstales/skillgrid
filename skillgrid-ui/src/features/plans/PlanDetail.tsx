import type { PlanDetail } from './api'
import { MarkdownView } from '../docs/MarkdownView'
import { linkifyTaskRefs, linkifyTaskRefsMarkdown } from './taskLinks'
import { StatusChip } from './status'

// PlanDetail — the detail pane for a selected change: status chip + progress
// bullet, briefing, SDD ledger steps (✓/○ hairline rows), and the spec file
// inventory (each file is a button that opens it in SpecViewer).
export function PlanDetailPanel({
  plan,
  onOpenSpec,
}: {
  plan: PlanDetail
  onOpenSpec: (path: string) => void
}) {
  const pct = Math.round(plan.progress * 100)
  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center gap-2">
          <h2 className="truncate font-mono text-[14px] font-semibold text-ink">{plan.name}</h2>
          <StatusChip status={plan.status} />
          <span className="text-[11px] text-ink-5">{plan.status || 'unknown'}</span>
        </div>
        <div className="mt-2 flex items-center gap-3">
          <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-inset">
            <div className="h-full bg-accent" style={{ width: `${pct}%` }} />
          </div>
          <span className="shrink-0 font-mono text-[11px] tabular-nums text-ink-5">
            {plan.tasksDone}/{plan.tasksTotal} tasks · {pct}%
          </span>
        </div>
      </div>

      {(plan.steps?.length ?? 0) > 0 && (
        <section>
          <SectionHeader>SDD ledger steps</SectionHeader>
          <ul className="rounded-[6px] border border-edge bg-card">
            {plan.steps!.map((s, i) => {
              const done = s.status === 'COMPLETE' || s.status === 'DONE'
              return (
                <li
                  key={i}
                  className="flex items-start gap-2 border-b border-edge px-3 py-2 font-mono text-[12px] last:border-b-0"
                >
                  <span className={`shrink-0 ${done ? 'text-ok' : 'text-ink-5'}`}>{done ? '✓' : '○'}</span>
                  <span
                    className="min-w-0 flex-1 text-ink-3"
                    dangerouslySetInnerHTML={{
                      __html: linkifyTaskRefs(s.raw.replace(/^[-*]\s+/, '')),
                    }}
                  />
                </li>
              )
            })}
          </ul>
        </section>
      )}

      <section>
        <SectionHeader>Files</SectionHeader>
        {plan.files.length === 0 ? (
          <p className="text-[12px] text-ink-5">No spec files.</p>
        ) : (
          <ul className="rounded-[6px] border border-edge bg-card">
            {plan.files.map((f) => (
              <li key={f.name} className="border-b border-edge last:border-b-0">
                <button
                  type="button"
                  onClick={() => onOpenSpec(`${plan.name}/${f.name}`)}
                  className="grid w-full grid-cols-[minmax(0,1fr)_64px] items-center gap-2 px-3 py-2 text-left font-mono text-[12px] text-ink-3 transition-colors hover:bg-edge/40 hover:text-accent"
                  title={`${f.name} (${f.size} bytes)`}
                >
                  <span className="truncate">{f.name}</span>
                  <span className="text-right tabular-nums text-ink-6">{(f.size / 1024).toFixed(1)} KB</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      {plan.briefing.trim() !== '' && (
        <section>
          <SectionHeader>Briefing</SectionHeader>
          <MarkdownView body={linkifyTaskRefsMarkdown(plan.briefing)} />
        </section>
      )}
    </div>
  )
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mb-2 text-[11px] font-semibold uppercase tracking-[0.1em] text-ink-5">{children}</h3>
  )
}
