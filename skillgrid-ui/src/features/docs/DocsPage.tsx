import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchContent, fetchTree, DocsError } from './api'
import { DOCS_ROOTS, ROOT_LABEL, type DocsContent, type DocsRoot, type DocsTreeNode } from './types'
import { DocsTree } from './DocsTree'
import { MarkdownView } from './MarkdownView'
import { DocSearch } from './DocSearch'
import { FrontmatterChips } from './FrontmatterChips'
import { OnThisPage, extractToc } from './OnThisPage'

export function DocsPage() {
  const [root, setRoot] = useState<DocsRoot>('all')
  const [nodes, setNodes] = useState<DocsTreeNode[]>([])
  const [treeError, setTreeError] = useState('')
  // Pre-select from ?file= deep-link (kanban doc_refs) immediately — do not wait
  // for the tree. Content is fetched by path; the tree only highlights when ready.
  const [selected, setSelected] = useState(() => {
    try {
      return new URLSearchParams(window.location.search).get('file') ?? ''
    } catch {
      return ''
    }
  })
  const [content, setContent] = useState<DocsContent | null>(null)
  const [contentError, setContentError] = useState('')
  const [loadingContent, setLoadingContent] = useState(false)

  // load the tree when the root changes
  useEffect(() => {
    let cancelled = false
    setTreeError('')
    fetchTree(root)
      .then((res) => {
        if (!cancelled) setNodes(res.nodes ?? [])
      })
      .catch((e) => {
        if (!cancelled) setTreeError(e instanceof DocsError ? e.message : 'failed to load tree')
      })
    return () => {
      cancelled = true
    }
  }, [root])

  // Keep ?file= in the URL when the user picks a doc from the tree (shareable).
  const selectFile = useCallback((path: string) => {
    setSelected(path)
    const url = new URL(window.location.href)
    if (!url.pathname.endsWith('/docs') && url.pathname !== '/docs') return
    url.searchParams.set('file', path)
    window.history.replaceState(null, '', `${url.pathname}${url.search}`)
  }, [])

  // Keep selection in sync if the URL ?file= changes (back/forward).
  useEffect(() => {
    const onNav = () => {
      const file = new URLSearchParams(window.location.search).get('file')
      if (file) setSelected(file)
    }
    window.addEventListener('popstate', onNav)
    return () => window.removeEventListener('popstate', onNav)
  }, [])

  // load content when a doc is selected
  useEffect(() => {
    if (!selected) {
      setContent(null)
      return
    }
    let cancelled = false
    setLoadingContent(true)
    setContentError('')
    fetchContent(selected)
      .then((c) => {
        if (!cancelled) setContent(c)
      })
      .catch((e) => {
        if (!cancelled) setContentError(e instanceof DocsError ? e.message : 'failed to load doc')
      })
      .finally(() => {
        if (!cancelled) setLoadingContent(false)
      })
    return () => {
      cancelled = true
    }
  }, [selected])

  const toc = useMemo(() => (content ? extractToc(content.body) : []), [content])
  const isPlan = useMemo(
    () =>
      !!content &&
      (content.path.includes('.skillgrid/specs/') || content.path.includes('.skillgrid/archive/')) &&
      (content.path.endsWith('/briefing.md') || content.path.endsWith('/tasks.md')),
    [content],
  )

  const relatedPlanName = useMemo(() => {
    if (!isPlan || !content) return null
    const m = /(?:specs|archive)\/([^/]+)\//.exec(content.path)
    return m ? m[1] : null
  }, [isPlan, content])

  const copyMarkdown = useCallback(async () => {
    if (!content) return
    try {
      await navigator.clipboard.writeText(content.body)
    } catch {
      /* clipboard unavailable */
    }
  }, [content])

  const download = useCallback(() => {
    if (!content) return
    const blob = new Blob([content.body], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = content.path.split('/').pop() ?? 'doc.md'
    a.click()
    URL.revokeObjectURL(url)
  }, [content])

  return (
    <div className="flex h-full flex-col">
      {/* Root tabs */}
      <div className="flex items-center gap-1 border-b border-edge px-4 pt-3">
        {DOCS_ROOTS.map((r) => (
          <button
            key={r}
            type="button"
            onClick={() => setRoot(r)}
            className={[
              'rounded-t px-3 py-1.5 text-[13px] transition-colors',
              root === r ? 'bg-card text-ink' : 'text-ink-5 hover:text-ink-3',
            ].join(' ')}
          >
            {ROOT_LABEL[r]}
          </button>
        ))}
      </div>

      {/* Search bar */}
      <div className="border-b border-edge px-4 py-3">
        <DocSearch onSelect={selectFile} />
      </div>

      <div className="flex min-h-0 flex-1">
        <DocsTree nodes={nodes} root={root} selected={selected} onSelect={selectFile} />

        {/* Content pane */}
        <main className="flex min-h-0 flex-1 flex-col">
          {treeError && (
            <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
              {treeError}
            </div>
          )}

          {!selected && (
            <div className="flex flex-1 flex-col items-center justify-center gap-1 p-8 text-center">
              <span className="text-[13px] text-ink-4">Select a document from the tree to read it.</span>
              <span className="text-[12px] text-ink-6">Or use the search box above to find one.</span>
            </div>
          )}

          {selected && loadingContent && (
            <div className="flex flex-1 items-center justify-center text-[13px] text-ink-5">Loading…</div>
          )}

          {selected && contentError && (
            <div className="flex flex-1 items-center justify-center p-8">
              <span className="rounded border border-danger-ink bg-danger/10 px-4 py-2 text-[13px] text-danger">
                {contentError}
              </span>
            </div>
          )}

          {selected && content && (
            <>
              {/* Toolbar */}
              <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
                <p className="min-w-0 flex-1 truncate text-[13px] text-ink-4">{content.path}</p>
                <div className="flex shrink-0 items-center gap-1">
                  <ToolBtn label="Copy markdown" onClick={copyMarkdown}>
                    <CopyIcon />
                  </ToolBtn>
                  <ToolBtn label="Download" onClick={download}>
                    <DownloadIcon />
                  </ToolBtn>
                  <ToolBtn label="Print" onClick={() => window.print()}>
                    <PrintIcon />
                  </ToolBtn>
                </div>
              </div>

              <div className="flex min-h-0 flex-1">
                {/* Article + scroll-spy scroll container */}
                <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                    {/* ADR lifecycle banner (3b) — schema-aware decision view */}
                    {content.docType === 'adr' && content.decisionStatus && (
                      <div className="mb-4 flex items-center gap-2 rounded border border-edge-soft bg-surface-2/60 px-3 py-2 text-[12px] text-ink-2">
                        <span className="font-semibold">Decision record</span>
                        <span className="text-ink-5/70">·</span>
                        <span>
                          status: <b className="capitalize">{content.decisionStatus}</b>
                        </span>
                        <span className="ml-auto text-ink-5/70">
                          Context → Decision → Consequences
                        </span>
                      </div>
                    )}

                    {/* "View plan progress" cross-link (3.7) for SDD plan docs */}
                    {isPlan && relatedPlanName && (
                      <a
                        href={`/plans?change=${encodeURIComponent(relatedPlanName)}`}
                        className="mb-4 inline-flex items-center gap-1 rounded border border-accent/40 bg-accent/10 px-3 py-1.5 text-[12px] text-accent hover:bg-accent/20"
                      >
                        View plan progress →
                      </a>
                    )}

                  <h1 className="mb-3 text-[14px] font-semibold text-ink">{content.title}</h1>
                  <div className="mb-5">
                    <FrontmatterChips content={content} />
                  </div>
                  <MarkdownView body={content.body} securityLevel={content.mermaid?.securityLevel} />

                  {/* Related plans cross-links */}
                  {content.relatedPlans && content.relatedPlans.length > 0 && (
                    <div className="mt-8 border-t border-edge pt-4">
                      <h4 className="mb-2 text-[12px] font-semibold uppercase tracking-[0.1em] text-ink-5">
                        Related plans
                      </h4>
                      <div className="flex flex-wrap gap-2">
                        {content.relatedPlans.map((p) => (
                          <button
                            key={p}
                            type="button"
                            onClick={() => selectFile(`.skillgrid/specs/${p}/briefing.md`)}
                            className="rounded border border-edge bg-card px-2.5 py-1 text-[12px] text-ink-3 hover:border-accent/50"
                          >
                            {p}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                <OnThisPage entries={toc} />
              </div>
            </>
          )}
        </main>
      </div>
    </div>
  )
}

function ToolBtn({
  children,
  label,
  onClick,
}: {
  children: React.ReactNode
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      onClick={onClick}
      className="rounded border border-edge p-1.5 text-ink-4 hover:border-accent/50 hover:text-ink-2"
    >
      {children}
    </button>
  )
}

function CopyIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.3">
      <rect x="5" y="5" width="8" height="8" rx="1" />
      <path d="M11 5V4a1 1 0 0 0-1-1H4a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h1" />
    </svg>
  )
}
function DownloadIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.3">
      <path d="M8 2v8m0 0 3-3m-3 3L5 7" />
      <path d="M3 12v1a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1v-1" />
    </svg>
  )
}
function PrintIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.3">
      <path d="M5 6V2h6v4" />
      <rect x="3" y="6" width="10" height="6" rx="1" />
      <path d="M5 10h6v4H5z" />
    </svg>
  )
}
