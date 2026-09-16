export const PROJECT_STORAGE_KEY = 'skillgrid.project'

export interface ProjectsState {
  projects: string[]
  status: 'loading' | 'ready' | 'unavailable'
}

export function readStoredProject(): string | null {
  try {
    return window.localStorage.getItem(PROJECT_STORAGE_KEY)
  } catch {
    return null
  }
}

export function storeProject(project: string): void {
  try {
    window.localStorage.setItem(PROJECT_STORAGE_KEY, project)
  } catch {
    // localStorage may be unavailable (private mode); selection still works in memory
  }
}

export async function fetchProjects(): Promise<string[]> {
  const res = await fetch('/projects', {
    headers: { Accept: 'application/json' },
  })
  if (!res.ok) {
    throw new Error(`GET /projects returned ${res.status}`)
  }
  const data = (await res.json()) as { projects?: unknown }
  const list = Array.isArray(data.projects) ? data.projects : []
  return list.filter((p): p is string => typeof p === 'string')
}
