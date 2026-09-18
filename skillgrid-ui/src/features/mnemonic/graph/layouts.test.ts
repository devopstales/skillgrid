import { describe, it, expect } from 'vitest'
import Graph from 'graphology'
import { forceLayoutAnimated } from './layouts'
import type { GraphologyNode, GraphologyEdge } from './converters'

// makeGraph builds a directed graphology graph with seeded x/y + the node
// attributes the layouts expect (degree/community), so it's a valid input for
// forceLayoutAnimated without going through the HTTP converter.
function makeGraph(n: number, edges: [number, number][]): Graph<GraphologyNode, GraphologyEdge> {
  const g = new Graph<GraphologyNode, GraphologyEdge>({ type: 'directed', multi: true })
  const deg = new Array(n).fill(0)
  for (let i = 0; i < n; i++) {
    g.addNode(String(i), {
      size: 4,
      color: '#6366f1',
      label: `n${i}`,
      type: 'circle',
      nodeType: 'function',
      path: `f${i}.ts`,
      degree: 0,
      community: 0,
      uid: `u${i}`,
      x: Math.cos(i) * 10,
      y: Math.sin(i) * 10,
    })
  }
  for (const [s, t] of edges) {
    if (s === t || s >= n || t >= n) continue
    g.addEdge(String(s), String(t), {
      type: 'arrow',
      edgeType: 'calls',
      weight: 1,
      size: 1,
      color: '#cbd5e1',
    })
    deg[s]++
    deg[t]++
  }
  for (let i = 0; i < n; i++) {
    g.setNodeAttribute(String(i), 'degree', deg[i])
  }
  return g
}

function finitePositions(g: Graph<GraphologyNode, GraphologyEdge>): boolean {
  for (const n of g.nodes()) {
    const x = Number(g.getNodeAttribute(n, 'x'))
    const y = Number(g.getNodeAttribute(n, 'y'))
    if (!Number.isFinite(x) || !Number.isFinite(y)) return false
  }
  return true
}

describe('forceLayoutAnimated', () => {
  it('returns a done promise and stop() resolves it early with finite positions', async () => {
    // A ring of 30 nodes — a layout that takes several batches to settle.
    const g = makeGraph(30, Array.from({ length: 30 }, (_, i) => [i, (i + 1) % 30] as [number, number]))
    const handle = forceLayoutAnimated(g, { maxIterations: 10000, iterationsPerFrame: 2 })
    expect(handle.stop).toBeTypeOf('function')
    // Stop almost immediately — before it settles or exhausts iterations.
    handle.stop()
    await expect(handle.done).resolves.toBeUndefined()
    expect(finitePositions(g)).toBe(true)
  })

  it('positions change across frames while running (visible motion)', async () => {
    const g = makeGraph(20, Array.from({ length: 20 }, (_, i) => [i, (i + 1) % 20] as [number, number]))
    const handle = forceLayoutAnimated(g, { maxIterations: 10000, iterationsPerFrame: 1 })
    const snap = () => {
      let s = ''
      for (const n of g.nodes()) s += `${n}:${Number(g.getNodeAttribute(n, 'x')).toFixed(3)},${Number(g.getNodeAttribute(n, 'y')).toFixed(3)};`
      return s
    }
    const first = snap()
    // let a couple of frames run
    await new Promise((r) => setTimeout(r, 30))
    const mid = snap()
    handle.stop()
    await handle.done
    expect(mid).not.toBe(first) // nodes actually moved
    expect(finitePositions(g)).toBe(true)
  })

  it('settles on its own (done resolves without stop) for a small graph', async () => {
    // tiny graph settles quickly: low maxIterations keeps it bounded.
    const g = makeGraph(8, [[0, 1], [1, 2], [2, 3], [3, 4], [4, 5], [5, 6], [6, 7]])
    const handle = forceLayoutAnimated(g, { maxIterations: 60, iterationsPerFrame: 3, settleEpsilon: 0.01 })
    await expect(handle.done).resolves.toBeUndefined()
    expect(finitePositions(g)).toBe(true)
  })

  it('stop() is idempotent', async () => {
    const g = makeGraph(12, [[0, 1], [1, 2], [2, 3], [3, 4], [4, 5]])
    const handle = forceLayoutAnimated(g, { maxIterations: 10000, iterationsPerFrame: 1 })
    handle.stop()
    handle.stop()
    await expect(handle.done).resolves.toBeUndefined()
  })

  it('resolves immediately for an empty graph', async () => {
    const g = new Graph<GraphologyNode, GraphologyEdge>()
    const handle = forceLayoutAnimated(g)
    await expect(handle.done).resolves.toBeUndefined()
  })
})
