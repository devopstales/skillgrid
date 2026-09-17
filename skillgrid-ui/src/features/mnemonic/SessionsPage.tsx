import { useEffect, useState } from 'react'
import {
  fetchAudit,
  fetchSessions,
  MnemonicError,
  type AuditEntry,
  type MnemonicSession,
} from './api'

// SessionsPage is the session browser: the project's sessions (with memory
// counts + summary presence) on the left, and the hash-chained audit trail on
// the right (the append-only memory change log, with a chain-validity check).

export function SessionsPage() {
  const [sessions, setSessions] = useState<MnemonicSession[]>([])
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [chainValid, setChainValid] = useState(true)
  const [error, setError] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setError('')
    Promise.all([fetchSessions(), fetchAudit(200)])
      .then(([s, a]) => {
        if (cancelled) return
        setSessions(s.sessions)
        setEntries(a.entries)
        setChainValid(a.chain_valid)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof MnemonicError ? e.message : 'failed to load')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const selected = sessions.find((s) => s.id === selectedId) ?? null

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
        <h1 className="text-sm font-semibold text-zinc-100">Sessions</h1>
        <span className="text-xs text-zinc-500">{sessions.length} sessions</span>
        <span className="ml-auto text-xs">
          <ChainBadge valid={chainValid} />
        </span>
      </div>

      {error && (
        <div className="border-b border-red-500/30 bg-red-500/10 px-4 py-2 text-xs text-red-300">
          {error}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Session list */}
        <aside className="flex w-96 shrink-0 flex-col border-r border-edge">
          <div className="min-h-0 flex-1 overflow-y-auto">
            {sessions.length === 0 ? (
              <div className="p-4 text-xs text-zinc-600">No sessions recorded.</div>
            ) : (
              <ul className="divide-y divide-edge">
                {sessions.map((s) => (
                  <SessionRow key={s.id} s={s} active={s.id === selectedId} onSelect={setSelectedId} />
                ))}
              </ul>
            )}
          </div>
        </aside>

        {/* Audit trail */}
        <main className="min-h-0 flex-1 overflow-y-auto">
          <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
            <h2 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
              Audit trail
            </h2>
            <span className="text-xs text-zinc-600">{entries.length} entries · hash-chained</span>
          </div>
          {selected && (
            <div className="border-b border-edge px-4 py-2 text-xs text-zinc-500">
              Filtered to session <b className="text-zinc-300">{selected.title}</b> — showing all
              project entries (audit is project-scoped).
            </div>
          )}
          {entries.length === 0 ? (
            <div className="p-8 text-center text-sm text-zinc-500">
              No memory changes recorded yet.
            </div>
          ) : (
            <AuditTable entries={entries} />
          )}
        </main>
      </div>
    </div>
  )
}

function SessionRow({
  s,
  active,
  onSelect,
}: {
  s: MnemonicSession
  active: boolean
  onSelect: (id: string) => void
}) {
  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(s.id)}
        className={[
          'flex w-full items-center gap-2 px-3 py-2.5 text-left',
          active ? 'bg-accent/15' : 'hover:bg-card',
        ].join(' ')}
      >
        <StatusDot status={s.status} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-zinc-200">{s.title}</div>
          <div className="mt-0.5 flex items-center gap-2 text-[11px] text-zinc-500">
            <span>{s.started_at.slice(0, 10)}</span>
            <span className="rounded bg-zinc-700/60 px-1 text-[10px]">{s.memory_count} mem</span>
            {s.has_summary && <span className="text-emerald-400/80">summary</span>}
          </div>
        </div>
      </button>
    </li>
  )
}

function StatusDot({ status }: { status: string }) {
  const cls =
    status === 'active'
      ? 'bg-emerald-400'
      : status === 'ended'
        ? 'bg-zinc-500'
        : 'bg-amber-400'
  return <span className={`h-2 w-2 shrink-0 rounded-full ${cls}`} />
}

function ChainBadge({ valid }: { valid: boolean }) {
  return valid ? (
    <span className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-emerald-300">
      chain valid
    </span>
  ) : (
    <span className="rounded-md border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-red-300">
      chain broken
    </span>
  )
}

function AuditTable({ entries }: { entries: AuditEntry[] }) {
  return (
    <table className="w-full text-left text-xs">
      <thead className="sticky top-0 bg-background">
        <tr className="text-[10px] uppercase tracking-wide text-zinc-600">
          <th className="px-4 py-2">seq</th>
          <th className="px-4 py-2">obs</th>
          <th className="px-4 py-2">rev</th>
          <th className="px-4 py-2">time</th>
          <th className="px-4 py-2">hash</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-edge">
        {entries.map((e) => (
          <tr key={e.seq} className="font-mono text-[11px] text-zinc-400 hover:bg-card">
            <td className="px-4 py-1.5 text-zinc-600">{e.seq}</td>
            <td className="px-4 py-1.5">{e.observation_id}</td>
            <td className="px-4 py-1.5">{e.revision}</td>
            <td className="px-4 py-1.5">{e.created_at}</td>
            <td className="px-4 py-1.5">
              <span className="text-emerald-400/80">{e.hash.slice(0, 10)}</span>
              <span className="text-zinc-700">…</span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
