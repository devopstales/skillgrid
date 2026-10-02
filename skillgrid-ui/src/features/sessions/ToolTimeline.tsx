import { memo, useEffect, useRef, useState } from 'react'
import {
  fetchEvents,
  fetchSessionEvents,
  matchesFilter,
  SessionsError,
  type ObservationRow,
  type ToolEvent,
  type ToolFilter,
} from './api'
import { AgentBadge, McpBadge } from './AgentBadge'

const ACTIONS = ['file_read', 'file_write', 'command_exec', 'tool_use']
const SINCE = [
  { v: '', label: 'any time' },
  { v: '15m', label: '15 min' },
  { v: '1h', label: '1 hour' },
  { v: '24h', label: '24 hours' },
  { v: '7d', label: '7 days' },
  { v: '30d', label: '30 days' },
]

const ACTION_ICON: Record<string, string> = {
  file_read: 'R',
  file_write: 'W',
  command_exec: '$',
  tool_use: '⚙',
}

// RESULT_STYLE colors the result chip; policy decisions (Phase 5) share it.
const RESULT_STYLE: Record<string, string> = {
  error: 'text-danger',
  failed: 'text-danger',
  blocked: 'text-danger',
  warned: 'text-warn',
  guided: 'text-info',
}

const POLICY_RESULTS = new Set(['blocked', 'warned', 'guided'])

function clock(ts: string): string {
  const d = new Date(ts)
  return Number.isNaN(d.getTime()) ? ts : d.toLocaleTimeString([], { hour12: false })
}

type TimelineItem =
  | { kind: 'tool'; key: string; ts: string; tool: ToolEvent }
  | { kind: 'observation'; key: string; ts: string; row: ObservationRow }

function mergeTimeline(tools: ToolEvent[], observations: ObservationRow[]): TimelineItem[] {
  const items: TimelineItem[] = [
    ...tools.map((t) => ({ kind: 'tool' as const, key: `tool-${t.id}`, ts: t.ts, tool: t })),
    ...observations.map((o) => ({
      kind: 'observation' as const,
      key: `obs-${o.id}`,
      ts: o.created_at,
      row: o,
    })),
  ]
  return items.sort((a, b) => b.ts.localeCompare(a.ts)).slice(0, 500)
}

// ToolTimeline — the live tool-call stream for one session (sessionId) or the
// whole project (no sessionId). The page owns the single SSE connection and
// hands live frames in through registerTool; frames are filtered client-side
// with the same rules the server applies to the query.
export function ToolTimeline({
  sessionId,
  agents,
  live,
  registerTool,
  registerObservation,
}: {
  sessionId?: string | null
  agents: string[]
  live: boolean
  registerTool: (h: ((e: ToolEvent) => void) | null) => void
  registerObservation: (h: ((o: ObservationRow) => void) | null) => void
}) {
  const [filter, setFilter] = useState<ToolFilter>({})
  const [events, setEvents] = useState<ToolEvent[]>([])
  const [observations, setObservations] = useState<ObservationRow[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const timeline = mergeTimeline(events, sessionId ? observations : [])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    const t = setTimeout(() => {
      const load = sessionId
        ? fetchSessionEvents(sessionId, filter).then((r) => ({
            events: r.events ?? [],
            observations: r.observations ?? [],
          }))
        : fetchEvents(filter).then((r) => ({ events: r.events ?? [], observations: [] as ObservationRow[] }))
      load
        .then((r) => {
          if (!cancelled) {
            setEvents(r.events)
            setObservations(r.observations)
          }
        })
        .catch((e) => {
          if (!cancelled) setError(e instanceof SessionsError ? e.message : 'failed to load tool calls')
        })
        .finally(() => {
          if (!cancelled) setLoading(false)
        })
    }, 200)
    return () => {
      cancelled = true
      clearTimeout(t)
    }
  }, [sessionId, filter])

  const onToolRef = useRef<(e: ToolEvent) => void>(() => {})
  onToolRef.current = (e) => {
    if (['session_start', 'session_end', 'commit'].includes(e.action)) return
    if (!matchesFilter(e, { ...filter, session: sessionId ?? undefined })) return
    if (filter.since) return // a bounded window is a historical query, not a tail
    setEvents((prev) => (prev.some((p) => p.id === e.id) ? prev : [e, ...prev].slice(0, 500)))
  }
  useEffect(() => {
    registerTool((e) => onToolRef.current(e))
    return () => registerTool(null)
  }, [registerTool])

  const onObservationRef = useRef<(o: ObservationRow) => void>(() => {})
  onObservationRef.current = (o) => {
    if (!sessionId) return
    if (filter.since) return
    setObservations((prev) => (prev.some((p) => p.id === o.id) ? prev : [o, ...prev].slice(0, 500)))
  }
  useEffect(() => {
    registerObservation((o) => onObservationRef.current(o))
    return () => registerObservation(null)
  }, [registerObservation])

  return (
    <div className="flex h-full flex-col font-mono">
      <div className="flex flex-wrap items-center gap-2 border-b border-edge-soft px-4 py-2">
        <span className={`text-[12px] ${live ? 'text-accent' : 'text-ink-6'}`}>
          {live ? '● live' : '○ offline'}
        </span>
        <span className="text-[12px] text-ink-5">
          {sessionId ? 'session tool calls' : 'all agents, all sessions'} · {timeline.length}
        </span>
        <ToolFilterBar value={filter} onChange={setFilter} agents={agents} showAgent={!sessionId} />
      </div>
      <div
        className="min-h-0 flex-1 overflow-y-auto px-4 py-2"
        role="log"
        aria-live="polite"
        aria-label={sessionId ? 'Tool calls and observations' : 'Tool calls'}
      >
        {error ? (
          <div className="rounded border border-danger-ink bg-danger/10 p-3 text-[12px] text-danger">{error}</div>
        ) : timeline.length === 0 ? (
          <div className="p-8 text-center text-[12px] text-ink-5">
            {loading
              ? 'Loading…'
              : 'No tool calls recorded. Cursor, OpenCode, and Kilo hooks post every call here while skillgrid serve is running.'}
          </div>
        ) : (
          <ul className="divide-y divide-edge-soft">
            {timeline.map((item) =>
              item.kind === 'tool' ? (
                <ToolRow key={item.key} e={item.tool} showAgent={!sessionId} />
              ) : (
                <ObservationRowView key={item.key} row={item.row} />
              ),
            )}
          </ul>
        )}
      </div>
    </div>
  )
}

const ObservationRowView = memo(function ObservationRowView({ row }: { row: ObservationRow }) {
  return (
    <li className="animate-stream-in">
      <a
        href={`/mnemonic/memories?id=${row.id}`}
        className="flex w-full items-center gap-2 py-1.5 text-left text-[12px] hover:bg-card"
        data-testid="observation-row"
      >
        <span className="w-16 shrink-0 tabular-nums text-ink-6">{clock(row.created_at)}</span>
        <span className="w-4 shrink-0 text-center text-accent-light" title="observation" aria-hidden="true">
          ◆
        </span>
        <span className="shrink-0 rounded bg-edge/60 px-1.5 text-[10px] uppercase text-ink-4">{row.type}</span>
        <span className="min-w-0 flex-1 truncate font-medium text-ink-2">{row.title}</span>
      </a>
    </li>
  )
})

const ToolRow = memo(function ToolRow({ e, showAgent }: { e: ToolEvent; showAgent: boolean }) {
  const [open, setOpen] = useState(false)
  const target = e.command || e.path
  const resultCls = RESULT_STYLE[e.result] ?? 'text-ink-6'
  return (
    <li className="animate-stream-in">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 py-1.5 text-left text-[12px] hover:bg-card"
        data-testid="tool-row"
      >
        <span className="w-16 shrink-0 tabular-nums text-ink-6">{clock(e.ts)}</span>
        <span className="w-4 shrink-0 text-center text-accent-light" title={e.action}>
          {ACTION_ICON[e.action] ?? '•'}
        </span>
        {showAgent && <AgentBadge agent={e.agent} />}
        <span className="shrink-0 font-medium text-ink-2">{e.tool || e.action}</span>
        {e.mcp && <McpBadge />}
        {e.sensitive && (
          <span className="shrink-0 text-[10px] text-warn" title="sensitive path: content not stored">
            sensitive
          </span>
        )}
        <span className="min-w-0 flex-1 truncate text-ink-4" title={target}>
          {target}
        </span>
        {e.result && e.result !== 'success' && <span className={`shrink-0 text-[11px] ${resultCls}`}>{e.result}</span>}
      </button>
      {POLICY_RESULTS.has(e.result) && e.preview && (
        <div className={`mb-1 ml-20 truncate text-[11px] ${resultCls}`} data-testid="policy-note" title={e.preview}>
          policy {e.preview}
        </div>
      )}
      {open && (
        <div className="mb-2 ml-20 rounded border border-edge bg-inset p-2 text-[11px] text-ink-4">
          <div>
            session <span className="text-ink-3">{e.sessionId}</span> · #{e.sequence} · {e.action} · {e.ts}
          </div>
          {e.path && <div>path: {e.path}</div>}
          {e.command && <div>command: {e.command}</div>}
          {e.preview && <pre className="mt-1 max-h-48 overflow-auto whitespace-pre-wrap text-ink-3">{e.preview}</pre>}
        </div>
      )}
    </li>
  )
})

// ToolFilterBar — Gryph-style query: agent, action, tool, file glob, command
// glob, since. Globs use * / ** / ? like `gryph query`.
export function ToolFilterBar({
  value,
  onChange,
  agents,
  showAgent = true,
}: {
  value: ToolFilter
  onChange: (f: ToolFilter) => void
  agents: string[]
  showAgent?: boolean
}) {
  const set = (k: keyof ToolFilter, v: string) => onChange({ ...value, [k]: v || undefined })
  const active = Object.values(value).some((v) => v)
  const input =
    'rounded border border-edge-soft bg-inset px-2 py-1 text-[12px] text-ink-3 placeholder:text-ink-6 focus:border-accent focus:outline-none'
  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="tool-filter-bar">
      {showAgent && (
        <select aria-label="agent" value={value.agent ?? ''} onChange={(e) => set('agent', e.target.value)} className={input}>
          <option value="">all agents</option>
          {agents.map((a) => (
            <option key={a} value={a}>
              {a}
            </option>
          ))}
        </select>
      )}
      <select aria-label="action" value={value.action ?? ''} onChange={(e) => set('action', e.target.value)} className={input}>
        <option value="">all actions</option>
        {ACTIONS.map((a) => (
          <option key={a} value={a}>
            {a}
          </option>
        ))}
      </select>
      <input aria-label="tool" placeholder="tool" value={value.tool ?? ''} onChange={(e) => set('tool', e.target.value)} className={`${input} w-24`} />
      <input aria-label="file" placeholder="file glob src/**" value={value.file ?? ''} onChange={(e) => set('file', e.target.value)} className={`${input} w-36`} />
      <input aria-label="command" placeholder="command npm *" value={value.command ?? ''} onChange={(e) => set('command', e.target.value)} className={`${input} w-36`} />
      <select aria-label="since" value={value.since ?? ''} onChange={(e) => set('since', e.target.value)} className={input}>
        {SINCE.map((s) => (
          <option key={s.v} value={s.v}>
            {s.label}
          </option>
        ))}
      </select>
      {active && (
        <button type="button" onClick={() => onChange({})} className="rounded border border-edge-soft px-2 py-1 text-[12px] text-ink-4 hover:text-ink-2">
          Clear
        </button>
      )}
    </div>
  )
}
