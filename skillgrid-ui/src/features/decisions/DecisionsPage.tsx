import { useCallback, useEffect, useState } from 'react'
import { DecisionCard, type Decision } from './DecisionCard'
import { answerDecision, DecisionApiError, fetchDecisions, fetchOpenQuestions, type OpenQuestion } from './api'

// DecisionsPage is the decision inbox: the pending interview questions the
// agent posted, pollable on an interval (no WebSocket/SSE), with an answer
// round-trip. A pending decision appears here, the user picks an option +
// optional note and submits (POST …/answer), the card flips to state=answered
// and shows the stored answer; a second submit re-answers (backend appends a
// version). The default tab is the pending inbox.

const REFRESH_MS = 4000

type Tab = 'pending' | 'answered' | 'all' | 'open'

export function DecisionsPage() {
  const [tab, setTab] = useState<Tab>('pending')
  const [rows, setRows] = useState<Decision[]>([])
  const [openQuestions, setOpenQuestions] = useState<OpenQuestion[]>([])
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    try {
      setRows(await fetchDecisions(tab))
      setError('')
    } catch (e) {
      setError(e instanceof DecisionApiError ? e.message : 'failed to load decisions')
    }
  }, [tab])

  useEffect(() => {
    void load()
    const t = setInterval(() => void load(), REFRESH_MS)
    return () => clearInterval(t)
  }, [load])

  // The "open" tab reads ASSUMPTIONS.md (open questions) on demand, not on
  // the decisions poll interval.
  useEffect(() => {
    if (tab !== 'open') return
    let cancelled = false
    void fetchOpenQuestions()
      .then((qs) => {
        if (!cancelled) {
          setOpenQuestions(qs)
          setError('')
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setOpenQuestions([])
          setError(e instanceof DecisionApiError ? e.message : 'failed to load open questions')
        }
      })
    return () => {
      cancelled = true
    }
  }, [tab])

  const onAnswer = useCallback(
    async (id: number, answer: { optionId: string; note: string }) => {
      await answerDecision(id, answer)
      void load()
    },
    [load],
  )

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 border-b border-edge px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Decisions</h1>
        <div className="flex gap-1" role="tablist" aria-label="Decision state">
          {(['pending', 'answered', 'all', 'open'] as const).map((t) => (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={tab === t}
              onClick={() => setTab(t)}
                className={[
                  'rounded px-2.5 py-1 text-[12px] transition-colors',
                  tab === t ? 'bg-accent/20 text-accent' : 'text-ink-5 hover:bg-card hover:text-ink-3',
                ].join(' ')}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
          {error}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {tab === 'open' ? (
          openQuestions.length === 0 ? (
            <div className="text-[13px] text-ink-5">No open questions.</div>
          ) : (
            <ul className="mx-auto flex max-w-2xl flex-col gap-2">
              {openQuestions.map((q, i) => (
                <li
                  key={i}
                  className="flex items-start gap-2 rounded border border-edge-soft bg-card px-3 py-2"
                >
                  <span
                    className={[
                      'mt-0.5 inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px]',
                      q.checked ? 'border-accent bg-accent/20 text-accent' : 'border-edge text-ink-6',
                    ].join(' ')}
                    aria-label={q.checked ? 'resolved' : 'unresolved'}
                  >
                    {q.checked ? '✓' : ''}
                  </span>
                  <span className="min-w-0 flex-1 text-[13px] text-ink-2">{q.text}</span>
                </li>
              ))}
            </ul>
          )
        ) : rows.length === 0 ? (
          <div className="text-[13px] text-ink-5">No {tab} decisions.</div>
        ) : (
          <div className="mx-auto flex max-w-2xl flex-col gap-3">
            {rows.map((d) => (
              <DecisionCard key={d.id} decision={d} onAnswer={onAnswer} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
