import type { GraphNodesResponse, GraphResponse } from './types'
import { currentProjectName } from '../../../lib/projects'

export class GraphError extends Error {
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
    throw new GraphError(msg, res.status)
  }
  return (await res.json()) as T
}

let resolvedProject: string | null = null

// resolveProject picks the project to render: the git-derived name when it has
// a graph, otherwise the largest indexed project (by node count). This makes
// the graph view work regardless of how the server's project id was derived
// (git remote name vs. the id the graph was indexed under).
async function resolveProject(): Promise<string> {
  if (resolvedProject) return resolvedProject
  const gitName = await currentProjectName()
  try {
    const first = await get<GraphResponse>(
      `/mnemonic/graph/data?project=${encodeURIComponent(gitName)}&limit=1`,
    )
    if (first.nodes.length > 0) {
      resolvedProject = gitName
      return gitName
    }
  } catch {
    /* git-named project has no graph — fall through */
  }
  // fall back to the largest indexed project
  try {
    const { projects } = await get<{ projects: string[] }>('/projects')
    let best = gitName
    let bestCount = 0
    for (const p of projects) {
      try {
        const r = await get<GraphNodesResponse>(
          `/mnemonic/graph/nodes?project=${encodeURIComponent(p)}&limit=1`,
        )
        if (r.total > bestCount) {
          bestCount = r.total
          best = p
        }
      } catch {
        /* skip */
      }
    }
    resolvedProject = best
    return best
  } catch {
    resolvedProject = gitName
    return gitName
  }
}

// fetchGraph loads the project's symbol graph. node_id + depth scope it to a
// depth-hop neighborhood; otherwise the full graph is returned (server-capped).
export async function fetchGraph(opts?: {
  nodeId?: number
  depth?: number
  limit?: number
}): Promise<GraphResponse> {
  const project = await resolveProject()
  const q = new URLSearchParams({ project })
  if (opts?.nodeId != null) q.set('node_id', String(opts.nodeId))
  if (opts?.depth != null) q.set('depth', String(opts.depth))
  if (opts?.limit != null) q.set('limit', String(opts.limit))
  return get<GraphResponse>(`/mnemonic/graph/data?${q.toString()}`)
}

export async function fetchGraphNodes(limit = 5000): Promise<GraphNodesResponse> {
  const project = await resolveProject()
  const q = new URLSearchParams({ project, limit: String(limit) })
  return get<GraphNodesResponse>(`/mnemonic/graph/nodes?${q.toString()}`)
}
