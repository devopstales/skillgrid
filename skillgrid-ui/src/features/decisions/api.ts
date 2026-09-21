import { currentProjectName } from '../../lib/projects'
import type { Decision } from './DecisionCard'

// The decisions bridge read path is open; the answer (write) path is
// token-gated when the server sets SKILLGRID_HTTP_TOKEN. The token comes from
// the Settings view (localStorage) — the SPA cannot read the server's env, so
// the user supplies it once and it is attached to the mutating request, exactly
// like the tracker/kanban write routes.
const TOKEN_KEY = 'skillgrid.httpToken'

export class DecisionApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

function getToken(): string {
  try {
    return window.localStorage.getItem(TOKEN_KEY) ?? ''
  } catch {
    return ''
  }
}

function authHeaders(): Record<string, string> {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function parseError(res: Response): Promise<string> {
  try {
    const j = (await res.json()) as { error?: string }
    if (j.error) return j.error
  } catch {
    /* keep statusText */
  }
  return res.statusText
}

// DecisionRow mirrors the backend GET /mnemonic/decisions row (camelCase JSON).
export type DecisionRow = Decision

let resolvedProject: string | null = null

async function resolveProject(): Promise<string> {
  if (resolvedProject) return resolvedProject
  resolvedProject = await currentProjectName()
  return resolvedProject
}

export async function fetchDecisions(state: string): Promise<DecisionRow[]> {
  const project = await resolveProject()
  const q = new URLSearchParams({ project })
  if (state) q.set('state', state)
  const res = await fetch(`/mnemonic/decisions?${q.toString()}`, {
    headers: { Accept: 'application/json' },
  })
  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
  return (await res.json()) as DecisionRow[]
}

export async function answerDecision(
  id: number,
  answer: { optionId: string; note: string },
): Promise<void> {
  const project = await resolveProject()
  const res = await fetch(
    `/mnemonic/decisions/${id}/answer?project=${encodeURIComponent(project)}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json', ...authHeaders() },
      body: JSON.stringify(answer),
    },
  )
  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
}
