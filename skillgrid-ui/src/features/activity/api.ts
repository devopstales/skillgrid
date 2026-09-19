import { currentProjectName } from '../../lib/projects'

// Shared fetch + type contracts for the Phase 6 Activity view. The activity
// feed is backed by the observations table (no events table), so the project
// is resolved exactly like the mnemonic views (git name → first project with
// data). The SSE stream reuses the same project resolution.

export class ActivityError extends Error {
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
    throw new ActivityError(msg, res.status)
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

function qs(project: string, params: Record<string, string | number>): string {
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

// ---------------------------------------------------------------------------
// events + stats
// ---------------------------------------------------------------------------

export async function fetchActivityEvents(limit = 100): Promise<ActivityEventsResponse> {
  const project = await resolveProject()
  return get<ActivityEventsResponse>(`/activity/events?${qs(project, { limit })}`)
}

export async function fetchActivityStats(): Promise<ActivityStats> {
  const project = await resolveProject()
  return get<ActivityStats>(`/activity/stats?${qs(project, {})}`)
}

// ---------------------------------------------------------------------------
// SSE stream — live activity events. Returns a cleanup fn.
// ---------------------------------------------------------------------------

export interface StreamCallbacks {
  onActivity?: (e: ActivityEvent) => void
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
    es.onerror = () => cb.onError?.(new Error('activity stream error'))
  }
  void start()
  return () => {
    stopped = true
    es?.close()
  }
}
