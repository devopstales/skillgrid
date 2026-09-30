import { useMemo, useState } from 'react'
import type { DocsRoot, DocsTreeNode } from './types'
import { ROOT_LABEL } from './types'

function FileIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-3.5 w-3.5 shrink-0 text-ink-5" fill="none" stroke="currentColor" strokeWidth="1.2">
      <path d="M9.5 2H4a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V5.5L9.5 2Z" />
      <path d="M9.5 2v3.5H13" />
    </svg>
  )
}

function statusClass(s?: string): string {
  if (!s) return ''
  const v = s.toLowerCase()
  if (v.includes('done') || v.includes('pass') || v.includes('shipped')) return 'bg-accent/10 text-accent'
  if (v.includes('progress') || v.includes('ready')) return 'bg-warn/10 text-warn'
  if (v.includes('blocked') || v.includes('fail')) return 'bg-danger/10 text-danger'
  return 'bg-edge/50 text-ink-4'
}

function TreeItem({
  node,
  depth,
  selected,
  onSelect,
  filter,
  expanded,
  toggle,
}: {
  node: DocsTreeNode
  depth: number
  selected: string
  onSelect: (path: string) => void
  filter: string
  expanded: Set<string>
  toggle: (path: string) => void
}) {
  const title = node.title || node.name
  const matches =
    !filter ||
    title.toLowerCase().includes(filter) ||
    node.path.toLowerCase().includes(filter)

  if (node.dir) {
    const children = node.children ?? []
    const anyChildMatch =
      !filter ||
      children.some((c) => matchSubtree(c, filter)) ||
      title.toLowerCase().includes(filter)
    if (filter && !anyChildMatch) return null
    const open = expanded.has(node.path) || !!filter
    return (
      <div>
        <button
          type="button"
          onClick={() => toggle(node.path)}
          className="flex w-full items-center gap-1.5 rounded px-2 py-1 text-left text-[13px] text-ink-3 hover:bg-edge/40"
          style={{ paddingLeft: 8 + depth * 12 }}
        >
          <span className={`text-ink-6 transition-transform ${open ? 'rotate-90' : ''}`}>›</span>
          <span className="font-medium text-ink-2">{title}</span>
        </button>
        {open &&
          children.map((c) => (
            <TreeItem
              key={c.path}
              node={c}
              depth={depth + 1}
              selected={selected}
              onSelect={onSelect}
              filter={filter}
              expanded={expanded}
              toggle={toggle}
            />
          ))}
      </div>
    )
  }

  if (!matches) return null
  return (
    <button
      type="button"
      onClick={() => onSelect(node.path)}
      className={[
        'flex w-full items-center gap-1.5 rounded px-2 py-1 text-left text-[13px]',
        selected === node.path
          ? 'bg-accent/15 text-accent'
          : 'text-ink-4 hover:bg-edge/40 hover:text-ink-2',
      ].join(' ')}
      style={{ paddingLeft: 8 + depth * 12 + 12 }}
    >
      <FileIcon />
      <span className="min-w-0 flex-1 truncate">{title}</span>
      {node.status && (
        <span className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] ${statusClass(node.status)}`}>
          {node.status}
        </span>
      )}
    </button>
  )
}

function matchSubtree(node: DocsTreeNode, filter: string): boolean {
  if (node.dir) return (node.children ?? []).some((c) => matchSubtree(c, filter))
  return (node.title || node.name).toLowerCase().includes(filter) || node.path.toLowerCase().includes(filter)
}

export function DocsTree({
  nodes,
  root,
  selected,
  onSelect,
}: {
  nodes: DocsTreeNode[]
  root: DocsRoot
  selected: string
  onSelect: (path: string) => void
}) {
  const [filter, setFilter] = useState('')
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  function toggle(path: string) {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })
  }

  const flatCount = useMemo(() => countFiles(nodes), [nodes])

  return (
    <aside className="flex w-72 shrink-0 flex-col border-r border-edge bg-card">
      <div className="flex items-center gap-2 border-b border-edge-soft px-3 py-2.5">
        <span className="text-[12px] font-semibold uppercase tracking-[0.1em] text-ink-5">
          {ROOT_LABEL[root]}
        </span>
        <span className="ml-auto text-[11px] text-ink-6">{flatCount} files</span>
      </div>
      <div className="border-b border-edge-soft p-2">
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter docs…"
          className="w-full rounded border border-edge bg-inset px-2.5 py-1.5 text-[12px] text-ink-2 placeholder:text-ink-6 focus:border-accent/50 focus:outline-none"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {nodes.length === 0 && (
          <p className="px-2 py-4 text-[12px] text-ink-6">No markdown in this root.</p>
        )}
        {nodes.map((n) => (
          <TreeItem
            key={n.path}
            node={n}
            depth={0}
            selected={selected}
            onSelect={onSelect}
            filter={filter.trim().toLowerCase()}
            expanded={expanded}
            toggle={toggle}
          />
        ))}
      </div>
    </aside>
  )
}

function countFiles(nodes: DocsTreeNode[]): number {
  return nodes.reduce((acc, n) => (n.dir ? acc + countFiles(n.children ?? []) : acc + 1), 0)
}
