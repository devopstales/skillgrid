import { useMemo, useState } from 'react'

interface Props {
  // all node labels in the current graph (the search corpus)
  labels: string[]
  onSelect: (query: string) => void
  onClear: () => void
}

// SearchPanel: type-ahead over node labels. Enter (or clicking a suggestion)
// highlights the matching node in the graph.
export function SearchPanel({ labels, onSelect, onClear }: Props) {
  const [q, setQ] = useState('')
  const suggestions = useMemo(() => {
    const query = q.trim().toLowerCase()
    if (!query) return []
    return labels
      .filter((l) => l.toLowerCase().includes(query))
      .slice(0, 8)
  }, [labels, q])

  return (
    <div className="w-64 rounded-lg border border-slate-700 bg-slate-900/80 p-2">
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && q.trim()) {
            onSelect(q.trim())
            setQ('')
          }
        }}
        placeholder="Search nodes…"
        className="w-full rounded-md border border-slate-700 bg-slate-950 px-2 py-1 text-xs text-slate-100 outline-none placeholder:text-slate-500 focus:border-indigo-500"
      />
      {suggestions.length > 0 && (
        <ul className="mt-1 max-h-40 overflow-auto">
          {suggestions.map((s) => (
            <li key={s}>
              <button
                type="button"
                onClick={() => {
                  onSelect(s)
                  setQ('')
                }}
                className="w-full truncate rounded px-2 py-1 text-left text-xs text-slate-300 hover:bg-slate-800"
              >
                {s}
              </button>
            </li>
          ))}
        </ul>
      )}
      {q && (
        <button
          type="button"
          onClick={() => {
            setQ('')
            onClear()
          }}
          className="mt-1 text-[11px] text-slate-500 hover:text-slate-300"
        >
          clear highlight
        </button>
      )}
    </div>
  )
}
