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

// hasRenderableGraph fetches one node + a bounded edge sample for a project and
// reports whether it would render a real graph (at least one edge). A project
// with nodes but zero edges is a "stub" (degraded) — we want to prefer a
// project with edges over a stub even if the stub is the git-derived name.
async function hasRenderableGraph(project: string): Promise<boolean> {
  try {
    const d = await get<GraphResponse>(
      `/mnemonic/graph/data?project=${encodeURIComponent(project)}&limit=1`,
    )
    return d.nodes.length > 0 && d.edges.length > 0
  } catch {
    return false
  }
}

// nodeTotal returns the project's full symbol count (cheap: the /nodes endpoint
// reports total without paginating).
async function nodeTotal(project: string): Promise<number> {
  try {
    const r = await get<GraphNodesResponse>(
      `/mnemonic/graph/nodes?project=${encodeURIComponent(project)}&limit=1`,
    )
    return r.total
  } catch {
    return 0
  }
}

// resolveProject picks the project to render. Preference order:
//   1. the git-derived name, when it has a renderable graph (nodes AND edges);
//   2. the largest project (by node count) that has edges;
//   3. the largest project by node count, even if it's a 0-edge stub;
//   4. the git-derived name as a last resort.
// This keeps the graph view working regardless of how the server derived the
// project id, and avoids locking onto a git-named project that's an empty stub.
async function resolveProject(): Promise<string> {
  if (resolvedProject) return resolvedProject
  const gitName = await currentProjectName()
  if (await hasRenderableGraph(gitName)) {
    resolvedProject = gitName
    return gitName
  }
  let projects: string[] = []
  try {
    const j = await get<{ projects: string[] }>('/projects')
    projects = j.projects ?? []
  } catch {
    /* /projects unavailable — fall to the git name */
  }
  // score every project (nodes + whether it has edges)
  const scored = await Promise.all(
    [...new Set(projects)].map(async (p) => {
      const [total, hasEdges] = await Promise.all([nodeTotal(p), hasRenderableGraph(p)])
      return { p, total, hasEdges }
    }),
  )
  const withEdges = scored.filter((s) => s.hasEdges && s.total > 0)
  if (withEdges.length > 0) {
    withEdges.sort((a, b) => b.total - a.total)
    resolvedProject = withEdges[0].p
    return withEdges[0].p
  }
  // no project has edges — pick the largest by node count (may be a stub)
  const byTotal = scored
    .filter((s) => s.total > 0)
    .sort((a, b) => b.total - a.total)
  if (byTotal.length > 0) {
    resolvedProject = byTotal[0].p
    return byTotal[0].p
  }
  resolvedProject = gitName
  return gitName
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
