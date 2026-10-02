import { useEffect, useState } from 'react'
import { fetchContent, DocsError } from '../docs/api'
import type { DocsContent } from '../docs/types'
import { MarkdownView } from '../docs/MarkdownView'
import { parseAdrIndex, type AdrEntry } from './types'
import { parseOpenQuestions, type OpenQuestion } from '../decisions/api'

// ASSUMPTIONS_PATH holds both the in-force table (ADR-0019) and the Open
// Questions section. INDEX_PATH is the legacy table for repos that have not
// moved it yet.
const ASSUMPTIONS_PATH = '.skillgrid/ASSUMPTIONS.md'
const INDEX_PATH = '.skillgrid/artifacts/03-adr-index.md'

// loadAdrIndex reads ASSUMPTIONS.md for the in-force table and its Open
// Questions, falling back to 03-adr-index.md when that file has no table.
// Resolves to empty lists when no source has a table; rejects only when every
// source failed to fetch.
async function loadAdrIndex(): Promise<{ entries: AdrEntry[]; questions: OpenQuestion[] }> {
  let lastErr: unknown = null
  let fetchedAny = false
  let questions: OpenQuestion[] = []
  let entries: AdrEntry[] = []

  try {
    const assumptions = await fetchContent(ASSUMPTIONS_PATH)
    fetchedAny = true
    questions = parseOpenQuestions(assumptions.body)
    entries = parseAdrIndex(assumptions.body)
  } catch (e) {
    lastErr = e
  }

  if (entries.length === 0) {
    try {
      const index = await fetchContent(INDEX_PATH)
      fetchedAny = true
      const parsed = parseAdrIndex(index.body)
      if (parsed.length > 0) entries = parsed
    } catch (e) {
      lastErr = e
    }
  }

  if (!fetchedAny && lastErr) throw lastErr
  return { entries, questions }
}

// questionTitle is the bold lead of an open-question item, before the em dash.
function questionTitle(text: string): string {
  const head = text.split(/\s+—\s+/)[0] ?? text
  return head.replace(/\*\*/g, '').trim()
}

function questionBody(text: string): string {
  const parts = text.split(/\s+—\s+/)
  return parts.length > 1 ? parts.slice(1).join(' — ') : text
}

// AdrsPage is the Decisions view with two tabs. "ADRs" is the in-force set:
// on load it fetches the ADR index (ASSUMPTIONS.md, falling back to
// 03-adr-index.md) and parses its GFM table into structured entries; selecting
// a row fetches the full record (04-adr-NNNN-slug.md) from /docs/content and
// renders it in a detail pane. "Open questions" lists the ASSUMPTIONS.md
// § Open Questions items from the same fetch. Status badges are colored per
// ADR lifecycle (accepted → accent/green, superseded → muted).
type Tab = 'adrs' | 'open'

export function AdrsPage() {
  const [tab, setTab] = useState<Tab>('adrs')
  const [entries, setEntries] = useState<AdrEntry[]>([])
  const [questions, setQuestions] = useState<OpenQuestion[]>([])
  const [selectedQuestion, setSelectedQuestion] = useState<number | null>(null)
  const [indexError, setIndexError] = useState('')
  const [indexLoading, setIndexLoading] = useState(true)

  const [selected, setSelected] = useState<AdrEntry | null>(null)
  const [record, setRecord] = useState<DocsContent | null>(null)
  const [recordError, setRecordError] = useState('')
  const [recordLoading, setRecordLoading] = useState(false)

  // load + parse the ADR index on mount
  useEffect(() => {
    let cancelled = false
    setIndexLoading(true)
    setIndexError('')
    loadAdrIndex()
      .then((parsed) => {
        if (!cancelled) {
          setEntries(parsed.entries)
          setQuestions(parsed.questions)
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setEntries([])
          setIndexError(e instanceof DocsError ? e.message : 'failed to load ADR index')
        }
      })
      .finally(() => {
        if (!cancelled) setIndexLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // fetch the full record when an entry is selected
  useEffect(() => {
    if (!selected) {
      setRecord(null)
      setRecordError('')
      return
    }
    let cancelled = false
    setRecordLoading(true)
    setRecordError('')
    setRecord(null)
    fetchContent(selected.path)
      .then((c) => {
        if (!cancelled) setRecord(c)
      })
      .catch((e) => {
        if (!cancelled) setRecordError(e instanceof DocsError ? e.message : 'failed to load ADR record')
      })
      .finally(() => {
        if (!cancelled) setRecordLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [selected])

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 border-b border-edge px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Decisions</h1>
        <div className="flex gap-1" role="tablist" aria-label="Decision view">
          {(
            [
              { id: 'adrs', label: 'ADRs', count: entries.length },
              { id: 'open', label: 'Open questions', count: questions.length },
            ] as const
          ).map((t) => (
            <button
              key={t.id}
              type="button"
              role="tab"
              aria-selected={tab === t.id}
              onClick={() => setTab(t.id)}
              className={[
                'flex items-center gap-1.5 rounded px-2.5 py-1 text-[12px] transition-colors',
                tab === t.id ? 'bg-accent/20 text-accent' : 'text-ink-5 hover:bg-card hover:text-ink-3',
              ].join(' ')}
            >
              {t.label}
              <span className={tab === t.id ? 'text-accent/80' : 'text-ink-6'}>{t.count}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
        {/* List pane */}
        <div className="flex w-[460px] shrink-0 flex-col border-r border-edge bg-card">
          {indexError && (
            <div className="flex items-center gap-2 border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
              No ADR index found.
            </div>
          )}
          {indexLoading && (
            <div className="flex flex-1 items-center justify-center text-[13px] text-ink-5">Loading…</div>
          )}
          {!indexLoading && tab === 'open' && questions.length === 0 && (
            <div className="flex flex-1 items-center justify-center px-6 text-center text-[13px] text-ink-5">
              No open questions.
            </div>
          )}
          {!indexLoading && tab === 'open' && questions.length > 0 && (
            <div className="min-h-0 flex-1 overflow-y-auto p-2">
              <div className="mb-1 grid grid-cols-[3rem_1fr] gap-2 px-2 text-[10px] font-medium uppercase tracking-[0.1em] text-ink-6">
                <span>#</span>
                <span>Question</span>
              </div>
              <ul className="flex flex-col gap-1">
                {questions.map((q, i) => {
                  const active = selectedQuestion === i
                  return (
                    <li key={i}>
                      <button
                        type="button"
                        onClick={() => setSelectedQuestion(i)}
                        className={[
                          'flex w-full items-start gap-2 rounded border p-2 text-left transition',
                          active ? 'border-accent/60 bg-accent/10' : 'border-transparent hover:bg-edge/40',
                        ].join(' ')}
                      >
                        <span className="mt-0.5 shrink-0 font-mono text-[12px] text-ink-5">{i + 1}</span>
                        <span className="min-w-0 flex-1 text-[13px] font-medium leading-snug text-ink-2">
                          {questionTitle(q.text)}
                        </span>
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>
          )}
          {!indexLoading && tab === 'adrs' && !indexError && entries.length === 0 && (
            <div className="flex flex-1 items-center justify-center px-6 text-center text-[13px] text-ink-5">
              No ADR index found.
            </div>
          )}
          {!indexLoading && tab === 'adrs' && entries.length > 0 && (
            <div className="min-h-0 flex-1 overflow-y-auto p-2">
              <div className="mb-1 grid grid-cols-[3rem_1fr_6.5rem] gap-2 px-2 text-[10px] font-medium uppercase tracking-[0.1em] text-ink-6">
                <span>#</span>
                <span>Title</span>
                <span>Status</span>
              </div>
              <ul className="flex flex-col gap-1">
                {entries.map((e) => {
                  const active = selected?.path === e.path
                  return (
                    <li key={e.id}>
                      <button
                        type="button"
                        onClick={() => setSelected(e)}
                        className={[
                          'flex w-full items-start gap-2 rounded border p-2 text-left transition',
                          active ? 'border-accent/60 bg-accent/10' : 'border-transparent hover:bg-edge/40',
                        ].join(' ')}
                      >
                        <span className="mt-0.5 shrink-0 font-mono text-[12px] text-ink-5">{e.id}</span>
                        <span className="min-w-0 flex-1">
                          <span className="block text-[13px] font-medium leading-snug text-ink-2">
                            {e.title}
                          </span>
                          <span className="mt-0.5 block text-[11px] text-ink-6">{e.date}</span>
                        </span>
                        <StatusBadge status={e.status} inForce={e.inForce} />
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>
          )}
        </div>

        {/* Detail pane */}
        <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
          {tab === 'open' && selectedQuestion !== null && questions[selectedQuestion] && (
            <div className="px-6 py-5">
              <h1 className="mb-3 text-[14px] font-semibold text-ink">
                {questionTitle(questions[selectedQuestion].text)}
              </h1>
              <MarkdownView body={questionBody(questions[selectedQuestion].text)} />
            </div>
          )}
          {tab === 'open' && selectedQuestion === null && (
            <div className="flex flex-1 flex-col items-center justify-center gap-1 p-8 text-center">
              <span className="text-[13px] text-ink-4">Select an open question to read it.</span>
              <span className="text-[12px] text-ink-6">Not yet decided, not yet prototyped.</span>
            </div>
          )}
          {tab === 'adrs' && !selected && (
            <div className="flex flex-1 flex-col items-center justify-center gap-1 p-8 text-center">
              <span className="text-[13px] text-ink-4">Select an ADR to read its full record.</span>
              <span className="text-[12px] text-ink-6">Context → Decision → Consequences.</span>
            </div>
          )}
          {tab === 'adrs' && selected && recordLoading && (
            <div className="flex flex-1 items-center justify-center text-[13px] text-ink-5">Loading…</div>
          )}
          {tab === 'adrs' && selected && recordError && (
            <div className="flex flex-1 items-center justify-center p-8">
              <span className="rounded border border-danger-ink bg-danger/10 px-4 py-2 text-[13px] text-danger">
                {recordError}
              </span>
            </div>
          )}
          {tab === 'adrs' && selected && record && (
            <div className="px-6 py-5">
              <div className="mb-3 flex items-center gap-2">
                <StatusBadge status={record.decisionStatus ?? selected.status} inForce={selected.inForce} />
                <span className="text-[12px] text-ink-5">{record.path}</span>
              </div>
              <h1 className="mb-3 text-[14px] font-semibold text-ink">
                {record.title ?? selected.title}
              </h1>
              <MarkdownView body={record.body} securityLevel={record.mermaid?.securityLevel} />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function StatusBadge({ status, inForce }: { status: string; inForce: boolean }) {
  const s = status.toLowerCase()
  let cls = 'bg-edge/50 text-ink-4' // neutral / unknown
  if (s === 'accepted' && inForce) cls = 'bg-accent/15 text-accent'
  else if (s === 'accepted' && !inForce) cls = 'bg-edge/50 text-ink-5'
  else if (s === 'superseded' || s === 'deprecated' || s === 'rejected')
    cls = 'bg-edge/50 text-ink-5 line-through'
  else if (s === 'proposed' || s === 'draft') cls = 'bg-warn/10 text-warn'

  return (
    <span
      data-testid="adr-status-badge"
      data-status={s}
      className={[
        'mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide',
        cls,
      ].join(' ')}
    >
      {status || 'unknown'}
    </span>
  )
}
