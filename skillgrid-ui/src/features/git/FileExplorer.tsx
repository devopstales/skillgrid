import { useEffect, useState } from 'react'
import {
  fetchFileHistory,
  fetchBlame,
  GitError,
  type GitCommit,
  type GitBlameLine,
} from './api'

// FileHistory — the commit history for one file.
function FileHistory({ path }: { path: string }) {
  const [history, setHistory] = useState<GitCommit[] | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    let cancelled = false
    setError('')
    setHistory(null)
    fetchFileHistory(path)
      .then((r) => {
        if (!cancelled) setHistory(r.history ?? [])
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof GitError ? e.message : 'failed to load history')
      })
    return () => {
      cancelled = true
    }
  }, [path])
  if (error) return <p className="text-sm text-red-300">{error}</p>
  if (!history) return <p className="text-sm text-zinc-500">Loading…</p>
  return (
    <ul className="space-y-1">
      {history.map((c) => (
        <li key={c.sha} className="flex items-center gap-2 rounded-md border border-edge bg-card px-3 py-1.5 text-xs">
          <code className="shrink-0 rounded bg-edge/50 px-1.5 py-0.5 text-zinc-300">{c.sha.slice(0, 7)}</code>
          <span className="min-w-0 flex-1 truncate text-zinc-300" title={c.message}>
            {c.message}
          </span>
          <span className="shrink-0 text-zinc-600">{c.author}</span>
        </li>
      ))}
    </ul>
  )
}

// BlameView — per-line blame for one file.
function BlameView({ path }: { path: string }) {
  const [lines, setLines] = useState<GitBlameLine[] | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    let cancelled = false
    setError('')
    setLines(null)
    fetchBlame(path)
      .then((r) => {
        if (!cancelled) setLines(r.lines ?? [])
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof GitError ? e.message : 'failed to load blame')
      })
    return () => {
      cancelled = true
    }
  }, [path])
  if (error) return <p className="text-sm text-red-300">{error}</p>
  if (!lines) return <p className="text-sm text-zinc-500">Loading…</p>
  return (
    <div className="overflow-x-auto rounded-md border border-edge bg-[#0a0a0d] text-xs">
      {lines.map((l) => (
        <div key={l.line} className="flex items-stretch gap-2 border-b border-edge/40 last:border-0">
          <span className="w-10 shrink-0 select-none px-2 text-right text-zinc-600">{l.line}</span>
          <code className="w-16 shrink-0 self-center rounded bg-edge/40 px-1.5 text-[10px] text-zinc-400" title={l.sha}>
            {l.sha.slice(0, 7)}
          </code>
          <span className="w-28 shrink-0 self-center truncate text-zinc-500" title={`${l.author}: ${l.summary}`}>
            {l.author}
          </span>
          <pre className="min-w-0 flex-1 whitespace-pre px-2 py-0.5 text-zinc-300">{l.text}</pre>
        </div>
      ))}
    </div>
  )
}

// FileExplorer — a path input + History/Blame tabs for file history & blame.
export function FileExplorer() {
  const [path, setPath] = useState('')
  const [tab, setTab] = useState<'history' | 'blame'>('history')
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <input
          value={path}
          onChange={(e) => setPath(e.target.value)}
          placeholder="file path (e.g. skillgrid-cli/main.go)"
          className="min-w-0 flex-1 rounded-md border border-edge bg-card px-3 py-1.5 text-xs text-zinc-200 focus:border-accent focus:outline-none"
        />
        <div className="flex rounded-md border border-edge p-0.5">
          {(['history', 'blame'] as const).map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={`rounded px-3 py-1 text-xs ${
                tab === t ? 'bg-accent text-white' : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {t}
            </button>
          ))}
        </div>
      </div>
      {path.trim() ? (
        tab === 'history' ? <FileHistory path={path.trim()} /> : <BlameView path={path.trim()} />
      ) : (
        <p className="text-sm text-zinc-500">Enter a file path to view its history or blame.</p>
      )}
    </div>
  )
}
