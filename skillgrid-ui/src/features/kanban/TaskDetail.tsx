import { useEffect, useState } from 'react'
import { fetchDeps, fetchTask } from './api'
import type { TrackerDeps, UnifiedTask } from './types'
import { DependencyGraph } from './DependencyGraph'

// TaskDetail is the slide-in drawer: markdown description, a metadata grid,
// and the dependency mini-graph. Fetches lazily when a task is selected.
export function TaskDetail({
  task,
  provider,
  onClose,
}: {
  task: string | null
  provider?: string
  onClose: () => void
}) {
  const [full, setFull] = useState<UnifiedTask | null>(null)
  const [deps, setDeps] = useState<TrackerDeps | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!task) {
      setFull(null)
      setDeps(null)
      return
    }
    let disposed = false
    setLoading(true)
    setError('')
    Promise.all([fetchTask(task, provider), fetchDeps(task, provider)])
      .then(([t, d]) => {
        if (disposed) return
        setFull(t)
        setDeps(d)
      })
      .catch((e) => !disposed && setError(e.message))
      .finally(() => !disposed && setLoading(false))
    return () => {
      disposed = true
    }
  }, [task, provider])

  if (!task) return null
  const t = full

  return (
    <div className="fixed inset-0 z-40 flex justify-end">
      {/* Click-away scrim (transparent, keeps the board visible at 2/3). */}
      <div
        className="absolute inset-0 bg-black/20"
        onClick={onClose}
        aria-hidden
      />
      {/* 1/3-width detail sidebar. */}
      <aside className="relative right-0 top-0 flex h-full w-full max-w-[33vw] min-w-72 flex-col border-l border-edge-soft bg-card">
        <div className="flex items-center justify-between border-b border-edge-soft px-4 py-3">
          <span className="font-mono text-[12px] text-ink-5">{task}</span>
          <button
            type="button"
            onClick={onClose}
            className="rounded px-2 py-1 text-ink-5 hover:bg-edge/40 hover:text-ink-2"
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4 font-mono">
          {loading && <p className="text-[13px] text-ink-5">Loading…</p>}
          {error && (
            <p className="rounded border border-danger-ink bg-danger/10 p-3 text-[13px] text-danger">
              {error}
            </p>
          )}
          {t && (
            <>
              <h2 className="text-[15px] font-semibold text-ink">
                {t.title}
              </h2>
              <div className="mt-3 grid grid-cols-2 gap-2 text-[12px]">
                <Meta label="Status" value={t.status} />
                <Meta label="Type" value={t.type} />
                <Meta label="Priority" value={t.priority} />
                <Meta label="Assignee" value={t.assignees?.join(', ')} />
                <Meta label="Milestone" value={t.milestone} />
                <Meta label="Parent" value={t.parent} />
                <Meta label="Due date" value={t.due_date} />
                <Meta label="Provider" value={t.provider} />
                {t.ac_total ? (
                  <Meta
                    label="AC"
                    value={`${t.ac_completed ?? 0}/${t.ac_total}`}
                  />
                ) : null}
              </div>
              {t.labels && t.labels.length > 0 && (
                <div className="mt-3 flex flex-wrap gap-1.5">
                  {t.labels.map((l) => (
                    <span
                      key={l}
                      className="rounded bg-edge/40 px-1.5 py-0.5 text-[10px] text-ink-5"
                    >
                      {l}
                    </span>
                  ))}
                </div>
              )}
              {t.doc_refs && t.doc_refs.length > 0 && (
                <div className="mt-4">
                  <h3 className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.12em] text-ink-6">
                    Documents
                  </h3>
                  <ul className="space-y-1">
                    {t.doc_refs.map((ref) => (
                      <li key={ref}>
                        <a
                          href={`/docs?file=${encodeURIComponent(ref)}`}
                          className="block truncate rounded border border-edge bg-surface-2/60 px-2.5 py-1.5 text-[12px] text-ink-3 hover:border-accent/60 hover:text-accent"
                          title={ref}
                        >
                          {ref}
                        </a>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              <div className="mt-4">
                <h3 className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.12em] text-ink-6">
                  Description
                </h3>
                <pre className="whitespace-pre-wrap rounded border border-edge bg-surface-2/60 p-3 text-[12px] leading-relaxed text-ink-3">
                  {t.description || '—'}
                </pre>
              </div>
              <div className="mt-4">
                <h3 className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.12em] text-ink-6">
                  Dependencies
                </h3>
                <DependencyGraph task={{ id: t.id }} deps={deps} />
              </div>
            </>
          )}
        </div>
      </aside>
    </div>
  )
}

function Meta({ label, value }: { label: string; value?: string }) {
  return (
    <div className="rounded border border-edge bg-surface-2/60 px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-[0.1em] text-ink-6">
        {label}
      </div>
      <div className="mt-0.5 truncate text-ink-3">{value || '—'}</div>
    </div>
  )
}
