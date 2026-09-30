import type { DocsContent, DocsDocType } from './types'

// Doc-type badge label + accent class (schema-aware rendering cue).
const DOCTYPE_META: Record<DocsDocType, { label: string; cls: string }> = {
  adr: { label: 'ADR', cls: 'bg-violet/15 text-violet border-violet/30' },
  prd: { label: 'PRD', cls: 'bg-info/10 text-info border-info/30' },
  task: { label: 'Task', cls: 'bg-warn/10 text-warn border-warn-ink' },
  doc: { label: 'Doc', cls: 'bg-accent/10 text-accent border-accent-ink' },
  spec: { label: 'Spec', cls: 'bg-edge/60 text-ink-4 border-edge-soft' },
  note: { label: 'Note', cls: 'bg-edge/40 text-ink-5 border-edge-soft' },
}

// ADR lifecycle status → chip class.
const ADR_STATUS_CLS: Record<string, string> = {
  proposed: 'bg-warn/10 text-warn border-warn-ink',
  accepted: 'bg-accent/10 text-accent border-accent-ink',
  rejected: 'bg-danger/10 text-danger border-danger-ink',
  superseded: 'bg-edge/60 text-ink-5 border-edge-soft',
}

function DocTypeChip({ docType, decisionStatus }: { docType?: DocsDocType; decisionStatus?: string }) {
  if (!docType || docType === 'note') return null
  const meta = DOCTYPE_META[docType]
  const isAadr = docType === 'adr'
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`inline-flex items-center rounded border px-2 py-0.5 text-[11px] font-semibold ${meta.cls}`}>
        {meta.label}
      </span>
      {isAadr && decisionStatus ? (
        <span
          className={`inline-flex items-center rounded border px-2 py-0.5 text-[11px] font-medium ${
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
          className="inline-flex items-center gap-1 rounded border border-edge bg-black/20 px-2 py-0.5 text-[11px]"
        >
          <span className="text-ink-5">{k}:</span>
          <span className="font-medium text-ink-2">{v}</span>
        </span>
      ))}
    </div>
  )
}
