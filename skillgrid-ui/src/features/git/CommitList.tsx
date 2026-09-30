import type { GitCommit } from './api'

function timeAgo(ts: string): string {
  const t = new Date(ts).getTime()
  if (Number.isNaN(t)) return ts
  const s = Math.floor((Date.now() - t) / 1000)
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.floor(s / 60)}m`
  if (s < 86400) return `${Math.floor(s / 3600)}h`
  return `${Math.floor(s / 86400)}d`
}

function shortSha(sha: string): string {
  return sha.slice(0, 7)
}

// CommitList — the commit log (newest first): SHA (copyable), message, author,
// +/- stats. Selecting a commit opens its detail.
export function CommitList({
  commits,
  selectedSha,
  onSelect,
}: {
  commits: GitCommit[]
  selectedSha: string | null
  onSelect: (sha: string) => void
}) {
  return (
    <div className="space-y-1.5">
      {commits.map((c) => {
        const sel = c.sha === selectedSha
        return (
          <button
            key={c.sha}
            type="button"
            onClick={() => onSelect(c.sha)}
            className={`flex w-full items-center gap-3 rounded border px-3 py-2 text-left transition-colors ${
              sel ? 'border-accent/70 bg-accent/10' : 'border-edge bg-card hover:border-accent/40'
            }`}
          >
            <code
              className="shrink-0 rounded bg-edge/50 px-1.5 py-0.5 text-[11px] text-ink-3"
              title={c.sha}
              onClick={(e) => {
                e.stopPropagation()
                void navigator.clipboard?.writeText(c.sha)
              }}
            >
              {shortSha(c.sha)}
            </code>
            <div className="min-w-0 flex-1">
              <div className="truncate text-[13px] text-ink" title={c.message}>
                {c.message}
              </div>
              <div className="text-[11px] text-ink-5">
                {c.author} · {timeAgo(c.date)}
              </div>
            </div>
            <span className="shrink-0 text-[11px] tabular-nums">
              <span className="text-accent">+{c.additions}</span>{' '}
              <span className="text-danger">-{c.deletions}</span>
            </span>
          </button>
        )
      })}
    </div>
  )
}
