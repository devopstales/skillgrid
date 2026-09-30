import { useState } from 'react'

// DecisionCard renders one decision from the inbox: the question, its
// lettered options (the recommended one highlighted), an optional note, and a
// Submit that hands {optionId, note} to onAnswer. On an answered decision it
// shows the stored choice + note; the options stay re-answerable (the backend
// appends a version) and re-selecting the picked option re-submits it.
// A decision whose content carries a `visual` gets a sandboxed prototype
// iframe (src=/prototype/{visual}, sandbox="allow-scripts", no
// allow-same-origin) rendering the companion's throwaway HTML from
// .skillgrid/prototype/ — the same sandbox contract as SandboxPreview.

export interface DecisionOption {
  id: string
  label: string
  rationale?: string
}

export interface DecisionContent {
  question: string
  options: DecisionOption[]
  recommended?: string
  state: string
  answeredOption?: string
  answerNote?: string
  updatedBy?: string
  visual?: string
}

export interface Decision {
  id: number
  topicKey: string
  title: string
  createdAt: string
  updatedAt: string
  visibility: string
  parseError?: boolean
  content: DecisionContent
}

function day(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 10)
}

export function DecisionCard({
  decision,
  onAnswer,
}: {
  decision: Decision
  onAnswer: (id: number, answer: { optionId: string; note: string }) => void
}) {
  const [note, setNote] = useState('')
  const [selected, setSelected] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  if (decision.parseError) {
    return (
      <div className="rounded border border-warn-ink bg-warn/10 p-3 text-[13px] text-warn">
        Unreadable decision: {decision.topicKey}
      </div>
    )
  }

  const { content } = decision
  const answered = content.state === 'answered'
  const pickedId = answered ? content.answeredOption : undefined
  const effective = selected ?? pickedId ?? content.recommended ?? ''
  const hasVisual = Boolean(content.visual)

  const submit = () => {
    if (!effective || busy) return
    setBusy(true)
    onAnswer(decision.id, { optionId: effective, note })
  }

  return (
    <div className="rounded border border-edge bg-card p-4">
      <div className="mb-1 flex items-baseline justify-between gap-2">
        <span className="truncate text-[12px] uppercase tracking-[0.1em] text-ink-5">
          {decision.topicKey}
        </span>
        <span className="shrink-0 text-[11px] text-ink-5">{day(decision.createdAt)}</span>
      </div>
      <h3 className="mb-3 font-semibold text-ink">{content.question}</h3>

      <div className="flex flex-col gap-2" role="radiogroup" aria-label="Options">
        {content.options.map((o, i) => {
          const isRec = o.id === content.recommended
          const isPicked = answered && o.id === content.answeredOption
          const isEffective = effective === o.id
          return (
            <button
              key={o.id}
              type="button"
              role="radio"
              aria-checked={isEffective}
              data-testid={`decision-option-${o.id}`}
              disabled={busy}
              onClick={() => setSelected(o.id)}
              className={[
                'flex items-start gap-2 rounded border p-2 text-left text-[13px] transition',
                isRec ? 'recommended border-accent/70 bg-accent/10' : 'border-edge hover:bg-bg',
                isEffective ? 'ring-1 ring-accent' : '',
                isPicked ? 'border-accent/60 bg-accent/10' : '',
                busy ? 'opacity-60' : '',
              ].join(' ')}
            >
              <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-ink-5 text-[10px] font-semibold text-ink-3">
                {String.fromCharCode(65 + i)}
              </span>
              <span className="min-w-0 flex-1">
                <span className="font-medium text-ink-2">{o.label}</span>
                {isRec && !answered && (
                  <span className="ml-2 rounded bg-accent/20 px-1 py-0.5 text-[10px] uppercase tracking-wide text-accent">
                    recommended
                  </span>
                )}
                {isPicked && (
                  <span className="ml-2 text-[11px] text-accent">your choice</span>
                )}
                {o.rationale && (
                  <span className="mt-0.5 block text-[12px] text-ink-5">{o.rationale}</span>
                )}
              </span>
            </button>
          )
        })}
      </div>

      {answered && content.answerNote && (
        <div className="mt-2 text-[12px] text-ink-5">
          Note: <span className="text-ink-3">{content.answerNote}</span>
          {content.updatedBy && (
            <span className="ml-2 text-ink-6">· {content.updatedBy}</span>
          )}
        </div>
      )}

      {hasVisual && (
        <div className="mt-3 overflow-hidden rounded border border-edge bg-white">
          <iframe
            title={`Prototype: ${content.visual}`}
            src={`/prototype/${content.visual}`}
            sandbox="allow-scripts"
            className="h-[420px] w-full border-0"
          />
        </div>
      )}

      <div className="mt-3 flex items-center gap-2">
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Optional note for the agent…"
          aria-label="Optional note"
          className="min-w-0 flex-1 rounded border border-edge bg-inset px-2 py-1 text-[13px] text-ink-2 placeholder:text-ink-6 focus:border-accent/60 focus:outline-none"
        />
        <button
          type="button"
          data-testid="decision-submit"
          disabled={!effective || busy}
          onClick={submit}
          className="shrink-0 rounded bg-accent/80 px-3 py-1.5 text-[12px] font-medium text-bg hover:bg-accent disabled:opacity-40"
        >
          {answered ? 'Update answer' : 'Submit'}
        </button>
      </div>
    </div>
  )
}
