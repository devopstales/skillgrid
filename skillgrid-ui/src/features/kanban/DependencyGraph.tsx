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
      <div className="rounded-md border border-edge bg-bg/40 p-4 text-xs text-zinc-600">
        Loading dependencies…
      </div>
    )
  }
  const out = deps.deps_out ?? []
  const inc = deps.deps_in ?? []
  if (out.length === 0 && inc.length === 0) {
    return (
      <div className="rounded-md border border-edge bg-bg/40 p-4 text-center text-xs text-zinc-600">
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
    <div className="overflow-x-auto rounded-md border border-edge bg-bg/40 p-3">
      <svg width="360" height={height} className="mx-auto block">
        {/* blockers → task */}
        {out.map((id, i) => {
          const y = yFor(i, out.length)
          return (
            <g key={`out-${id}`}>
              <line x1="90" y1={y + nodeH / 2} x2={cx - 40} y2={height / 2}
                stroke="#3f3f46" strokeWidth="1" markerEnd="url(#arrow)" />
              <rect x="20" y={y} width="70" height={nodeH} rx="4"
                fill="#18181b" stroke="#3f3f46" />
              <text x="55" y={y + nodeH / 2 + 3} textAnchor="middle"
                fontSize="10" fill="#a1a1aa" fontFamily="monospace">{id}</text>
            </g>
          )
        })}
        {/* task → dependents */}
        {inc.map((id, i) => {
          const y = yFor(i, inc.length)
          return (
            <g key={`in-${id}`}>
              <line x1={cx + 40} y1={height / 2} x2="270" y2={y + nodeH / 2}
                stroke="#3f3f46" strokeWidth="1" markerEnd="url(#arrow)" />
              <rect x="270" y={y} width="70" height={nodeH} rx="4"
                fill="#18181b" stroke="#3f3f46" />
              <text x="305" y={y + nodeH / 2 + 3} textAnchor="middle"
                fontSize="10" fill="#a1a1aa" fontFamily="monospace">{id}</text>
            </g>
          )
        })}
        {/* center task */}
        <rect x={cx - 40} y={height / 2 - nodeH / 2} width="80" height={nodeH}
          rx="4" fill="#1e1b4b" stroke="#6366f1" strokeWidth="1.5" />
        <text x={cx} y={height / 2 + 3} textAnchor="middle" fontSize="10"
          fill="#c7d2fe" fontFamily="monospace">{task.id}</text>
        <defs>
          <marker id="arrow" markerWidth="6" markerHeight="6" refX="5" refY="3"
            orient="auto">
            <path d="M0,0 L6,3 L0,6 Z" fill="#3f3f46" />
          </marker>
        </defs>
      </svg>
      <div className="mt-2 flex justify-center gap-4 text-[10px] text-zinc-500">
        <span>← blockers ({out.length})</span>
        <span>dependents ({inc.length}) →</span>
      </div>
    </div>
  )
}
