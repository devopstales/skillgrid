import { currentProjectName } from '../../lib/projects'

// Shared fetch + type contracts for the unified Sessions view
// (sessions-activity-unification). This merges the former features:
//   - Activity (event feed + stats + SSE)  → activity/events, /activity/stats, /activity/stream
//   - Mnemonic sessions + audit            → /mnemonic/sessions, /sessions/{id}/summary, /sessions/{id}/activity, /mnemonic/audit
//   - Harness tool calls (Gryph-style)     → /sessions/{id}/events, /events, /events/stats, SSE `tool`
//
// Project resolution mirrors the activity view (git name → first project with
// data). The live SSE reuses the single /activity/stream endpoint.

export class SessionsError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = (await res.json()) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      /* keep statusText */
    }
    throw new SessionsError(msg, res.status)
  }
  return (await res.json()) as T
}

// A deep link from another view (Observe → Teams) names the store that owns
// the session with ?store=; it wins over the git-derived default.
function storeFromLocation(): string | null {
  if (typeof window === 'undefined') return null
  const store = new URLSearchParams(window.location.search).get('store')?.trim()
  return store ? store : null
}

let resolvedProject: string | null = null
async function resolveProject(): Promise<string> {
  const store = storeFromLocation()
  if (store) return store
  if (resolvedProject) return resolvedProject
  const gitName = await currentProjectName()
  try {
    const first = await get<{ total?: number }>(
      `/activity/stats?project=${encodeURIComponent(gitName)}`,
    )
    if ((first.total ?? 0) > 0) {
      resolvedProject = gitName
      return gitName
    }
  } catch {
    /* fall through */
  }
  try {
    const { projects } = await get<{ projects: string[] }>('/projects')
    for (const p of projects) {
      try {
        const r = await get<{ total?: number }>(`/activity/stats?project=${encodeURIComponent(p)}`)
        if ((r.total ?? 0) > 0) {
          resolvedProject = p
          return p
        }
      } catch {
        /* skip */
      }
    }
  } catch {
    /* fall through */
  }
  resolvedProject = gitName
  return gitName
}

function qs(project: string, params: Record<string, string | number> = {}): string {
  const q = new URLSearchParams({ project })
  for (const [k, v] of Object.entries(params)) q.set(k, String(v))
  return q.toString()
}

// ---------------------------------------------------------------------------
// types
// ---------------------------------------------------------------------------

export interface ActivityEvent {
  id: number
  ts: string
  type: string
  source: string
  actor?: string | null
  severity: string
  summary: string
  topicKey?: string | null
  sessionId: string
  relatedIds: number[]
}

export interface ActivityEventsResponse {
  project: string
  events: ActivityEvent[]
  limit: number
}

export interface ActivityStats {
  project: string
  total: number
  byType: Record<string, number>
  activeSessions: number
}

export type ObservationRow = {
  id: number
  type: string
  title: string
  created_at: string
  tokens?: number
  pinned?: boolean
}

export interface MnemonicSession {
  id: string
  title: string
  started_at: string
  ended_at?: string
  status: string
  memory_count: number
  observations?: number
  has_summary: boolean
  // Harness observability (Gryph-style). Older servers omit these.
  agent?: string
  files_read?: number
  files_written?: number
  commands_exec?: number
  errors?: number
  blocked_actions?: number
  policy_decisions?: number
  tool_calls?: number
  last_tool?: string
  last_active?: string
  input_tokens?: number
  output_tokens?: number
  cache_tokens?: number
  model?: string
  cost_usd?: number | null
}

// ToolEvent is one harness tool call (session_events row) — the same shape
// from GET /sessions/{id}/events, GET /events, and the SSE `tool` frame.
export interface ToolEvent {
  id: number
  ts: string
  sessionId: string
  sequence: number
  agent: string
  action: string
  tool: string
  path: string
  command: string
  result: string
  sensitive: boolean
  preview: string
  mcp: boolean
  newSession?: boolean
}

export interface ToolFilter {
  agent?: string
  action?: string
  tool?: string
  file?: string
  command?: string
  since?: string
  session?: string
}

export interface AgentStat {
  agent: string
  sessions: number
  events: number
  reads: number
  writes: number
  commands: number
  mcp: number
  errors: number
  blocked: number
  inputTokens: number
  outputTokens: number
  costUsd: number | null
}

export interface ModelStat {
  model: string
  sessions: number
  inputTokens: number
  outputTokens: number
  cacheTokens: number
  costUsd: number | null
}

export interface EventStats {
  project: string
  since: string
  total: number
  sessions: number
  mcp: number
  errors: number
  sensitive: number
  blocked: number
  warned?: number
  guided?: number
  byAction: Record<string, number>
  byAgent: AgentStat[]
  byModel?: ModelStat[]
  topFiles: { name: string; count: number }[]
  topCommands: { name: string; count: number }[]
  topTools: { name: string; count: number; mcp: boolean }[]
}

export interface SessionSummary {
  id: string
  summary: string
  status: string
  ended_at: string
}

export interface AuditEntry {
  seq: number
  observation_id: number
  revision: number
  created_at: string
  hash: string
  prev_hash: string
}

// ---------------------------------------------------------------------------
// activity feed (global) + session-scoped
// ---------------------------------------------------------------------------

export async function fetchActivityEvents(limit = 100): Promise<ActivityEventsResponse> {
  const project = await resolveProject()
  return get<ActivityEventsResponse>(`/activity/events?${qs(project, { limit })}`)
}

// Session-scoped feed: GET /sessions/{id}/activity (server-side filter added in
// sessions-activity-unification). Same shape as the global feed.
export async function fetchSessionActivity(id: string, limit = 200): Promise<ActivityEventsResponse> {
  const project = await resolveProject()
  return get<ActivityEventsResponse>(
    `/sessions/${encodeURIComponent(id)}/activity?${qs(project, { limit })}`,
  )
}

export async function fetchActivityStats(): Promise<ActivityStats> {
  const project = await resolveProject()
  return get<ActivityStats>(`/activity/stats?${qs(project)}`)
}

// ---------------------------------------------------------------------------
// sessions + summary + audit
// ---------------------------------------------------------------------------

export async function fetchSessions(): Promise<{ project: string; sessions: MnemonicSession[] }> {
  const project = await resolveProject()
  return get(`/mnemonic/sessions?${qs(project)}`)
}

function filterParams(f: ToolFilter): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(f)) {
    if (v && String(v).trim() !== '') out[k] = String(v).trim()
  }
  return out
}

// Session tool timeline: GET /sessions/{id}/events (newest first).
export async function fetchSessionEvents(
  id: string,
  filter: ToolFilter = {},
  limit = 300,
): Promise<{ events: ToolEvent[]; observations?: ObservationRow[] }> {
  const project = await resolveProject()
  const r = await get<{ events: ToolEvent[]; observations?: ObservationRow[] }>(
    `/sessions/${encodeURIComponent(id)}/events?${qs(project, { ...filterParams(filter), limit })}`,
  )
  return { ...r, observations: r.observations ?? [] }
}

// Project-wide tool query: GET /events.
export async function fetchEvents(filter: ToolFilter = {}, limit = 300): Promise<{ events: ToolEvent[] }> {
  const project = await resolveProject()
  return get(`/events?${qs(project, { ...filterParams(filter), limit })}`)
}

export async function fetchEventStats(filter: { since?: string; agent?: string } = {}): Promise<EventStats> {
  const project = await resolveProject()
  return get(`/events/stats?${qs(project, filterParams(filter))}`)
}

// ---------------------------------------------------------------------------
// pre-tool policy (ADR-0021) — read-only view of the merged rule set
// ---------------------------------------------------------------------------

export interface PolicyRule {
  name: string
  match: {
    action?: string[]
    path?: string[]
    command?: string[]
    tool?: string[]
    agent?: string[]
    project?: string[]
    counters?: Record<string, string>
  }
  effect: 'block' | 'warn' | 'guide' | 'allow'
  message?: string
  source: string
}

export interface PolicyView {
  project: string
  enabled: boolean
  rules: PolicyRule[]
  files: string[]
  repoFile?: string
  error?: string
}

export async function fetchPolicy(): Promise<PolicyView> {
  const project = await resolveProject()
  return get(`/policy?${qs(project)}`)
}

// globMatch mirrors the server's glob→LIKE mapping (* and ** any run, ? one
// char, case-sensitive) so live SSE frames honor the same filter as the query.
export function globMatch(glob: string, value: string): boolean {
  const re = glob
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    .replace(/\*+/g, '.*')
    .replace(/\?/g, '.')
  return new RegExp(`^${re}$`).test(value)
}

// matchesFilter applies a ToolFilter to one event client-side (live frames).
export function matchesFilter(e: ToolEvent, f: ToolFilter): boolean {
  if (f.session && e.sessionId !== f.session) return false
  if (f.agent && e.agent !== f.agent) return false
  if (f.action && e.action !== f.action) return false
  if (f.tool && e.tool.toLowerCase() !== f.tool.toLowerCase()) return false
  if (f.file && !globMatch(f.file, e.path)) return false
  if (f.command && !globMatch(f.command, e.command)) return false
  return true
}

export async function fetchSessionSummary(id: string): Promise<SessionSummary> {
  const project = await resolveProject()
  return get<SessionSummary>(
    `/sessions/${encodeURIComponent(id)}/summary?${qs(project)}`,
  )
}

export async function fetchAudit(limit = 200): Promise<{
  project: string
  entries: AuditEntry[]
  chain_valid: boolean
}> {
  const project = await resolveProject()
  return get(`/mnemonic/audit?${qs(project, { limit })}`)
}

// ---------------------------------------------------------------------------
// SSE — the single /activity/stream carries both `activity` and `snapshot`
// events. openActivityStream returns a cleanup fn; both callbacks are optional.
// ---------------------------------------------------------------------------

export interface StreamCallbacks {
  onActivity?: (e: ActivityEvent) => void
  onTool?: (e: ToolEvent) => void
  onSnapshot?: (s: {
    commit: string
    commitShort: string
    subject: string
    branch: string
    author: string
    committedAt: string
  }) => void
  onReady?: () => void
  onError?: (err: Error) => void
}

export function openActivityStream(cb: StreamCallbacks): () => void {
  let es: EventSource | null = null
  let stopped = false
  const start = async () => {
    const project = await resolveProject()
    if (stopped) return
    es = new EventSource(`/activity/stream?project=${encodeURIComponent(project)}`)
    es.addEventListener('ready', () => cb.onReady?.())
    es.addEventListener('activity', (ev) => {
      try {
        cb.onActivity?.(JSON.parse((ev as MessageEvent).data))
      } catch (e) {
        cb.onError?.(e as Error)
      }
    })
    es.addEventListener('tool', (ev) => {
      try {
        cb.onTool?.(JSON.parse((ev as MessageEvent).data))
      } catch (e) {
        cb.onError?.(e as Error)
      }
    })
    es.addEventListener('snapshot', (ev) => {
      try {
        cb.onSnapshot?.(JSON.parse((ev as MessageEvent).data))
      } catch (e) {
        cb.onError?.(e as Error)
      }
    })
    es.onerror = () => cb.onError?.(new Error('activity stream error'))
  }
  void start()
  return () => {
    stopped = true
    es?.close()
  }
}
