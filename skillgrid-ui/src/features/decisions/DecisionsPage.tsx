import { useCallback, useEffect, useState } from 'react'
import { DecisionCard, type Decision } from './DecisionCard'
import { answerDecision, DecisionApiError, fetchDecisions } from './api'

// DecisionsPage is the decision inbox: the pending interview questions the
// agent posted, pollable on an interval (no WebSocket/SSE), with an answer
// round-trip. A pending decision appears here, the user picks an option +
// optional note and submits (POST …/answer), the card flips to state=answered
// and shows the stored answer; a second submit re-answers (backend appends a
// version). The default tab is the pending inbox.

const REFRESH_MS = 4000

type Tab = 'pending' | 'answered' | 'all'

export function DecisionsPage() {
  const [tab, setTab] = useState<Tab>('pending')
  const [rows, setRows] = useState<Decision[]>([])
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
        <h1 className="text-sm font-semibold text-zinc-100">Decisions</h1>
        <div className="flex gap-1" role="tablist" aria-label="Decision state">
          {(['pending', 'answered', 'all'] as const).map((t) => (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={tab === t}
              onClick={() => setTab(t)}
              className={[
                'rounded px-2.5 py-1 text-xs transition-colors',
                tab === t ? 'bg-accent/20 text-accent' : 'text-zinc-500 hover:bg-card hover:text-zinc-300',
              ].join(' ')}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="border-b border-red-500/30 bg-red-500/10 px-4 py-2 text-xs text-red-300">
          {error}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {rows.length === 0 ? (
          <div className="text-sm text-zinc-500">No {tab} decisions.</div>
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
