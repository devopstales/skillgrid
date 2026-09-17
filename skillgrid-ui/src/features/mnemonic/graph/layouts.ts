import type Graph from 'graphology'
import forceAtlas2 from 'graphology-layout-forceatlas2'
import type { GraphologyNode, GraphologyEdge } from './converters'

export type LayoutKind = 'force' | 'tree' | 'circles'

// forceLayout runs ForceAtlas2 for a bounded number of iterations. It stops
// early when the layout "settles" (force below the threshold) so large graphs
// don't spin forever.
export function forceLayout(
  g: Graph<GraphologyNode, GraphologyEdge>,
  opts?: { iterations?: number },
): void {
  const iterations = opts?.iterations ?? 120
  // .assign applies the computed positions back onto the graph (the plain call
  // only returns a mapping). Settings live under params.settings.
  forceAtlas2.assign(g, {
    iterations,
    settings: {
      scalingRatio: 20,
      gravity: 0.05,
      strongGravityMode: true,
      outboundAttractionDistribution: false,
      adjustSizes: true,
      barnesHutTheta: 0.5,
      linLogMode: false,
    },
  })
}

// circlesLayout assigns nodes to concentric circles by degree rank: the most
// connected nodes sit on the inner circle, the rest fan out. Deterministic and
// cheap — a good "organized" default.
export function circlesLayout(g: Graph<GraphologyNode, GraphologyEdge>): void {
  const nodes = g.order()
  if (nodes.length === 0) return
  const byDegree = [...nodes].sort(
    (a, b) => (g.getNodeAttribute(b, 'degree') ?? 0) - (g.getNodeAttribute(a, 'degree') ?? 0),
  )
  // assign a ring to each node based on its rank percentile
  const maxRank = Math.max(1, byDegree.length - 1)
  byDegree.forEach((node, rank) => {
    const t = rank / maxRank // 0 = center, 1 = outer
    const radius = 20 + t * 400
    const angle = (rank * 2.399963) // golden angle for even spread
    g.setNodeAttribute(node, 'x', radius * Math.cos(angle))
    g.setNodeAttribute(node, 'y', radius * Math.sin(angle))
  })
}

// treeLayout assigns positions via a BFS tree from the highest-degree node,
// spreading children by depth + sibling index. Fallback to circles when the
// graph is disconnected (BFS won't reach every node).
export function treeLayout(g: Graph<GraphologyNode, GraphologyEdge>): void {
  if (g.order() === 0) return
  // root = highest-degree node
  let root = g.order()[0]
  let best = -1
  for (const n of g.order()) {
    const d = g.getNodeAttribute(n, 'degree') ?? 0
    if (d > best) {
      best = d
      root = n
    }
  }
  const parent = new Map<string, string>()
  const depth = new Map<string, number>()
  const siblingIndex = new Map<string, number>()
  const queue: string[] = [root]
  depth.set(root, 0)
  siblingIndex.set(root, 0)
  let placed = 0
  while (queue.length > 0) {
    const cur = queue.shift()!
    const d = depth.get(cur)!
    let sib = 0
    for (const nb of g.neighbors(cur)) {
      if (!depth.has(nb)) {
        parent.set(nb, cur)
        depth.set(nb, d + 1)
        siblingIndex.set(nb, sib++)
        queue.push(nb)
        placed++
      }
    }
  }
  // place nodes: x by sibling index within depth, y by depth
  const byDepth = new Map<number, string[]>()
  for (const n of g.order()) {
    const d = depth.get(n)
    if (d == null) continue // unreachable from root
    const arr = byDepth.get(d) ?? []
    arr.push(n)
    byDepth.set(d, arr)
  }
  for (const [d, ns] of byDepth) {
    ns.sort(
      (a, b) =>
        (siblingIndex.get(a) ?? 0) - (siblingIndex.get(b) ?? 0),
    )
    ns.forEach((n, i) => {
      g.setNodeAttribute(n, 'x', (i - ns.length / 2) * 60)
      g.setNodeAttribute(n, 'y', d * 80)
    })
  }
  // any unreachable nodes fall back to a circles position
  if (placed < g.order()) {
    for (const n of g.order()) {
      if (!depth.has(n)) {
        const angle = g.hashCode(n) * 0.7
        g.setNodeAttribute(n, 'x', 100 * Math.cos(angle))
        g.setNodeAttribute(n, 'y', 100 * Math.sin(angle))
      }
    }
  }
}

// applyLayout dispatches by kind.
export function applyLayout(
  g: Graph<GraphologyNode, GraphologyEdge>,
  kind: LayoutKind,
  opts?: { iterations?: number },
): void {
  switch (kind) {
    case 'force':
      forceLayout(g, opts)
      break
    case 'tree':
      treeLayout(g)
      break
    case 'circles':
      circlesLayout(g)
      break
  }
}
