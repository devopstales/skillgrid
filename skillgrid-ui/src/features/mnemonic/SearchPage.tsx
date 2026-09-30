import { useState } from 'react'
import { MnemonicError, searchMemories, type SearchResult } from './api'

// SearchPage is the hybrid memory search: a query box (hybrid / fts modes), a
// ranked results list with relevance bars, and a "open in memories" deep link.

export function SearchPage() {
  const [q, setQ] = useState('')
  const [mode, setMode] = useState<'hybrid' | 'fts'>('hybrid')
  const [results, setResults] = useState<SearchResult[] | null>(null)
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const run = async (query: string, m: 'hybrid' | 'fts') => {
    const trimmed = query.trim()
    if (!trimmed) {
      setResults([])
      setTotal(0)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await searchMemories(trimmed, m, 25)
      setResults(res.results)
      setTotal(res.total)
    } catch (e) {
      setError(e instanceof MnemonicError ? e.message : 'search failed')
    } finally {
      setLoading(false)
    }
  }

  const maxRel = results && results.length > 0 ? results[0].relevance || 1 : 1

  return (
    <div className="flex h-full min-h-0 flex-col font-mono">
      <div className="border-b border-edge-soft px-4 py-3">
        <h1 className="mb-2 text-[14px] font-semibold text-ink">Search</h1>
        <form
          className="flex items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            run(q, mode)
          }}
        >
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="search memories (auth, tokens, bug)…"
            className="min-w-0 flex-1 rounded border border-edge-soft bg-inset px-3 py-2 text-[13px] text-ink-2 placeholder:text-ink-6 focus:border-accent/60 focus:outline-none"
          />
          <select
            value={mode}
            onChange={(e) => {
              const m = e.target.value as 'hybrid' | 'fts'
              setMode(m)
              run(q, m)
            }}
            className="rounded border border-edge-soft bg-inset px-2 py-2 text-[12px] text-ink-3"
          >
            <option value="hybrid">hybrid</option>
            <option value="fts">fts</option>
          </select>
          <button
            type="submit"
            disabled={loading}
            className="rounded border border-accent bg-accent/15 px-4 py-2 text-[13px] font-medium text-accent hover:bg-accent/25 disabled:opacity-40"
          >
            {loading ? 'Searching…' : 'Search'}
          </button>
        </form>
      </div>

      {error && (
        <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
          {error}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto">
        {results == null && (
          <div className="flex h-full items-center justify-center p-8 text-center text-[13px] text-ink-5">
            Type a query to search across the project's memories.
          </div>
        )}

        {results != null && results.length === 0 && (
          <div className="p-8 text-center text-[13px] text-ink-5">No results.</div>
        )}

        {results != null && results.length > 0 && (
          <div className="px-4 py-3">
            <div className="mb-2 text-[12px] text-ink-5">{total} results</div>
            <ul className="space-y-2">
              {results.map((r) => (
                <li
                  key={r.id}
                  className="rounded border border-edge bg-surface-2/60 p-3 transition-colors hover:border-accent/40"
                >
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-edge/60 px-1.5 py-0.5 text-[10px] text-ink-4">
                      {r.type}
                    </span>
                    <a
                      href={`/memories`}
                      title="Open in Memories"
                      className="truncate text-[13px] font-medium text-ink hover:text-accent"
                    >
                      {r.title || '(untitled)'}
                    </a>
                    {r.topic_key && <span className="text-[12px] text-ink-5">{r.topic_key}</span>}
                    <span className="ml-auto shrink-0 text-[12px] text-ink-6">{r.created_at.slice(0, 10)}</span>
                  </div>
                  <p className="mt-1.5 line-clamp-2 text-[12px] leading-relaxed text-ink-4">{r.preview}</p>
                  {/* relevance bar */}
                  <div className="mt-2 flex items-center gap-2">
                    <span className="text-[10px] uppercase tracking-[0.1em] text-ink-6">relevance</span>
                    <div className="h-1.5 w-40 overflow-hidden rounded-full bg-inset">
                      <div
                        className="h-full rounded-full bg-accent"
                        style={{ width: `${Math.round((r.relevance / maxRel) * 100)}%` }}
                      />
                    </div>
                    <span className="text-[10px] text-ink-5">{r.relevance.toFixed(2)}</span>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
