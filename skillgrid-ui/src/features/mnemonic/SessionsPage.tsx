import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
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
  type ToolEvent,
} from '../sessions/api'
import { ActivityPane } from '../sessions/ActivityPane'
import { AuditTab } from '../sessions/AuditTab'
import { AgentBadge } from '../sessions/AgentBadge'
import { ToolTimeline } from '../sessions/ToolTimeline'
import { formatCost, formatTokens } from '../sessions/format'

type Tab = 'tools' | 'memory' | 'audit'
const TAB_LABEL: Record<Tab, string> = { tools: 'tool calls', memory: 'memory', audit: 'audit' }

// A session is "live" when its last recorded action is this recent.
const LIVE_WINDOW_MS = 2 * 60 * 1000

// SessionsPage — the single "what happened" home. Left: every session, tagged
// with the harness that owns it (cursor, opencode, kilo) and its last tool.
// Right: the selected session's live tool-call timeline (all sessions when
// none is selected), with memory activity and the audit trail as secondary
// tabs. One SSE connection feeds the timeline, the memory feed, and the list.
function sessionFromLocation(): string | null {
  if (typeof window === 'undefined') return null
  const q = new URLSearchParams(window.location.search)
  const id = (q.get('id') ?? q.get('session'))?.trim()
  return id ? id : null
}

function writeSessionToLocation(id: string | null) {
  if (typeof window === 'undefined') return
  const url = new URL(window.location.href)
  url.searchParams.delete('session')
  if (id) url.searchParams.set('id', id)
  else url.searchParams.delete('id')
  window.history.replaceState(window.history.state, '', url)
}

function isLive(s: MnemonicSession, now: number): boolean {
  if (s.status !== 'active' || !s.last_active) return false
  const t = new Date(s.last_active).getTime()
  return !Number.isNaN(t) && now - t < LIVE_WINDOW_MS
}

export function SessionsPage() {
  const [sessions, setSessions] = useState<MnemonicSession[]>([])
  const [tab, setTab] = useState<Tab>('tools')
  const [selectedId, setSelectedIdState] = useState<string | null>(sessionFromLocation)
  const [live, setLive] = useState(false)
  const [now, setNow] = useState(() => Date.now())

  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [chainValid, setChainValid] = useState(true)
  const [auditError, setAuditError] = useState('')

  const [summary, setSummary] = useState<SessionSummary | null>(null)
  const [error, setError] = useState('')

  const setSelectedId = useCallback((id: string | null) => {
    setSelectedIdState(id)
    writeSessionToLocation(id)
  }, [])

  const activityHandlerRef = useRef<((e: ActivityEvent) => void) | null>(null)
  const registerActivity = useCallback((h: ((e: ActivityEvent) => void) | null) => {
    activityHandlerRef.current = h
  }, [])
  const toolHandlerRef = useRef<((e: ToolEvent) => void) | null>(null)
  const registerTool = useCallback((h: ((e: ToolEvent) => void) | null) => {
    toolHandlerRef.current = h
  }, [])

  const selected = sessions.find((s) => s.id === selectedId) ?? null
  const agents = useMemo(
    () => Array.from(new Set(sessions.map((s) => s.agent).filter((a): a is string => !!a))).sort(),
    [sessions],
  )

  const loadSessions = useCallback(() => {
    fetchSessions()
      .then((r) => setSessions(r.sessions))
      .catch((e) => setError(e instanceof SessionsError ? e.message : 'failed to load sessions'))
  }, [])
  useEffect(loadSessions, [loadSessions])

  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 15_000)
    return () => clearInterval(t)
  }, [])

  // A tool frame updates its session row in place; an unknown session (a
  // harness that just started) reloads the list so it appears without a refresh.
  const onToolForList = useRef<(e: ToolEvent) => void>(() => {})
  onToolForList.current = (e) => {
    const known = sessions.some((s) => s.id === e.sessionId)
    if (!known || e.newSession) {
      loadSessions()
      return
    }
    if (['session_start', 'session_end', 'commit'].includes(e.action)) {
      if (e.action === 'session_end') loadSessions()
      return
    }
    const policy = e.result === 'blocked' || e.result === 'warned' || e.result === 'guided'
    setSessions((prev) => {
      const next = prev.map((s) =>
        s.id === e.sessionId
          ? {
              ...s,
              agent: s.agent || e.agent,
              last_tool: e.tool || s.last_tool,
              last_active: e.ts,
              tool_calls: (s.tool_calls ?? 0) + (policy ? 0 : 1),
              errors: (s.errors ?? 0) + (e.result === 'error' ? 1 : 0),
              blocked_actions: (s.blocked_actions ?? 0) + (e.result === 'blocked' ? 1 : 0),
              policy_decisions: (s.policy_decisions ?? 0) + (policy ? 1 : 0),
            }
          : s,
      )
      return next.sort((a, b) => (b.last_active ?? b.started_at).localeCompare(a.last_active ?? a.started_at))
    })
    setNow(Date.now())
  }

  useEffect(() => {
    const close = openActivityStream({
      onReady: () => setLive(true),
      onActivity: (e) => activityHandlerRef.current?.(e),
      onTool: (e) => {
        onToolForList.current(e)
        toolHandlerRef.current?.(e)
      },
      onError: () => setLive(false),
    })
    return close
  }, [])

  useEffect(() => {
    if (tab !== 'audit') return
    setAuditError('')
    fetchAudit(200)
      .then((a) => {
        setEntries(a.entries)
        setChainValid(a.chain_valid)
      })
      .catch((e) => setAuditError(e instanceof SessionsError ? e.message : 'failed to load audit trail'))
  }, [tab])

  useEffect(() => {
    if (!selectedId) {
      setSummary(null)
      return
    }
    fetchSessionSummary(selectedId)
      .then(setSummary)
      .catch(() => setSummary(null))
  }, [selectedId])

  const liveCount = sessions.filter((s) => isLive(s, now)).length

  return (
    <div className="flex h-full min-h-0 flex-col font-mono">
      <div className="flex items-center gap-2 border-b border-edge-soft px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Sessions</h1>
        <span className="text-[12px] text-ink-5">
          {sessions.length} sessions · {liveCount} live
        </span>
        {selectedId && (
          <button
            type="button"
            onClick={() => setSelectedId(null)}
            className="rounded border border-edge-soft px-2 py-0.5 text-[11px] text-ink-4 hover:text-ink-2"
          >
            all sessions
          </button>
        )}
        <span className="ml-auto flex items-center gap-2 text-[12px]">
          <span className={live ? 'text-accent' : 'text-ink-6'}>{live ? '● live' : '○ offline'}</span>
          <nav className="flex gap-1 rounded border border-edge p-0.5">
            {(['tools', 'memory', 'audit'] as Tab[]).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => setTab(t)}
                className={`rounded px-2.5 py-1 text-[12px] ${
                  tab === t ? 'bg-accent/15 text-accent' : 'text-ink-5 hover:text-ink-2'
                }`}
              >
                {TAB_LABEL[t]}
              </button>
            ))}
          </nav>
        </span>
      </div>

      {error && (
        <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">{error}</div>
      )}

      <div className="flex min-h-0 flex-1">
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
                    live={isLive(s, now)}
                    onSelect={(id) => setSelectedId(id === selectedId ? null : id)}
                  />
                ))}
              </ul>
            )}
          </div>
        </aside>

        <main className="flex min-h-0 flex-1 flex-col">
          {selected && tab !== 'audit' && (
            <SessionHeader summary={summary} session={selected} live={isLive(selected, now)} />
          )}
          <div className="min-h-0 flex-1">
            {tab === 'tools' && (
              <ToolTimeline sessionId={selectedId} agents={agents} live={live} registerTool={registerTool} />
            )}
            {tab === 'memory' && (
              <ActivityPane sessionId={selectedId} live={live} registerActivity={registerActivity} />
            )}
            {tab === 'audit' && <AuditTab entries={entries} chainValid={chainValid} error={auditError} />}
          </div>
        </main>
      </div>
    </div>
  )
}

function SessionHeader({
  summary,
  session,
  live,
}: {
  summary: SessionSummary | null
  session: MnemonicSession
  live: boolean
}) {
  const ended = summary?.ended_at || session.ended_at
  const tokens = (session.input_tokens ?? 0) + (session.output_tokens ?? 0)
  return (
    <div className="border-b border-edge px-6 py-3" data-testid="session-header">
      <div className="flex flex-wrap items-center gap-2">
        <StatusDot status={session.status} live={live} />
        <AgentBadge agent={session.agent} testId="session-header-agent" />
        <span className="text-[13px] font-medium text-ink-2">{session.title}</span>
        <span className="text-[11px] text-ink-5">
          {session.started_at.slice(0, 16).replace('T', ' ')}
          {ended ? ` → ${ended.slice(0, 16).replace('T', ' ')}` : ''}
        </span>
        {session.last_tool && (
          <span className="ml-auto text-[12px] text-ink-4">
            {live ? 'now' : 'last'}: <span className="text-accent-light">{session.last_tool}</span>
          </span>
        )}
      </div>
      <div className="mt-1.5 flex flex-wrap gap-3 text-[11px] text-ink-5">
        <span>{session.tool_calls ?? 0} calls</span>
        <span>{session.files_read ?? 0} reads</span>
        <span>{session.files_written ?? 0} writes</span>
        <span>{session.commands_exec ?? 0} commands</span>
        {(session.errors ?? 0) > 0 && <span className="text-danger">{session.errors} errors</span>}
        {(session.blocked_actions ?? 0) > 0 && <span className="text-danger">{session.blocked_actions} blocked</span>}
        {(session.policy_decisions ?? 0) > 0 && (
          <span className="text-warn" data-testid="policy-count" title="block, warn, and guide decisions">
            {session.policy_decisions} policy
          </span>
        )}
        <span>{session.memory_count} memories</span>
        <span title={session.model || undefined}>
          {formatTokens(tokens)} tokens · {formatCost(session.cost_usd)}
        </span>
      </div>
      {summary && summary.summary && (
        <p className="mt-2 line-clamp-3 whitespace-pre-line text-[12px] text-ink-4">{summary.summary}</p>
      )}
    </div>
  )
}

function SessionRow({
  s,
  active,
  live,
  onSelect,
}: {
  s: MnemonicSession
  active: boolean
  live: boolean
  onSelect: (id: string) => void
}) {
  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(s.id)}
        aria-pressed={active}
        className={[
          'flex w-full cursor-pointer items-center gap-2 border-l-2 px-3 py-2.5 text-left',
          active ? 'border-accent bg-accent/10' : 'border-transparent hover:bg-card',
        ].join(' ')}
      >
        <StatusDot status={s.status} live={live} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <AgentBadge agent={s.agent} />
            <span className="truncate text-[13px] font-medium text-ink-2">{s.title}</span>
          </div>
          <div className="mt-0.5 flex items-center gap-2 text-[11px] text-ink-5">
            <span>{(s.last_active || s.started_at).slice(5, 16).replace('T', ' ')}</span>
            {s.last_tool && <span className="truncate text-accent-light">{s.last_tool}</span>}
            <span className="rounded bg-edge/60 px-1 text-[10px] text-ink-4">{s.tool_calls ?? 0} calls</span>
            <span className="rounded bg-edge/60 px-1 text-[10px] text-ink-4">{s.memory_count} mem</span>
            {(s.errors ?? 0) > 0 && <span className="text-danger">{s.errors} err</span>}
            {s.cost_usd != null && <span>{formatCost(s.cost_usd)}</span>}
          </div>
        </div>
      </button>
    </li>
  )
}

function StatusDot({ status, live }: { status: string; live?: boolean }) {
  const cls = live ? 'bg-ok animate-pulse' : status === 'active' ? 'bg-accent' : status === 'ended' ? 'bg-ink-5' : 'bg-warn'
  return <span className={`h-2 w-2 shrink-0 rounded-full ${cls}`} title={live ? 'live' : status} />
}
