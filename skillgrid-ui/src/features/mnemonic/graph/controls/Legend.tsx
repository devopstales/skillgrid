import { COMMUNITY_PALETTE, NEUTRAL_COLOR, communityStats } from '../communities'

interface Props {
  nodes: { community: number }[]
}

// Legend: one swatch per community (by member count, descending) + a neutral
// row for nodes without community data.
export function Legend({ nodes }: Props) {
  const stats = communityStats(nodes)
  const entries = [...stats.entries()].sort((a, b) => b[1] - a[1])

  return (
    <div className="max-h-52 w-44 overflow-auto rounded-lg border border-slate-700 bg-slate-900/80 p-2">
      <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-slate-400">
        Communities
      </div>
      <ul className="space-y-0.5">
        {entries.map(([id, count]) => (
          <li key={id} className="flex items-center gap-2 text-xs text-slate-300">
            <span
              className="inline-block h-3 w-3 rounded-full"
              style={{
                backgroundColor:
                  id < 0 ? NEUTRAL_COLOR : COMMUNITY_PALETTE[id % COMMUNITY_PALETTE.length],
              }}
            />
            <span className="truncate">
              {id < 0 ? 'no community' : `community ${id}`}
            </span>
            <span className="ml-auto text-[10px] text-slate-500">{count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
