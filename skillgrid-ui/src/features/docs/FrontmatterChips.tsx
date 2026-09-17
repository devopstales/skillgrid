import type { DocsContent, DocsDocType } from './types'

// Doc-type badge label + accent class (schema-aware rendering cue).
const DOCTYPE_META: Record<DocsDocType, { label: string; cls: string }> = {
  adr: { label: 'ADR', cls: 'bg-violet-500/15 text-violet-300 border-violet-500/30' },
  prd: { label: 'PRD', cls: 'bg-sky-500/15 text-sky-300 border-sky-500/30' },
  task: { label: 'Task', cls: 'bg-amber-500/15 text-amber-300 border-amber-500/30' },
  doc: { label: 'Doc', cls: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30' },
  spec: { label: 'Spec', cls: 'bg-zinc-500/15 text-zinc-300 border-zinc-500/30' },
  note: { label: 'Note', cls: 'bg-zinc-500/10 text-zinc-400 border-zinc-600/30' },
}

// ADR lifecycle status → chip class.
const ADR_STATUS_CLS: Record<string, string> = {
  proposed: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
  accepted: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
  rejected: 'bg-rose-500/15 text-rose-300 border-rose-500/30',
  superseded: 'bg-zinc-500/15 text-zinc-400 border-zinc-600/30',
}

function DocTypeChip({ docType, decisionStatus }: { docType?: DocsDocType; decisionStatus?: string }) {
  if (!docType || docType === 'note') return null
  const meta = DOCTYPE_META[docType]
  const isAadr = docType === 'adr'
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[11px] font-semibold ${meta.cls}`}>
        {meta.label}
      </span>
      {isAadr && decisionStatus ? (
        <span
          className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[11px] font-medium ${
            ADR_STATUS_CLS[decisionStatus] ?? ADR_STATUS_CLS.proposed
          }`}
        >
          {decisionStatus}
        </span>
      ) : null}
    </span>
  )
}

// Frontmatter chips: doc-type + ADR status (schema-aware) / status / author /
// updated + any other key/value.
export function FrontmatterChips({ content }: { content: DocsContent }) {
  const fm = content.frontmatter

  const order: [string, string][] = []
  for (const key of ['status', 'priority', 'author', 'assignee', 'type', 'milestone', 'id']) {
    if (fm?.[key]) order.push([key, fm[key]])
  }
  for (const [k, v] of Object.entries(fm ?? {})) {
    if (!order.some(([kk]) => kk === k) && v) order.push([k, v])
  }
  if (order.length === 0 && content.docType === 'note') return null

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <DocTypeChip docType={content.docType} decisionStatus={content.decisionStatus} />
      {order.map(([k, v]) => (
        <span
          key={k}
          className="inline-flex items-center gap-1 rounded-md border border-edge bg-black/20 px-2 py-0.5 text-[11px]"
        >
          <span className="text-zinc-500">{k}:</span>
          <span className="font-medium text-zinc-200">{v}</span>
        </span>
      ))}
    </div>
  )
}
