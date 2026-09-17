import type Graph from 'graphology'
import type { GraphologyNode, GraphologyEdge } from './converters'

// nodeKeys returns all node keys. Uses g.nodes() (not g.order(), which is a
// *method* on Sigma's internal graph object but a *property* on a raw graphology
// instance) so it works with either shape.
function nodeKeys(g: Graph<GraphologyNode, GraphologyEdge>): string[] {
  return g.nodes()
}

// filterGraphByDepth returns the set of node keys within `depth` hops (undirected
// BFS) of `center`. Used by the DepthSlider + semantic search highlight: nodes
// outside the set are hidden (Sigma's nodeReducer reads this set).
export function neighborhood(g: Graph<GraphologyNode, GraphologyEdge>, center: string, depth: number): Set<string> {
  if (depth <= 0) return new Set([center])
  const dist = new Map<string, number>([[center, 0]])
  const queue: string[] = [center]
  while (queue.length > 0) {
    const cur = queue.shift()!
    if (dist.get(cur)! >= depth) continue
    for (const nb of g.neighbors(cur)) {
      if (!dist.has(nb)) {
        dist.set(nb, dist.get(cur)! + 1)
        queue.push(nb)
      }
    }
  }
  return new Set(dist.keys())
}

// filterGraphByDepth wraps neighborhood with a null-safe guard: a null center
// or a center not present in the graph returns the full node set (show all).
export function filterGraphByDepth(
  g: Graph<GraphologyNode, GraphologyEdge>,
  center: string | null,
  depth: number,
): Set<string> {
  if (center == null || !g.hasNode(center)) return new Set(nodeKeys(g))
  return neighborhood(g, center, depth)
}

// visibleLabels returns the labels of nodes in a visible set — the data source
// for the search panel's autocomplete.
export function visibleLabels(
  g: Graph<GraphologyNode, GraphologyEdge>,
  visible: Set<string>,
): string[] {
  const out: string[] = []
  for (const n of visible) {
    const label = g.getNodeAttribute(n, 'label')
    if (typeof label === 'string') out.push(label)
  }
  return out
}

// searchNode finds a node key whose label/path/type matches the query
// (case-insensitive substring). Returns the first match, or null.
export function searchNode(
  g: Graph<GraphologyNode, GraphologyEdge>,
  query: string,
): string | null {
  const q = query.trim().toLowerCase()
  if (!q) return null
  for (const n of nodeKeys(g)) {
    const label = String(g.getNodeAttribute(n, 'label') ?? '').toLowerCase()
    const path = String(g.getNodeAttribute(n, 'path') ?? '').toLowerCase()
    const type = String(g.getNodeAttribute(n, 'nodeType') ?? '').toLowerCase()
    if (label.includes(q) || path.includes(q) || type.includes(q)) return n
  }
  return null
}
