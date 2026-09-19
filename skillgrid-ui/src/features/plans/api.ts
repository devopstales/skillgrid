// Shared fetch + type contracts for the Phase 6 Plans view. Plans/specs are
// read from the .skillgrid/ tree (repo root), so these are plain relative GETs
// (no ?project= — the server resolves the repo from CWD / SKILLGIT_DOCS_CWD).

export class PlansError extends Error {
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
    throw new PlansError(msg, res.status)
  }
  return (await res.json()) as T
}

// ---------------------------------------------------------------------------
// types
// ---------------------------------------------------------------------------

export interface PlanSummary {
  name: string
  status: string
  progress: number
  tasksDone: number
  tasksTotal: number
  hasLedger: boolean
}

export interface PlanFile {
  name: string
  size: number
}

export interface PlanStep {
  raw: string
  status: string
}

export interface PlanDetail extends PlanSummary {
  files: PlanFile[]
  steps: PlanStep[]
  briefing: string
  tasks: string
}

export interface SpecFile {
  path: string
  content: string
}

// ---------------------------------------------------------------------------
// fetchers
// ---------------------------------------------------------------------------

export async function fetchPlans(): Promise<{ plans: PlanSummary[] }> {
  return get('/plans')
}

export async function fetchPlan(name: string): Promise<PlanDetail> {
  return get(`/plans/${encodeURIComponent(name)}`)
}

export async function fetchSpecFiles(): Promise<{ files: string[] }> {
  return get('/specs')
}

export async function fetchSpecContent(path: string): Promise<SpecFile> {
  return get(`/specs/${path.split('/').map(encodeURIComponent).join('/')}`)
}
