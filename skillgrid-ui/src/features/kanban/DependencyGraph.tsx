import type { TrackerDeps } from './types'

// DependencyGraph is the task-detail mini-graph: the selected task in the
// center, its blockers (deps_out) on the left, its dependents (deps_in) on the
// right. SVG lines connect them. Degrades to a "no dependencies" note when
// both sides are empty (GitHub/GitLab/Jira expose no native edges).
export function DependencyGraph({
  task,
  deps,
}: {
  task: { id: string }
  deps: TrackerDeps | null
}) {
  if (!deps) {
    return (
      <div className="rounded border border-edge bg-surface-2/60 p-4 font-mono text-[12px] text-ink-6">
        Loading dependencies…
      </div>
    )
  }
  const out = deps.deps_out ?? []
  const inc = deps.deps_in ?? []
  if (out.length === 0 && inc.length === 0) {
    return (
      <div className="rounded border border-edge bg-surface-2/60 p-4 text-center font-mono text-[12px] text-ink-6">
        No dependencies
      </div>
    )
  }

  const nodeH = 28
  const gap = 8
  const cx = 180
  const maxRows = Math.max(out.length, inc.length, 1)
  const height = Math.max(120, maxRows * (nodeH + gap) + 24)

  const yFor = (i: number, total: number) =>
    height / 2 - ((total - 1) * (nodeH + gap)) / 2 + i * (nodeH + gap)

  return (
    <div className="overflow-x-auto rounded border border-edge bg-surface-2/60 p-3">
      <svg width="360" height={height} className="mx-auto block">
        {/* blockers → task */}
        {out.map((id, i) => {
          const y = yFor(i, out.length)
          return (
            <g key={`out-${id}`}>
              <line x1="90" y1={y + nodeH / 2} x2={cx - 40} y2={height / 2}
                stroke="#272f42" strokeWidth="1" markerEnd="url(#arrow)" />
              <rect x="20" y={y} width="70" height={nodeH} rx="4"
                fill="#1b2336" stroke="#272f42" />
              <text x="55" y={y + nodeH / 2 + 3} textAnchor="middle"
                fontSize="10" fill="#94a3b8" fontFamily="monospace">{id}</text>
            </g>
          )
        })}
        {/* task → dependents */}
        {inc.map((id, i) => {
          const y = yFor(i, inc.length)
          return (
            <g key={`in-${id}`}>
              <line x1={cx + 40} y1={height / 2} x2="270" y2={y + nodeH / 2}
                stroke="#272f42" strokeWidth="1" markerEnd="url(#arrow)" />
              <rect x="270" y={y} width="70" height={nodeH} rx="4"
                fill="#1b2336" stroke="#272f42" />
              <text x="305" y={y + nodeH / 2 + 3} textAnchor="middle"
                fontSize="10" fill="#94a3b8" fontFamily="monospace">{id}</text>
            </g>
          )
        })}
        {/* center task */}
        <rect x={cx - 40} y={height / 2 - nodeH / 2} width="80" height={nodeH}
          rx="4" fill="#14532d" stroke="#22c55e" strokeWidth="1.5" />
        <text x={cx} y={height / 2 + 3} textAnchor="middle" fontSize="10"
          fill="#bbf7d0" fontFamily="monospace">{task.id}</text>
        <defs>
          <marker id="arrow" markerWidth="6" markerHeight="6" refX="5" refY="3"
            orient="auto">
            <path d="M0,0 L6,3 L0,6 Z" fill="#272f42" />
          </marker>
        </defs>
      </svg>
      <div className="mt-2 flex justify-center gap-4 font-mono text-[10px] text-ink-5">
        <span>← blockers ({out.length})</span>
        <span>dependents ({inc.length}) →</span>
      </div>
    </div>
  )
}
