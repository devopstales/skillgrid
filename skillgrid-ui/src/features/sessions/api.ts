import { currentProjectName } from '../../lib/projects'

// Shared fetch + type contracts for the unified Sessions view
// (sessions-activity-unification). This merges the former features:
//   - Activity (event feed + stats + SSE)  → activity/events, /activity/stats, /activity/stream
//   - Mnemonic sessions + audit            → /mnemonic/sessions, /sessions/{id}/summary, /sessions/{id}/activity, /mnemonic/audit
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

let resolvedProject: string | null = null
async function resolveProject(): Promise<string> {
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

export interface SnapshotContext {
  task?: string
  decisions?: string
  remaining?: string
  tried?: string
}

export interface ChangeSnapshot {
  id: number
  branch: string
  commit: string
  commitShort: string
  subject: string
  author: string
  committedAt: string
  changedFiles: string
  context?: SnapshotContext
}

export interface Checkpoint {
  id: number
  name: string
  branch: string
  commit: string
  dirty: boolean
  prdPath: string
  specDir: string
  evidence: string
  status: 'open' | 'verified' | 'stale' | 'archived'
  createdAt: string
  verifiedAt: string
}

export interface SnapshotsResponse {
  project: string
  snapshots: ChangeSnapshot[]
  limit: number
}

export interface MnemonicSession {
  id: string
  title: string
  started_at: string
  ended_at?: string
  status: string
  memory_count: number
  has_summary: boolean
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
