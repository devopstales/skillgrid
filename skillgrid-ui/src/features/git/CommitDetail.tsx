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

  if (loading) return <p className="p-4 text-[13px] text-ink-5">Loading commit…</p>
  if (error) {
    return (
      <div className="rounded border border-danger-ink bg-danger/10 p-4 text-[13px] text-danger">
        {error}
      </div>
    )
  }
  if (!commit) return null

  return (
    <div className="space-y-4">
      <div className="rounded border border-edge bg-card p-3">
        <div className="flex items-center gap-2 text-[12px]">
          <code className="rounded bg-edge/50 px-1.5 py-0.5 text-ink-3">{commit.sha.slice(0, 12)}</code>
          <span className="text-ink-5">{commit.author}</span>
          <span className="text-ink-6">{commit.date}</span>
          <span className="ml-auto tabular-nums">
            <span className="text-accent">+{commit.additions}</span>{' '}
            <span className="text-danger">-{commit.deletions}</span>
          </span>
        </div>
        <p className="mt-2 text-[13px] text-ink">{commit.message}</p>
      </div>

      {commit.files && commit.files.length > 0 && (
        <div>
          <h3 className="mb-2 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
            Changed files ({commit.files.length})
          </h3>
          <ul className="space-y-1">
            {commit.files.map((f) => (
              <li
                key={f.path}
                className="flex items-center gap-2 rounded border border-edge bg-card px-3 py-1.5 text-[12px]"
              >
                <span className="min-w-0 flex-1 truncate text-ink-3" title={f.path}>
                  {f.path}
                </span>
                <span className="shrink-0 tabular-nums text-[11px]">
                  <span className="text-accent">+{f.additions}</span>{' '}
                  <span className="text-danger">-{f.deletions}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div>
        <h3 className="mb-2 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">Diff</h3>
        <DiffViewer diff={diff} />
      </div>
    </div>
  )
}
