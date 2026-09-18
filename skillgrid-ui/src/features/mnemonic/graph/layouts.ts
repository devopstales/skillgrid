import type Graph from 'graphology'
import forceAtlas2 from 'graphology-layout-forceatlas2'
import type { GraphologyNode, GraphologyEdge } from './converters'

export type LayoutKind = 'force' | 'tree' | 'circles'

// hashKey maps a node key to a stable pseudo-random-ish number in [0, 2π) for
// spreading disconnected nodes on the fallback circle. Deterministic (string
// hash) so repeated renders place the same node in the same spot.
function hashKey(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return (h >>> 0) % (Math.PI * 2)
}

// forceLayout runs ForceAtlas2 for a bounded number of iterations. It stops
// early when the layout "settles" (force below the threshold) so large graphs
// don't spin forever.
export function forceLayout(
  g: Graph<GraphologyNode, GraphologyEdge>,
  opts?: { iterations?: number },
): void {
  const iterations = opts?.iterations ?? 300
  // .assign applies the computed positions back onto the graph (the plain call
  // only returns a mapping). Shared settings (see FORCE_SETTINGS).
  forceAtlas2.assign(g, {
    iterations,
    settings: FORCE_SETTINGS,
  })
}

// ForceAtlas2 settings shared by the sync and animated force layouts. Tuned so
// nodes don't collapse into one blob: high scalingRatio + low gravity pushes
// hubs apart, outboundAttraction keeps chains straight, adjustSizes forces a
// per-node repulsion floor, and a low barnesHutTheta gives crisper repulsion
// for the ~5k-node graphs this renders.
const FORCE_SETTINGS = {
  scalingRatio: 60,
  gravity: 0.1,
  strongGravityMode: false,
  outboundAttractionDistribution: true,
  adjustSizes: true,
  barnesHutTheta: 0.4,
  linLogMode: false,
} as const

export interface AnimatedLayoutHandle {
  // Resolve when the layout settles, is stopped, or exhausts maxIterations.
  done: Promise<void>
  // Interrupt mid-flight. Idempotent; positions remain whatever they are at
  // the moment of stop (always a valid, finite layout).
  stop: () => void
}

// forceLayoutAnimated runs the ForceAtlas2 simulation incrementally across
// animation frames so the graph visibly settles instead of the tab freezing.
// The library only exposes a synchronous assign(graph, iterations), so we drive
// it in small batches: each frame runs `iterationsPerFrame` iterations (assign
// reads the graph's current x/y as its starting state, so batching is safe and
// the motion is continuous), then yields to rAF. It stops early when the max
// per-frame displacement drops below `settleEpsilon` for a few frames, when
// `maxIterations` is reached, or when stop() is called.
//
// In a non-DOM environment (tests) requestAnimationFrame falls back to
// setTimeout(0) via the `raf`/`cancel` injectables so it's fully testable.
export function forceLayoutAnimated(
  g: Graph<GraphologyNode, GraphologyEdge>,
  opts?: {
    maxIterations?: number
    iterationsPerFrame?: number
    settleEpsilon?: number
    raf?: (cb: () => void) => number
    cancel?: (id: number) => void
  },
): AnimatedLayoutHandle {
  const maxIterations = opts?.maxIterations ?? 300
  const perFrame = opts?.iterationsPerFrame ?? 4
  const settleEpsilon = opts?.settleEpsilon ?? 0.05
  // Prefer requestAnimationFrame in the browser; fall back to a 16 ms timer in
  // non-DOM environments (tests) so the simulation is fully testable.
  const raf =
    opts?.raf ??
    (typeof requestAnimationFrame === 'function'
      ? (cb: () => void) => requestAnimationFrame(cb)
      : (cb: () => void) => setTimeout(cb, 16) as unknown as number)
  const cancel =
    opts?.cancel ??
    (typeof cancelAnimationFrame === 'function'
      ? (id: number) => cancelAnimationFrame(id)
      : (id: number) => clearTimeout(id as unknown as ReturnType<typeof setTimeout>))

  const nodes = g.nodes()
  if (nodes.length === 0) {
    return { done: Promise.resolve(), stop: () => {} }
  }

  let rafId = 0
  let frame = 0
  let stableFrames = 0
  let done = false
  let resolveDone: () => void = () => {}
  const donePromise = new Promise<void>((r) => {
    resolveDone = r
  })

  function positions(): Map<string, { x: number; y: number }> {
    const m = new Map<string, { x: number; y: number }>()
    for (const n of nodes) {
      m.set(n, {
        x: Number(g.getNodeAttribute(n, 'x') ?? 0),
        y: Number(g.getNodeAttribute(n, 'y') ?? 0),
      })
    }
    return m
  }

  function step(): void {
    if (done) return
    const before = positions()
    forceAtlas2.assign(g, { iterations: perFrame, settings: FORCE_SETTINGS })
    frame++
    // max per-frame displacement → settle heuristic.
    let maxMove = 0
    for (const n of nodes) {
      const b = before.get(n)!
      const dx = Number(g.getNodeAttribute(n, 'x') ?? 0) - b.x
      const dy = Number(g.getNodeAttribute(n, 'y') ?? 0) - b.y
      const d = Math.hypot(dx, dy)
      if (d > maxMove) maxMove = d
    }
    if (maxMove < settleEpsilon) stableFrames++
    else stableFrames = 0
    const settled = stableFrames >= 3
    if (settled || frame * perFrame >= maxIterations) {
      done = true
      resolveDone()
      return
    }
    rafId = raf(step)
  }

  rafId = raf(step)

  function stop(): void {
    if (done) return
    done = true
    cancel(rafId)
    resolveDone()
  }

  return { done: donePromise, stop }
}

// circlesLayout assigns nodes to concentric circles by degree rank: the most
// connected nodes sit on the inner circle, the rest fan out. Deterministic and
// cheap — a good "organized" default.
export function circlesLayout(g: Graph<GraphologyNode, GraphologyEdge>): void {
  const nodes = g.nodes()
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
  if (g.order === 0) return
  // root = highest-degree node
  let root = g.nodes()[0]
  let best = -1
  for (const n of g.nodes()) {
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
  for (const n of g.nodes()) {
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
  if (placed < g.order) {
    for (const n of g.nodes()) {
      if (!depth.has(n)) {
        const angle = hashKey(n) * 0.7
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
