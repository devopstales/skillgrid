import type {
  ProvidersInfo,
  TaskListResponse,
  TrackerDeps,
  UnifiedTask,
} from './types'

// The HTTP write path is token-gated. The token comes from the Settings view
// (stored in localStorage) — the SPA cannot read the server's env, so the user
// supplies it once and it's attached to every mutating request.
const TOKEN_KEY = 'skillgrid.httpToken'

export function getToken(): string {
  try {
    return window.localStorage.getItem(TOKEN_KEY) ?? ''
  } catch {
    return ''
  }
}

export function setToken(token: string): void {
  try {
    if (token) window.localStorage.setItem(TOKEN_KEY, token)
    else window.localStorage.removeItem(TOKEN_KEY)
  } catch {
    // localStorage unavailable; writes will 401 until the user retries
  }
}

function authHeaders(): Record<string, string> {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export class TrackerError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function parse<T>(res: Response): Promise<T> {
  const text = await res.text()
  let body: any
  try {
    body = text ? JSON.parse(text) : {}
  } catch {
    body = { error: text }
  }
  if (!res.ok) {
    throw new TrackerError(body?.error ?? `HTTP ${res.status}`, res.status)
  }
  return body as T
}

export function fetchProviders(provider?: string): Promise<ProvidersInfo> {
  const qs = provider ? `?provider=${encodeURIComponent(provider)}` : ''
  return fetch(`/tracker/providers${qs}`).then((r) => parse<ProvidersInfo>(r))
}

export function fetchTasks(provider?: string): Promise<TaskListResponse> {
  const qs = provider ? `?provider=${encodeURIComponent(provider)}` : ''
  return fetch(`/tracker/tasks${qs}`).then((r) => parse<TaskListResponse>(r))
}

export function fetchTask(id: string, provider?: string): Promise<UnifiedTask> {
  const qs = provider ? `?provider=${encodeURIComponent(provider)}` : ''
  return fetch(`/tracker/tasks/${encodeURIComponent(id)}${qs}`).then((r) =>
    parse<UnifiedTask>(r),
  )
}

export function fetchDeps(
  id: string,
  provider?: string,
): Promise<TrackerDeps> {
  const qs = provider ? `?provider=${encodeURIComponent(provider)}` : ''
  return fetch(`/tracker/tasks/${encodeURIComponent(id)}/deps${qs}`).then((r) =>
    parse<TrackerDeps>(r),
  )
}

export function updateTaskStatus(
  id: string,
  status: string,
  provider?: string,
): Promise<UnifiedTask> {
  const qs = provider ? `?provider=${encodeURIComponent(provider)}` : ''
  return fetch(`/tracker/tasks/${encodeURIComponent(id)}${qs}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ status }),
  }).then((r) => parse<UnifiedTask>(r))
}
