import { currentProjectName } from '../../lib/projects'

// Shared fetch helpers + type contracts for the Phase 5 mnemonic views
// (files / memories / sessions / audit / search). Project resolution mirrors
// the graph view: the git-derived name when it has data, otherwise the first
// project with memories.

export class MnemonicError extends Error {
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
    throw new MnemonicError(msg, res.status)
  }
  return (await res.json()) as T
}

async function write<T>(url: string, method: 'PUT' | 'POST', body: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = (await res.json()) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      /* keep statusText */
    }
    throw new MnemonicError(msg, res.status)
  }
  return (await res.json()) as T
}

let resolvedProject: string | null = null

// resolveProject picks the project to render: the git-derived name when it has
// memories, otherwise the first project with any observations. Falls back to
// the git name when nothing has memories yet.
async function resolveProject(): Promise<string> {
  if (resolvedProject) return resolvedProject
  const gitName = await currentProjectName()
  try {
    const first = await get<{ total?: number }>(
      `/mnemonic/memories?project=${encodeURIComponent(gitName)}&limit=1`,
    )
    if ((first.total ?? 0) > 0) {
      resolvedProject = gitName
      return gitName
    }
  } catch {
    /* git-named project has no memories — fall through */
  }
  try {
    const { projects } = await get<{ projects: string[] }>('/projects')
    for (const p of projects) {
      try {
        const r = await get<{ total?: number }>(
          `/mnemonic/memories?project=${encodeURIComponent(p)}&limit=1`,
        )
        if ((r.total ?? 0) > 0) {
          resolvedProject = p
          return p
        }
      } catch {
        /* skip */
      }
    }
    resolvedProject = gitName
    return gitName
  } catch {
    resolvedProject = gitName
    return gitName
  }
}

export async function mnemonicProject(): Promise<string> {
  return resolveProject()
}

function qs(project: string, params: Record<string, string | number>): string {
  const q = new URLSearchParams({ project })
  for (const [k, v] of Object.entries(params)) q.set(k, String(v))
  return q.toString()
}

// ---------------------------------------------------------------------------
// files
// ---------------------------------------------------------------------------

export interface FileTreeNode {
  name: string
  path: string
  leaf: boolean
  memory_count: number
  last_indexed?: string
  children?: FileTreeNode[]
}

export interface FileTreeResponse {
  project: string
  root: FileTreeNode
  nodes: number
}

export interface FileContentItem {
  id: number
  title: string
  type: string
  content: string
  created_at: string
}

export interface FileContentTier {
  label: 'abstract' | 'overview' | 'details'
  content: string
  count: number
  items?: FileContentItem[]
}

export interface FileContentResponse {
  uri: string
  project: string
  l0?: FileContentTier
  l1?: FileContentTier
  l2?: FileContentTier
}

export async function fetchFileTree(path?: string): Promise<FileTreeResponse> {
  const project = await resolveProject()
  const params: Record<string, string | number> = {}
  if (path) params.path = path
  return get<FileTreeResponse>(`/mnemonic/files/tree?${qs(project, params)}`)
}

export async function fetchFileContent(uri: string): Promise<FileContentResponse> {
  const project = await resolveProject()
  const q = new URLSearchParams({ project, uri })
  return get<FileContentResponse>(`/mnemonic/files/content?${q.toString()}`)
}

// ---------------------------------------------------------------------------
// memories
// ---------------------------------------------------------------------------

export interface MemorySummary {
  id: number
  title: string
  type: string
  scope: string
  topic_key?: string
  source?: string
  owner?: string
  visibility?: string
  status?: string
  pinned: boolean
  retrieval_usage: number
  revision_count: number
  created_at: string
  updated_at: string
}

export interface MemoryListResponse {
  project: string
  total: number
  limit: number
  offset: number
  memories: MemorySummary[]
}

export interface VersionItem {
  revision: number
  created_at: string
}

export interface GrantItem {
  grantee: string
  grant_type: string
}

export interface MemoryDetail {
  id: number
  title: string
  type: string
  content: string
  scope: string
  topic_key?: string
  source?: string
  created_at: string
  updated_at: string
  pinned: boolean
  owner?: string
  visibility?: string
  status?: string
  retrieval_usage: number
  revision_count: number
  versions?: VersionItem[]
  grants?: GrantItem[]
  governance: boolean
}

export async function fetchMemories(
  limit = 50,
  offset = 0,
): Promise<MemoryListResponse> {
  const project = await resolveProject()
  return get<MemoryListResponse>(
    `/mnemonic/memories?${qs(project, { limit, offset })}`,
  )
}

export async function fetchMemory(id: number): Promise<MemoryDetail> {
  const project = await resolveProject()
  return get<MemoryDetail>(
    `/mnemonic/memories/${id}?${qs(project, {})}`,
  )
}

export async function editMemory(
  id: number,
  body: { content?: string; title?: string },
): Promise<{ id: number; updated: boolean; revision: number }> {
  const project = await resolveProject()
  return write(`/mnemonic/memories/${id}?${qs(project, {})}`, 'PUT', body)
}

export async function shareMemory(
  id: number,
  body: { visibility: string; grants?: string[] },
): Promise<{ id: number; visibility: string }> {
  const project = await resolveProject()
  return write(
    `/mnemonic/memories/${id}/share?${qs(project, {})}`,
    'POST',
    body,
  )
}

export async function setStatus(
  id: number,
  status: string,
): Promise<{ id: number; status: string }> {
  const project = await resolveProject()
  return write(
    `/mnemonic/memories/${id}/status?${qs(project, {})}`,
    'POST',
    { status },
  )
}

// ---------------------------------------------------------------------------
// sessions + audit
// ---------------------------------------------------------------------------

export interface MnemonicSession {
  id: string
  title: string
  started_at: string
  ended_at?: string
  status: string
  memory_count: number
  has_summary: boolean
  agent?: string
}

export interface AuditEntry {
  seq: number
  observation_id: number
  revision: number
  created_at: string
  hash: string
  prev_hash: string
}

export async function fetchSessions(): Promise<{
  project: string
  sessions: MnemonicSession[]
}> {
  const project = await resolveProject()
  return get(`/mnemonic/sessions?${qs(project, {})}`)
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
// search
// ---------------------------------------------------------------------------

export interface SearchResult {
  id: number
  title: string
  type: string
  scope: string
  topic_key?: string
  source?: string
  preview: string
  created_at: string
  relevance: number
}

export async function searchMemories(
  q: string,
  mode: 'hybrid' | 'fts' = 'hybrid',
  limit = 25,
): Promise<{ project: string; results: SearchResult[]; total: number }> {
  const project = await resolveProject()
  return get(
    `/mnemonic/search?${qs(project, { q, mode, limit })}`,
  )
}
