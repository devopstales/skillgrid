// Shared JSON fetch for dashboard panels. Injects `project` on every call.
import { currentProjectName } from './projects'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

let cachedProject: string | null = null

export async function resolveProject(): Promise<string> {
  if (cachedProject) return cachedProject
  cachedProject = await currentProjectName()
  return cachedProject
}

export function clearProjectCache() {
  cachedProject = null
}

export async function apiGet<T>(
  path: string,
  params: Record<string, string | number | boolean | undefined> = {},
  opts: { project?: boolean } = { project: true },
): Promise<T> {
  const q = new URLSearchParams()
  if (opts.project !== false) {
    q.set('project', await resolveProject())
  }
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined) continue
    q.set(k, String(v))
  }
  const qs = q.toString()
  const url = qs ? `${path}?${qs}` : path
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = (await res.json()) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      /* keep statusText */
    }
    throw new ApiError(msg, res.status)
  }
  return (await res.json()) as T
}

/** Absolute fetch without project injection (security/trivy, openapi, etc.). */
export async function apiFetch<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: 'application/json' } })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = (await res.json()) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      /* keep */
    }
    throw new ApiError(msg, res.status)
  }
  return (await res.json()) as T
}
