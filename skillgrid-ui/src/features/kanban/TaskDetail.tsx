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
    <div className="fixed inset-0 z-40">
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
        aria-hidden
      />
      <aside className="absolute right-0 top-0 flex h-full w-full max-w-md flex-col border-l border-edge bg-card shadow-2xl">
        <div className="flex items-center justify-between border-b border-edge px-4 py-3">
          <span className="font-mono text-xs text-zinc-500">{task}</span>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md px-2 py-1 text-zinc-500 hover:bg-edge/40 hover:text-zinc-300"
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4">
          {loading && <p className="text-sm text-zinc-500">Loading…</p>}
          {error && (
            <p className="rounded-md border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">
              {error}
            </p>
          )}
          {t && (
            <>
              <h2 className="text-base font-semibold text-zinc-100">
                {t.title}
              </h2>
              <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
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
                      className="rounded bg-edge/40 px-1.5 py-0.5 text-[10px] text-zinc-500"
                    >
                      {l}
                    </span>
                  ))}
                </div>
              )}
              <div className="mt-4">
                <h3 className="mb-1.5 text-xs font-medium uppercase tracking-wide text-zinc-500">
                  Description
                </h3>
                <pre className="whitespace-pre-wrap rounded-md border border-edge bg-bg/40 p-3 text-xs leading-relaxed text-zinc-300">
                  {t.description || '—'}
                </pre>
              </div>
              <div className="mt-4">
                <h3 className="mb-1.5 text-xs font-medium uppercase tracking-wide text-zinc-500">
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
    <div className="rounded-md border border-edge bg-bg/40 px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-wide text-zinc-600">
        {label}
      </div>
      <div className="mt-0.5 truncate text-zinc-300">{value || '—'}</div>
    </div>
  )
}
