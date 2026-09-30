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
    <div className="w-64 rounded border border-edge bg-surface-2/90 p-2">
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
        className="w-full rounded border border-edge-soft bg-inset px-2 py-1 text-[12px] text-ink-2 outline-none placeholder:text-ink-6 focus:border-accent"
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
                className="w-full truncate rounded px-2 py-1 text-left text-[12px] text-ink-3 hover:bg-surface-2"
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
          className="mt-1 text-[11px] text-ink-6 hover:text-ink-3"
        >
          clear highlight
        </button>
      )}
    </div>
  )
}
