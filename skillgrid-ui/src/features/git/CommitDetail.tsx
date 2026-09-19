import { useEffect, useState } from 'react'
import { fetchCommit, fetchDiff, GitError, type GitCommit } from './api'
import { DiffViewer } from './DiffViewer'

// CommitDetail — metadata + changed-file tree + unified diff for one commit.
export function CommitDetail({ sha }: { sha: string }) {
  const [commit, setCommit] = useState<GitCommit | null>(null)
  const [diff, setDiff] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    setCommit(null)
    setDiff('')
    Promise.all([fetchCommit(sha), fetchDiff(sha)])
      .then(([c, d]) => {
        if (cancelled) return
        setCommit(c)
        setDiff(d.diff)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof GitError ? e.message : 'failed to load commit')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [sha])

  if (loading) return <p className="p-4 text-sm text-zinc-500">Loading commit…</p>
  if (error) {
    return (
      <div className="rounded-md border border-red-900/60 bg-red-950/30 p-4 text-sm text-red-300">
        {error}
      </div>
    )
  }
  if (!commit) return null

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-edge bg-card p-3">
        <div className="flex items-center gap-2 text-xs">
          <code className="rounded bg-edge/50 px-1.5 py-0.5 text-zinc-300">{commit.sha.slice(0, 12)}</code>
          <span className="text-zinc-500">{commit.author}</span>
          <span className="text-zinc-600">{commit.date}</span>
          <span className="ml-auto tabular-nums">
            <span className="text-emerald-400">+{commit.additions}</span>{' '}
            <span className="text-red-400">-{commit.deletions}</span>
          </span>
        </div>
        <p className="mt-2 text-sm text-zinc-100">{commit.message}</p>
      </div>

      {commit.files && commit.files.length > 0 && (
        <div>
          <h3 className="mb-2 text-xs font-medium uppercase tracking-widest text-zinc-500">
            Changed files ({commit.files.length})
          </h3>
          <ul className="space-y-1">
            {commit.files.map((f) => (
              <li
                key={f.path}
                className="flex items-center gap-2 rounded-md border border-edge bg-card px-3 py-1.5 text-xs"
              >
                <span className="min-w-0 flex-1 truncate text-zinc-300" title={f.path}>
                  {f.path}
                </span>
                <span className="shrink-0 tabular-nums text-[11px]">
                  <span className="text-emerald-400">+{f.additions}</span>{' '}
                  <span className="text-red-400">-{f.deletions}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div>
        <h3 className="mb-2 text-xs font-medium uppercase tracking-widest text-zinc-500">Diff</h3>
        <DiffViewer diff={diff} />
      </div>
    </div>
  )
}
