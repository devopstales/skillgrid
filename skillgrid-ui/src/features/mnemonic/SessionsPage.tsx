import { useEffect, useMemo, useState } from 'react'
import {
  fetchAudit,
  fetchHubStatus,
  fetchSessionSummary,
  fetchSessions,
  fetchSnapshots,
  openActivityStream,
  SessionsError,
  type AuditEntry,
  type ChangeSnapshot,
  type Checkpoint,
  type HandoffRef,
  type MnemonicSession,
  type SessionSummary,
} from '../sessions/api'
import { ActivityPane } from '../sessions/ActivityPane'
import { ChangesPane } from '../sessions/ChangesPane'
import { HandoffsPane } from '../sessions/HandoffsPane'
import { AuditTab } from '../sessions/AuditTab'

type Tab = 'activity' | 'changes' | 'handoffs' | 'audit'

// SessionsPage — the single "what happened" home (sessions-activity-unification).
// Left: the project's sessions. Right: a session-scoped activity feed (global
// when nothing is selected) as the default tab, with the Handoff Hub's change
// log + handoffs folded in as tabs, and the hash-chained audit trail demoted to
// a secondary tab.
export function SessionsPage() {
  const [sessions, setSessions] = useState<MnemonicSession[]>([])
  const [tab, setTab] = useState<Tab>('activity')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  // Change-log + handoff data (Handoff Hub), shared across tabs.
  const [snapshots, setSnapshots] = useState<ChangeSnapshot[]>([])
  const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([])
  const [refs, setRefs] = useState<HandoffRef[]>([])
  const [latest, setLatest] = useState<ChangeSnapshot | null>(null)
  const [live, setLive] = useState(false)

  // Audit trail (secondary tab).
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [chainValid, setChainValid] = useState(true)

  // Selected session header.
  const [summary, setSummary] = useState<SessionSummary | null>(null)
  const [error, setError] = useState('')

  const selected = sessions.find((s) => s.id === selectedId) ?? null

  // Sessions list.
  useEffect(() => {
    fetchSessions()
      .then((r) => setSessions(r.sessions))
      .catch((e) => setError(e instanceof SessionsError ? e.message : 'failed to load sessions'))
  }, [])

  // Change log + hub status + live snapshot stream (event: snapshot).
  useEffect(() => {
    Promise.all([fetchSnapshots(200), fetchHubStatus()])
      .then(([snaps, status]) => {
        setSnapshots(snaps.snapshots ?? [])
        setCheckpoints(status.checkpoints ?? [])
        setRefs(status.handoff_refs ?? [])
        setLatest(status.latest_snapshot ?? null)
      })
      .catch(() => {})
    const close = openActivityStream({
      onReady: () => setLive(true),
      onSnapshot: (s) => {
        setSnapshots((prev) => {
          if (prev.some((p) => p.commit === s.commit)) return prev
          const ns: ChangeSnapshot = {
            id: Date.now(),
            branch: s.branch,
            commit: s.commit,
            commitShort: s.commitShort,
            subject: s.subject,
            author: s.author,
            committedAt: s.committedAt,
            changedFiles: '',
          }
          setLatest(ns)
          return [ns, ...prev].slice(0, 200)
        })
      },
      onError: () => setLive(false),
    })
    return close
  }, [])

  // Audit trail (lazy: only when the Audit tab is opened).
  useEffect(() => {
    if (tab !== 'audit') return
    fetchAudit(200)
      .then((a) => {
        setEntries(a.entries)
        setChainValid(a.chain_valid)
      })
      .catch(() => {})
  }, [tab])

  // Selected session header (summary + status + ended_at).
  useEffect(() => {
    if (!selectedId) {
      setSummary(null)
      return
    }
    fetchSessionSummary(selectedId)
      .then(setSummary)
      .catch(() => setSummary(null))
  }, [selectedId])

  const checkpointByCommit = useMemo(() => {
    const m = new Map<string, Checkpoint>()
    for (const c of checkpoints) m.set(c.commit, c)
    return m
  }, [checkpoints])

  const openCheckpoints = useMemo(
    () => checkpoints.filter((c) => c.status === 'open' || c.status === 'stale'),
    [checkpoints],
  )

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
        <h1 className="text-sm font-semibold text-zinc-100">Sessions</h1>
        <span className="text-xs text-zinc-500">
          {sessions.length} sessions · one activity home
        </span>
        <span className="ml-auto flex items-center gap-2 text-xs">
          <span className={live ? 'text-emerald-500' : 'text-zinc-600'}>
            {live ? '● live' : '○ offline'}
          </span>
          <nav className="flex gap-1 rounded-lg border border-edge p-0.5">
            {(['activity', 'changes', 'handoffs', 'audit'] as Tab[]).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => setTab(t)}
                className={`rounded-md px-2.5 py-1 text-xs capitalize ${
                  tab === t
                    ? 'bg-accent/20 text-zinc-100'
                    : 'text-zinc-500 hover:text-zinc-200'
                }`}
              >
                {t}
              </button>
            ))}
          </nav>
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
                  <SessionRow
                    key={s.id}
                    s={s}
                    active={s.id === selectedId}
                    onSelect={setSelectedId}
                  />
                ))}
              </ul>
            )}
          </div>
        </aside>

        {/* Detail pane: header + tab content */}
        <main className="flex min-h-0 flex-1 flex-col">
          {(tab === 'activity' && selected) && (
            <SessionHeader summary={summary} session={selected} />
          )}
          <div className="min-h-0 flex-1">
            {tab === 'activity' && <ActivityPane sessionId={selectedId} />}
            {tab === 'changes' && (
              <ChangesPane
                snapshots={snapshots}
                checkpointByCommit={checkpointByCommit}
                openCheckpoints={openCheckpoints}
                latest={latest}
              />
            )}
            {tab === 'handoffs' && <HandoffsPane checkpoints={checkpoints} refs={refs} />}
            {tab === 'audit' && <AuditTab entries={entries} chainValid={chainValid} />}
          </div>
        </main>
      </div>
    </div>
  )
}

function SessionHeader({
  summary,
  session,
}: {
  summary: SessionSummary | null
  session: MnemonicSession
}) {
  const ended = summary?.ended_at || session.ended_at
  return (
    <div className="border-b border-edge px-6 py-3">
      <div className="flex items-center gap-2">
        <StatusDot status={session.status} />
        <span className="text-sm font-medium text-zinc-200">{session.title}</span>
        <span className="text-[11px] text-zinc-500">
          {session.started_at.slice(0, 10)}
          {ended ? ` → ${ended.slice(0, 10)}` : ''}
        </span>
      </div>
      {summary && summary.summary && (
        <p className="mt-2 line-clamp-3 whitespace-pre-line text-xs text-zinc-400">
          {summary.summary}
        </p>
      )}
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
