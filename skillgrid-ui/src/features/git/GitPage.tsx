import { useEffect, useState } from 'react'
import { fetchCommits, GitError, type GitCommit } from './api'
import { CommitList } from './CommitList'
import { CommitDetail } from './CommitDetail'
import { FileExplorer } from './FileExplorer'

export function GitPage() {
  const [commits, setCommits] = useState<GitCommit[]>([])
  const [error, setError] = useState('')
  const [selectedSha, setSelectedSha] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setError('')
    fetchCommits(50)
      .then((r) => {
        if (cancelled) return
        setCommits(r.commits ?? [])
        if (r.commits?.length) setSelectedSha(r.commits[0].sha)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof GitError ? e.message : 'failed to load commits')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const notRepo = error && /not a git repository|503/i.test(error)

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-8">
      <div>
        <h1 className="text-2xl font-semibold text-zinc-100">Git</h1>
        <p className="mt-1 text-sm text-zinc-500">
          Read-only commit log, diff, file history &amp; blame
        </p>
      </div>

      {error ? (
        <div
          className={`rounded-md border p-4 text-sm ${
            notRepo
              ? 'border-amber-900/60 bg-amber-950/30 text-amber-300'
              : 'border-red-900/60 bg-red-950/30 text-red-300'
          }`}
        >
          {error}
          {notRepo && (
            <p className="mt-1 text-xs opacity-80">
              Run the server from a git repository root to enable the Git view.
            </p>
          )}
        </div>
      ) : (
        <div className="grid flex-1 gap-6 lg:grid-cols-2">
          <section>
            <h2 className="mb-3 text-xs font-medium uppercase tracking-widest text-zinc-500">
              Commits
            </h2>
            {commits.length === 0 ? (
              <div className="rounded-md border border-edge bg-card p-8 text-center text-sm text-zinc-500">
                No commits.
              </div>
            ) : (
              <CommitList commits={commits} selectedSha={selectedSha} onSelect={setSelectedSha} />
            )}
          </section>

          <section className="space-y-6">
            <div>
              <h2 className="mb-3 text-xs font-medium uppercase tracking-widest text-zinc-500">
                Commit detail
              </h2>
              <div className="rounded-lg border border-edge bg-card p-4">
                {selectedSha ? (
                  <CommitDetail sha={selectedSha} />
                ) : (
                  <p className="text-sm text-zinc-500">Select a commit.</p>
                )}
              </div>
            </div>

            <div>
              <h2 className="mb-3 text-xs font-medium uppercase tracking-widest text-zinc-500">
                File history &amp; blame
              </h2>
              <div className="rounded-lg border border-edge bg-card p-4">
                <FileExplorer />
              </div>
            </div>
          </section>
        </div>
      )}
    </div>
  )
}
