import { currentProjectName } from '../../lib/projects'

// Shared fetch + type contracts for the Handoff Hub (change 015-handoff-hub).
// Reuses the Activity view's project resolution (same git name → first
// project with data). The change log + hub status are read-only REST; the
// live snapshot stream reuses the /activity/stream SSE (event: snapshot).

export class HandoffError extends Error {
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
    throw new HandoffError(msg, res.status)
  }
  return (await res.json()) as T
}

let resolvedProject: string | null = null
async function resolveProject(): Promise<string> {
  if (resolvedProject) return resolvedProject
  const gitName = await currentProjectName()
  try {
    const first = await get<{ snapshots?: unknown[] }>(
      `/activity/snapshots?project=${encodeURIComponent(gitName)}&limit=1`,
    )
    if ((first.snapshots?.length ?? 0) > 0) {
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
        const r = await get<{ snapshots?: unknown[] }>(`/activity/snapshots?project=${encodeURIComponent(p)}&limit=1`)
        if ((r.snapshots?.length ?? 0) > 0) {
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
  handoffFile: string
  evidence: string
  status: 'open' | 'verified' | 'stale' | 'archived'
  createdAt: string
  verifiedAt: string
}

export interface HandoffRef {
  handoffId: string
  handoffType: 'session' | 'team' | 'checkpoint'
  fromCommit: string
  toCommit: string
  specDir: string
  teamId: string
  taskId: string
  createdAt: string
}

export interface SnapshotsResponse {
  project: string
  snapshots: ChangeSnapshot[]
  limit: number
}

export interface HubStatus {
  project: string
  latest_snapshot: ChangeSnapshot | null
  checkpoints: Checkpoint[]
  handoff_refs: HandoffRef[]
}

// ---------------------------------------------------------------------------
// REST
// ---------------------------------------------------------------------------

export async function fetchSnapshots(limit = 100): Promise<SnapshotsResponse> {
  const project = await resolveProject()
  return get<SnapshotsResponse>(`/activity/snapshots?${qs(project, { limit })}`)
}

export async function fetchHubStatus(): Promise<HubStatus> {
  const project = await resolveProject()
  return get<HubStatus>(`/handoff/status?${qs(project, {})}`)
}

// ---------------------------------------------------------------------------
// SSE — live change snapshots (event: snapshot on /activity/stream).
// ---------------------------------------------------------------------------

export interface SnapshotStreamCallbacks {
  onSnapshot?: (s: { commit: string; commitShort: string; subject: string; branch: string; author: string; committedAt: string }) => void
  onReady?: () => void
  onError?: (err: Error) => void
}

export function openSnapshotStream(cb: SnapshotStreamCallbacks): () => void {
  let es: EventSource | null = null
  let stopped = false
  const start = async () => {
    const project = await resolveProject()
    if (stopped) return
    es = new EventSource(`/activity/stream?project=${encodeURIComponent(project)}`)
    es.addEventListener('ready', () => cb.onReady?.())
    es.addEventListener('snapshot', (ev) => {
      try {
        cb.onSnapshot?.(JSON.parse((ev as MessageEvent).data))
      } catch (e) {
        cb.onError?.(e as Error)
      }
    })
    es.onerror = () => cb.onError?.(new Error('snapshot stream error'))
  }
  void start()
  return () => {
    stopped = true
    es?.close()
  }
}
