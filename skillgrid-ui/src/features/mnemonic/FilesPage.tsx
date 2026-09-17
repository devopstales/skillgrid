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
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
        <h1 className="text-sm font-semibold text-zinc-100">Files</h1>
        <span className="text-xs text-zinc-500">OpenViking — topic_key tree</span>
      </div>

      {treeError && (
        <div className="border-b border-red-500/30 bg-red-500/10 px-4 py-2 text-xs text-red-300">
          {treeError}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Tree */}
        <aside className="w-72 shrink-0 overflow-y-auto border-r border-edge px-2 py-3">
          {tree ? (
            <FileTree nodes={tree.children ?? []} selected={selected} onSelect={setSelected} />
          ) : (
            <div className="p-2 text-xs text-zinc-500">Loading tree…</div>
          )}
        </aside>

        {/* Content pane */}
        <main className="min-h-0 flex-1 overflow-y-auto">
          {!selected && (
            <div className="flex h-full items-center justify-center p-8 text-center">
              <span className="text-sm text-zinc-400">
                Select a topic from the tree to read its memory content.
              </span>
            </div>
          )}

          {selected && loading && (
            <div className="flex h-full items-center justify-center text-sm text-zinc-500">
              Loading…
            </div>
          )}

          {selected && contentError && (
            <div className="flex h-full items-center justify-center p-8">
              <span className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-300">
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
    return <div className="p-2 text-xs text-zinc-600">No topics indexed yet.</div>
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
          'group flex cursor-pointer items-center gap-1 rounded-md px-1.5 py-1 text-sm',
          isSel ? 'bg-accent/20 text-accent' : 'text-zinc-300 hover:bg-card',
        ].join(' ')}
        style={{ paddingLeft: `${depth * 12 + 6}px` }}
        onClick={() => (isDir ? setOpen((o) => !o) : onSelect(node.path))}
      >
        <span className="w-3 text-center text-xs text-zinc-500">
          {isDir ? (open ? '▾' : '▸') : node.leaf ? '◆' : '·'}
        </span>
        <span className="min-w-0 flex-1 truncate">{node.name}</span>
        {node.memory_count > 0 && (
          <span className="shrink-0 rounded bg-zinc-700/60 px-1 text-[10px] text-zinc-400">
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
    <div className="px-6 py-5">
      <div className="mb-1 text-xs uppercase tracking-widest text-zinc-500">{content.uri}</div>
      <h1 className="mb-4 text-xl font-semibold text-zinc-100">
        {content.uri.split('/').pop()}
      </h1>

      {l0 && (
        <Tier label="Abstract" tint="violet">
          <p className="text-sm text-zinc-300">{l0.content}</p>
        </Tier>
      )}

      {l1 && (
        <Tier label={`Overview · ${l1.count} memories`} tint="sky">
          <ul className="space-y-1">
            {(l1.items ?? []).map((it) => (
              <li key={it.id} className="flex items-baseline gap-2 text-sm">
                <span className="text-zinc-500">{it.type}</span>
                <span className="text-zinc-200">{it.title}</span>
                <span className="ml-auto text-xs text-zinc-600">{it.created_at}</span>
              </li>
            ))}
          </ul>
        </Tier>
      )}

      {l2 && (
        <Tier label={`Details · ${l2.count} memories`} tint="emerald">
          <div className="space-y-4">
            {(l2.items ?? []).map((it) => (
              <article key={it.id} className="rounded-md border border-edge bg-card/40 p-3">
                <div className="mb-1 flex items-center gap-2">
                  <span className="rounded bg-zinc-700/60 px-1.5 py-0.5 text-[10px] text-zinc-400">
                    {it.type}
                  </span>
                  <span className="text-sm font-medium text-zinc-200">{it.title}</span>
                  <span className="ml-auto text-xs text-zinc-600">{it.created_at}</span>
                </div>
                <pre className="whitespace-pre-wrap font-sans text-xs leading-relaxed text-zinc-400">
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
  tint: 'violet' | 'sky' | 'emerald'
  children: React.ReactNode
}) {
  const cls = {
    violet: 'border-violet-500/30 bg-violet-500/5 text-violet-200',
    sky: 'border-sky-500/30 bg-sky-500/5 text-sky-200',
    emerald: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-200',
  }[tint]
  return (
    <section className="mb-4 rounded-lg border p-3">
      <div className={`mb-2 text-xs font-semibold uppercase tracking-wide ${cls.split(' ').pop()}`}>
        {label}
      </div>
      {children}
    </section>
  )
}
