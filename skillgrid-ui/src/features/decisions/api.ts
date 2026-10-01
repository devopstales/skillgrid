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

// OpenQuestion is one item from the "Open Questions" section of
// .skillgrid/ASSUMPTIONS.md (numbered list: `1. **Title** — detail`, with an
// optional leading checkbox).
export interface OpenQuestion {
  text: string
  checked: boolean
}

const ASSUMPTIONS_PATH = '.skillgrid/ASSUMPTIONS.md'

// fetchOpenQuestions reads ASSUMPTIONS.md via the shared docs bridge and
// extracts its Open Questions section. A missing file/section yields [].
export async function fetchOpenQuestions(): Promise<OpenQuestion[]> {
  const res = await fetch(
    `/docs/content?path=${encodeURIComponent(ASSUMPTIONS_PATH)}`,
    { headers: { Accept: 'application/json' } },
  )
  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
  const j = (await res.json()) as { body?: string }
  return parseOpenQuestions(j.body ?? '')
}

// parseOpenQuestions extracts the Open Questions section (## or ### heading)
// and returns its list items (numbered or bulleted, with optional checkbox).
export function parseOpenQuestions(markdown: string): OpenQuestion[] {
  const lines = markdown.split('\n')
  const start = lines.findIndex((l) => /^#{2,3}\s+open questions\s*$/i.test(l))
  if (start === -1) return []

  const out: OpenQuestion[] = []
  for (let i = start + 1; i < lines.length; i++) {
    const line = lines[i]
    if (/^##\s+/.test(line)) break
    const item = /^\s*(?:[-*]|\d+[.)])\s+(.*)$/.exec(line)
    if (!item) continue
    let text = item[1].trim()
    let checked = false
    const cb = /^\[( |x|X)\]\s*(.*)$/.exec(text)
    if (cb) {
      checked = cb[1].toLowerCase() === 'x'
      text = cb[2].trim()
    }
    if (text) out.push({ text, checked })
  }
  return out
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
