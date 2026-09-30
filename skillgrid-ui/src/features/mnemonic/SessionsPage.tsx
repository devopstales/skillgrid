import { useCallback, useEffect, useRef, useState } from 'react'
import {
  fetchAudit,
  fetchSessionSummary,
  fetchSessions,
  openActivityStream,
  SessionsError,
  type ActivityEvent,
  type AuditEntry,
  type MnemonicSession,
  type SessionSummary,
} from '../sessions/api'
import { ActivityPane } from '../sessions/ActivityPane'
import { AuditTab } from '../sessions/AuditTab'

type Tab = 'activity' | 'audit'

// SessionsPage — the single "what happened" home (sessions-activity-unification).
// Left: the project's sessions. Right: a session-scoped activity feed (global
// when nothing is selected) as the default tab, and the hash-chained audit
// trail demoted to a secondary tab.
export function SessionsPage() {
  const [sessions, setSessions] = useState<MnemonicSession[]>([])
  const [tab, setTab] = useState<Tab>('activity')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const [live, setLive] = useState(false)

  // Audit trail (secondary tab).
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [chainValid, setChainValid] = useState(true)
  const [auditError, setAuditError] = useState('')

  // Selected session header.
  const [summary, setSummary] = useState<SessionSummary | null>(null)
  const [error, setError] = useState('')

  // Single SSE subscription: the page is the only owner of /activity/stream.
  // Activity events are routed to the ActivityPane (scoped there); snapshot
  // events feed the change log; ready/error drive the `live` indicator.
  const activityHandlerRef = useRef<((e: ActivityEvent) => void) | null>(null)
  const registerActivity = useCallback(
    (h: ((e: ActivityEvent) => void) | null) => {
      activityHandlerRef.current = h
    },
    [],
  )

  const selected = sessions.find((s) => s.id === selectedId) ?? null

  // Sessions list.
  useEffect(() => {
    fetchSessions()
      .then((r) => setSessions(r.sessions))
      .catch((e) => setError(e instanceof SessionsError ? e.message : 'failed to load sessions'))
  }, [])

  // The single live stream (activity events).
  useEffect(() => {
    const close = openActivityStream({
      onReady: () => setLive(true),
      onActivity: (e) => activityHandlerRef.current?.(e),
      onError: () => setLive(false),
    })
    return close
  }, [])

  // Audit trail (lazy: only when the Audit tab is opened).
  useEffect(() => {
    if (tab !== 'audit') return
    setAuditError('')
    fetchAudit(200)
      .then((a) => {
        setEntries(a.entries)
        setChainValid(a.chain_valid)
      })
      .catch((e) =>
        setAuditError(e instanceof SessionsError ? e.message : 'failed to load audit trail'),
      )
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

  return (
    <div className="flex h-full min-h-0 flex-col font-mono">
      <div className="flex items-center gap-2 border-b border-edge-soft px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Sessions</h1>
        <span className="text-[12px] text-ink-5">{sessions.length} sessions</span>
        <span className="ml-auto flex items-center gap-2 text-[12px]">
          <span className={live ? 'text-accent' : 'text-ink-6'}>
            {live ? '● live' : '○ offline'}
          </span>
          <nav className="flex gap-1 rounded border border-edge p-0.5">
            {(['activity', 'audit'] as Tab[]).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => setTab(t)}
                className={`rounded px-2.5 py-1 text-[12px] capitalize ${
                  tab === t
                    ? 'bg-accent/15 text-accent'
                    : 'text-ink-5 hover:text-ink-2'
                }`}
              >
                {t}
              </button>
            ))}
          </nav>
        </span>
      </div>

      {error && (
        <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
          {error}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Session list */}
        <aside className="flex w-96 shrink-0 flex-col border-r border-edge-soft">
          <div className="min-h-0 flex-1 overflow-y-auto">
            {sessions.length === 0 ? (
              <div className="p-4 text-[12px] text-ink-6">No sessions recorded.</div>
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
            {tab === 'activity' && (
              <ActivityPane sessionId={selectedId} live={live} registerActivity={registerActivity} />
            )}
            {tab === 'audit' && (
              <AuditTab entries={entries} chainValid={chainValid} error={auditError} />
            )}
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
        <span className="text-[13px] font-medium text-ink-2">{session.title}</span>
        <span className="text-[11px] text-ink-5">
          {session.started_at.slice(0, 10)}
          {ended ? ` → ${ended.slice(0, 10)}` : ''}
        </span>
      </div>
      {summary && summary.summary && (
        <p className="mt-2 line-clamp-3 whitespace-pre-line text-[12px] text-ink-4">
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
          'flex w-full items-center gap-2 border-l-2 px-3 py-2.5 text-left',
          active ? 'border-accent bg-accent/10' : 'border-transparent hover:bg-card',
        ].join(' ')}
      >
        <StatusDot status={s.status} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-[13px] font-medium text-ink-2">{s.title}</div>
          <div className="mt-0.5 flex items-center gap-2 text-[11px] text-ink-5">
            <span>{s.started_at.slice(0, 10)}</span>
            <span className="rounded bg-edge/60 px-1 text-[10px] text-ink-4">{s.memory_count} mem</span>
            {s.has_summary && <span className="text-accent/80">summary</span>}
          </div>
        </div>
      </button>
    </li>
  )
}

function StatusDot({ status }: { status: string }) {
  const cls =
    status === 'active'
      ? 'bg-accent'
      : status === 'ended'
        ? 'bg-ink-5'
        : 'bg-warn'
  return <span className={`h-2 w-2 shrink-0 rounded-full ${cls}`} />
}
