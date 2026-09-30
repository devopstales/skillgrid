import { useCallback, useEffect, useState } from 'react'
import {
  fetchFileContent,
  fetchFileTree,
  MnemonicError,
  type FileContentResponse,
  type FileTreeNode,
} from './api'

// FilesPage is the OpenViking view: a topic_key-derived file tree on the left,
// and an L0/L1/L2 content pane on the right (abstract → overview → details).

export function FilesPage() {
  const [tree, setTree] = useState<FileTreeNode | null>(null)
  const [treeError, setTreeError] = useState('')
  const [selected, setSelected] = useState('')
  const [content, setContent] = useState<FileContentResponse | null>(null)
  const [contentError, setContentError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    setTreeError('')
    fetchFileTree()
      .then((res) => {
        if (!cancelled) setTree(res.root)
      })
      .catch((e) => {
        if (!cancelled) setTreeError(e instanceof MnemonicError ? e.message : 'failed to load tree')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const load = useCallback((path: string) => {
    if (!path) {
      setContent(null)
      return
    }
    let cancelled = false
    setLoading(true)
    setContentError('')
    fetchFileContent(`mnemonic://${path}`)
      .then((c) => {
        if (!cancelled) setContent(c)
      })
      .catch((e) => {
        if (!cancelled)
          setContentError(e instanceof MnemonicError ? e.message : 'failed to load content')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => load(selected), [selected, load])

  return (
    <div className="flex h-full min-h-0 flex-col font-mono">
      <div className="flex items-center gap-2 border-b border-edge-soft px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Files</h1>
        <span className="text-[12px] text-ink-5">OpenViking — topic_key tree</span>
      </div>

      {treeError && (
        <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
          {treeError}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Tree */}
        <aside className="w-72 shrink-0 overflow-y-auto border-r border-edge-soft px-2 py-3">
          {tree ? (
            <FileTree nodes={tree.children ?? []} selected={selected} onSelect={setSelected} />
          ) : (
            <div className="p-2 text-[12px] text-ink-5">Loading tree…</div>
          )}
        </aside>

        {/* Content pane */}
        <main className="min-h-0 flex-1 overflow-y-auto">
          {!selected && (
            <div className="flex h-full items-center justify-center p-8 text-center">
              <span className="text-[13px] text-ink-5">
                Select a topic from the tree to read its memory content.
              </span>
            </div>
          )}

          {selected && loading && (
            <div className="flex h-full items-center justify-center text-[13px] text-ink-5">
              Loading…
            </div>
          )}

          {selected && contentError && (
            <div className="flex h-full items-center justify-center p-8">
              <span className="rounded border border-danger-ink bg-danger/10 px-4 py-2 text-[13px] text-danger">
                {contentError}
              </span>
            </div>
          )}

          {selected && content && <ContentPanel content={content} />}
        </main>
      </div>
    </div>
  )
}

function FileTree({
  nodes,
  selected,
  onSelect,
}: {
  nodes: FileTreeNode[]
  selected: string
  onSelect: (path: string) => void
}) {
  if (nodes.length === 0) {
    return <div className="p-2 text-[12px] text-ink-6">No topics indexed yet.</div>
  }
  return (
    <ul className="space-y-0.5">
      {nodes.map((n) => (
        <TreeNode key={n.path + n.name} node={n} selected={selected} onSelect={onSelect} depth={0} />
      ))}
    </ul>
  )
}

function TreeNode({
  node,
  selected,
  onSelect,
  depth,
}: {
  node: FileTreeNode
  selected: string
  onSelect: (path: string) => void
  depth: number
}) {
  const [open, setOpen] = useState(depth < 2)
  const isDir = (node.children?.length ?? 0) > 0
  const isSel = selected === node.path
  return (
    <li>
      <div
        className={[
          'group flex cursor-pointer items-center gap-1 rounded px-1.5 py-1 text-[13px]',
          isSel ? 'bg-accent/15 text-accent' : 'text-ink-3 hover:bg-card',
        ].join(' ')}
        style={{ paddingLeft: `${depth * 12 + 6}px` }}
        onClick={() => (isDir ? setOpen((o) => !o) : onSelect(node.path))}
      >
        <span className="w-3 text-center text-[12px] text-ink-5">
          {isDir ? (open ? '▾' : '▸') : node.leaf ? '◆' : '·'}
        </span>
        <span className="min-w-0 flex-1 truncate">{node.name}</span>
        {node.memory_count > 0 && (
          <span className="shrink-0 rounded bg-edge/60 px-1 text-[10px] text-ink-4">
            {node.memory_count}
          </span>
        )}
      </div>
      {isDir && open && (
        <ul className="space-y-0.5">
          {node.children!.map((c) => (
            <TreeNode key={c.path + c.name} node={c} selected={selected} onSelect={onSelect} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  )
}

function ContentPanel({ content }: { content: FileContentResponse }) {
  const l0 = content.l0
  const l1 = content.l1
  const l2 = content.l2
  return (
    <div className="px-6 py-5 font-mono">
      <div className="mb-1 text-[11px] uppercase tracking-[0.14em] text-ink-6">{content.uri}</div>
      <h1 className="mb-4 text-[19px] font-semibold text-ink">
        {content.uri.split('/').pop()}
      </h1>

      {l0 && (
        <Tier label="Abstract" tint="violet">
          <p className="text-[13px] text-ink-3">{l0.content}</p>
        </Tier>
      )}

      {l1 && (
        <Tier label={`Overview · ${l1.count} memories`} tint="info">
          <ul className="space-y-1">
            {(l1.items ?? []).map((it) => (
              <li key={it.id} className="flex items-baseline gap-2 text-[13px]">
                <span className="text-ink-5">{it.type}</span>
                <span className="text-ink-2">{it.title}</span>
                <span className="ml-auto text-[12px] text-ink-6">{it.created_at}</span>
              </li>
            ))}
          </ul>
        </Tier>
      )}

      {l2 && (
        <Tier label={`Details · ${l2.count} memories`} tint="accent">
          <div className="space-y-4">
            {(l2.items ?? []).map((it) => (
              <article key={it.id} className="rounded border border-edge bg-surface-2/60 p-3">
                <div className="mb-1 flex items-center gap-2">
                  <span className="rounded bg-edge/60 px-1.5 py-0.5 text-[10px] text-ink-4">
                    {it.type}
                  </span>
                  <span className="text-[13px] font-medium text-ink-2">{it.title}</span>
                  <span className="ml-auto text-[12px] text-ink-6">{it.created_at}</span>
                </div>
                <pre className="whitespace-pre-wrap text-[12px] leading-relaxed text-ink-4">
                  {it.content}
                </pre>
              </article>
            ))}
          </div>
        </Tier>
      )}
    </div>
  )
}

function Tier({
  label,
  tint,
  children,
}: {
  label: string
  tint: 'violet' | 'info' | 'accent'
  children: React.ReactNode
}) {
  const border = {
    violet: 'border-violet/30',
    info: 'border-info/30',
    accent: 'border-accent-ink',
  }[tint]
  const labelColor = {
    violet: 'text-violet',
    info: 'text-info',
    accent: 'text-accent',
  }[tint]
  return (
    <section className={`mb-4 rounded border ${border} bg-card/30 p-3`}>
      <div className={`mb-2 text-[11px] font-semibold uppercase tracking-[0.12em] ${labelColor}`}>
        {label}
      </div>
      {children}
    </section>
  )
}
