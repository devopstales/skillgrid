import { useEffect, useState } from 'react'
import { fetchSearch, DocsError } from './api'
import type { DocsSearchHit } from './types'
import { ROOT_LABEL } from './types'

export function DocSearch({ onSelect }: { onSelect: (path: string) => void }) {
  const [q, setQ] = useState('')
  const [hits, setHits] = useState<DocsSearchHit[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!q.trim()) {
      setHits(null)
      return
    }
    let cancelled = false
    const t = setTimeout(async () => {
      setBusy(true)
      setError('')
      try {
        const res = await fetchSearch(q.trim())
        if (!cancelled) setHits(res.results)
      } catch (e) {
        if (!cancelled) setError(e instanceof DocsError ? e.message : 'search failed')
      } finally {
        if (!cancelled) setBusy(false)
      }
    }, 250)
    return () => {
      cancelled = true
      clearTimeout(t)
    }
  }, [q])

  return (
    <div className="relative">
      <input
        type="text"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Search docs… (title, body, path)"
        className="w-full rounded-md border border-edge bg-black/20 px-3 py-2 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-accent/50 focus:outline-none"
      />
      {(busy || hits !== null || error) && (
        <div className="absolute left-0 right-0 top-full z-20 mt-1 max-h-96 overflow-y-auto rounded-md border border-edge bg-card shadow-lg">
          {error && <p className="px-3 py-2 text-xs text-red-300">{error}</p>}
          {busy && !error && <p className="px-3 py-2 text-xs text-zinc-500">Searching…</p>}
          {!busy && hits && hits.length === 0 && !error && (
            <p className="px-3 py-2 text-xs text-zinc-500">No matches.</p>
          )}
          {!busy &&
            hits?.map((h) => (
              <button
                key={h.path}
                type="button"
                onClick={() => {
                  onSelect(h.path)
                  setQ('')
                  setHits(null)
                }}
                className="block w-full border-b border-edge/50 px-3 py-2 text-left last:border-0 hover:bg-edge/40"
              >
                <div className="flex items-center gap-2">
                  <span className="truncate text-sm text-zinc-200">{h.title || h.path}</span>
                  {h.root && (
                    <span className="ml-auto shrink-0 rounded bg-edge/50 px-1.5 py-0.5 text-[10px] text-zinc-500">
                      {ROOT_LABEL[h.root as keyof typeof ROOT_LABEL] ?? h.root}
                    </span>
                  )}
                </div>
                {h.snippet && (
                  <p className="mt-0.5 truncate text-xs text-zinc-500">{h.snippet}</p>
                )}
                <p className="mt-0.5 truncate text-[10px] text-zinc-600">{h.path}</p>
              </button>
            ))}
        </div>
      )}
    </div>
  )
}
