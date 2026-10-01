import { currentProjectName } from '../../lib/projects'
import type {
  ActivityEventsResponse,
  ActivityStats,
  CodeStatus,
  PipelineState,
} from './types'

// Shared fetch + project resolution for the Overview landing page. Reuses the
// same /activity/* + /code/status + /docs/content endpoints as the Sessions and
// Docs views. Project resolution mirrors the sessions view (git name → first
// project with data).

export class OverviewError extends Error {
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
    throw new OverviewError(msg, res.status)
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

export async function fetchActivityStats(): Promise<ActivityStats> {
  const project = await resolveProject()
  return get<ActivityStats>(`/activity/stats?${qs(project)}`)
}

export async function fetchActivityEvents(limit = 10): Promise<ActivityEventsResponse> {
  const project = await resolveProject()
  return get<ActivityEventsResponse>(`/activity/events?${qs(project, { limit })}`)
}

export async function fetchCodeStatus(): Promise<CodeStatus> {
  const project = await resolveProject()
  return get<CodeStatus>(`/code/status?${qs(project)}`)
}

// fetchPipelineState reads .skillgrid/state.yaml from /docs/content and parses
// the flat fields the Overview page needs. Returns null when the file is absent
// or the fields can't be read (the UI shows a fallback).
export async function fetchPipelineState(): Promise<PipelineState | null> {
  const res = await fetch(`/docs/content?path=${encodeURIComponent('.skillgrid/state.yaml')}`, {
    headers: { Accept: 'application/json' },
  })
  if (!res.ok) return null
  let body: { body?: string }
  try {
    body = (await res.json()) as { body?: string }
  } catch {
    return null
  }
  return parsePipelineState(body.body ?? '')
}

// Minimal line-based parse of the flat state.yaml (schema: skillgrid/state/v1).
// The relevant fields are simple scalars at 2-space indentation; no YAML
// dependency is available (and none may be added without an ADR).
export function parsePipelineState(yaml: string): PipelineState | null {
  const out: PipelineState = {
    current_phase: '',
    current_change: '',
    status: '',
    completed_changes: 0,
  }
  let sawPipeline = false
  let sawProgress = false
  for (const raw of yaml.split('\n')) {
    const line = raw.replace(/\r$/, '')
    if (/^pipeline:\s*$/.test(line)) {
      sawPipeline = true
      sawProgress = false
      continue
    }
    if (/^progress:\s*$/.test(line)) {
      sawProgress = true
      sawPipeline = false
      continue
    }
    if (sawPipeline) {
      const m = /^ {2}([a-z_]+):\s*(.*)$/.exec(line)
      if (!m) continue
      if (m[1] === 'current_phase') out.current_phase = unquote(m[2])
      else if (m[1] === 'current_change') out.current_change = unquote(m[2])
      else if (m[1] === 'status') out.status = unquote(m[2])
    } else if (sawProgress) {
      const m = /^ {2}([a-z_]+):\s*(.*)$/.exec(line)
      if (!m) continue
      if (m[1] === 'completed_changes') out.completed_changes = parseInt(m[2], 10) || 0
    }
  }
  return out
}

function unquote(s: string): string {
  const t = s.trim()
  if ((t.startsWith('"') && t.endsWith('"') && t.length >= 2) || (t.startsWith("'") && t.endsWith("'") && t.length >= 2)) {
    return t.slice(1, -1)
  }
  return t
}
