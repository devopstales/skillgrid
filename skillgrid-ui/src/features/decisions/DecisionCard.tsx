import { useState } from 'react'

// DecisionCard renders one decision from the inbox: the question, its
// lettered options (the recommended one highlighted), an optional note, and a
// Submit that hands {optionId, note} to onAnswer. On an answered decision it
// shows the stored choice + note; the options stay re-answerable (the backend
// appends a version) and re-selecting the picked option re-submits it.
// A decision whose content carries a `visual` gets a sandboxed prototype
// iframe in P3 — until then it renders a "visual available" placeholder,
// never an iframe.

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
      <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
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
    <div className="rounded-md border border-edge bg-card p-4">
      <div className="mb-1 flex items-baseline justify-between gap-2">
        <span className="truncate text-xs uppercase tracking-wide text-zinc-500">
          {decision.topicKey}
        </span>
        <span className="shrink-0 text-[11px] text-zinc-500">{day(decision.createdAt)}</span>
      </div>
      <h3 className="mb-3 font-semibold text-zinc-100">{content.question}</h3>

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
                'flex items-start gap-2 rounded border p-2 text-left text-sm transition',
                isRec ? 'recommended border-accent/70 bg-accent/10' : 'border-edge hover:bg-bg',
                isEffective ? 'ring-1 ring-accent' : '',
                isPicked ? 'border-emerald-500/60 bg-emerald-500/10' : '',
                busy ? 'opacity-60' : '',
              ].join(' ')}
            >
              <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-zinc-500 text-[10px] font-semibold text-zinc-300">
                {String.fromCharCode(65 + i)}
              </span>
              <span className="min-w-0 flex-1">
                <span className="font-medium text-zinc-200">{o.label}</span>
                {isRec && !answered && (
                  <span className="ml-2 rounded bg-accent/20 px-1 py-0.5 text-[10px] uppercase tracking-wide text-accent">
                    recommended
                  </span>
                )}
                {isPicked && (
                  <span className="ml-2 text-[11px] text-emerald-300">your choice</span>
                )}
                {o.rationale && (
                  <span className="mt-0.5 block text-xs text-zinc-500">{o.rationale}</span>
                )}
              </span>
            </button>
          )
        })}
      </div>

      {answered && content.answerNote && (
        <div className="mt-2 text-xs text-zinc-500">
          Note: <span className="text-zinc-300">{content.answerNote}</span>
          {content.updatedBy && (
            <span className="ml-2 text-zinc-600">· {content.updatedBy}</span>
          )}
        </div>
      )}

      {hasVisual && (
        <div className="mt-3 rounded-md border border-dashed border-edge px-3 py-2 text-xs text-zinc-500">
          Visual prototype available (renders in P3).
        </div>
      )}

      <div className="mt-3 flex items-center gap-2">
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Optional note for the agent…"
          aria-label="Optional note"
          className="min-w-0 flex-1 rounded-md border border-edge bg-bg px-2 py-1 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-accent/60 focus:outline-none"
        />
        <button
          type="button"
          data-testid="decision-submit"
          disabled={!effective || busy}
          onClick={submit}
          className="shrink-0 rounded-md bg-accent/80 px-3 py-1.5 text-xs font-medium text-zinc-950 hover:bg-accent disabled:opacity-40"
        >
          {answered ? 'Update answer' : 'Submit'}
        </button>
      </div>
    </div>
  )
}
