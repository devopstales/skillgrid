import { useMemo, useState } from 'react'
import { buildPathTree, agentForPath, type PathTreeNode } from './explorer'
import type { GraphNode } from './types'

const AGENTS = [
  { id: 'cursor', label: 'cursor' },
  { id: 'kilo', label: 'kilo' },
  { id: 'opencode', label: 'opencode' },
  { id: 'all', label: 'shared' },
] as const

interface Props {
  nodes: GraphNode[]
  collapsed: boolean
  onToggleCollapse: () => void
  onSelectFile: (path: string) => void
  activeAgents: Set<string>
  onToggleAgent: (agent: string) => void
}

// ExplorerPanel is a left-column file tree built client-side from the graph
// nodes' `path` fields. "Search files…" filters the tree by filename; the agent
// chips (cursor/kilo/opencode/shared) filter by the agentForPath heuristic; a
// collapse button gives the graph full width. Clicking a file leaf calls
// onSelectFile (the parent highlights matching nodes).
export function ExplorerPanel({
  nodes,
  collapsed,
  onToggleCollapse,
  onSelectFile,
  activeAgents,
  onToggleAgent,
}: Props) {
  const [query, setQuery] = useState('')

  // Build the path→agent map once per nodes change (for the agent filter).
  const pathAgent = useMemo(() => {
    const m = new Map<string, string | null>()
    for (const n of nodes) m.set(n.path, agentForPath(n.path))
    return m
  }, [nodes])

  const tree = useMemo(() => {
    let paths = nodes.map((n) => n.path).filter(Boolean)
    if (activeAgents.size > 0) {
      paths = paths.filter((p) => {
        const a = pathAgent.get(p)
        return a != null && activeAgents.has(a)
      })
    }
    if (query.trim()) {
      const q = query.trim().toLowerCase()
      paths = paths.filter((p) => p.split('/').pop()!.toLowerCase().includes(q))
    }
    return buildPathTree(paths)
  }, [nodes, activeAgents, query, pathAgent])

  if (collapsed) {
    return (
      <div className="flex h-full w-9 items-center border-r border-edge bg-surface-2 font-mono">
        <button
          type="button"
          onClick={onToggleCollapse}
          className="flex h-full w-full items-center justify-center text-ink-5 hover:bg-card hover:text-ink"
          title="Expand explorer"
          aria-label="Expand explorer"
        >
          »
        </button>
      </div>
    )
  }

  return (
    <div className="flex h-full w-64 flex-col border-r border-edge bg-surface-2 font-mono">
      <div className="flex items-center justify-between border-b border-edge px-3 py-2">
        <span className="text-[11px] font-semibold uppercase tracking-[0.12em] text-ink-4">Explorer</span>
        <button
          type="button"
          onClick={onToggleCollapse}
          className="text-ink-5 hover:text-ink"
          title="Collapse explorer"
          aria-label="Collapse explorer"
        >
          «
        </button>
      </div>

      <div className="space-y-2 border-b border-edge p-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search files…"
          className="w-full rounded border border-edge-soft bg-inset px-2 py-1 text-[12px] text-ink-2 outline-none placeholder:text-ink-6 focus:border-accent"
        />
        <div className="flex flex-wrap gap-1">
          {AGENTS.map((a) => {
            const active = activeAgents.has(a.id)
            return (
              <button
                key={a.id}
                type="button"
                onClick={() => onToggleAgent(a.id)}
                className={
                  'rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors ' +
                  (active
                    ? 'border-accent bg-accent/15 text-accent'
                    : 'border-edge-soft bg-inset text-ink-4 hover:border-edge')
                }
              >
                {a.label}
              </button>
            )
          })}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-1 text-[12px]">
        <TreeNode node={tree} depth={0} onSelectFile={onSelectFile} />
      </div>
    </div>
  )
}

function TreeNode({
  node,
  depth,
  onSelectFile,
}: {
  node: PathTreeNode
  depth: number
  onSelectFile: (path: string) => void
}) {
  const [open, setOpen] = useState(depth < 1)
  const isDir = !!node.children
  if (!isDir) {
    return (
      <button
        type="button"
        onClick={() => node.file && onSelectFile(node.file)}
        className="flex w-full items-center gap-1 rounded px-1 py-0.5 text-left text-ink-3 hover:bg-card hover:text-ink"
        style={{ paddingLeft: depth * 12 + 4 }}
      >
        <span className="truncate">{node.name}</span>
      </button>
    )
  }
  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-1 rounded px-1 py-0.5 text-left font-medium text-ink-2 hover:bg-card"
        style={{ paddingLeft: depth * 12 + 4 }}
      >
        <span className="w-3 text-ink-6">{open ? '▾' : '▸'}</span>
        <span className="truncate">{node.name}/</span>
      </button>
      {open && (
        <div>
          {node.children!.map((c, i) => (
            <TreeNode key={c.name + i} node={c} depth={depth + 1} onSelectFile={onSelectFile} />
          ))}
        </div>
      )}
    </div>
  )
}
