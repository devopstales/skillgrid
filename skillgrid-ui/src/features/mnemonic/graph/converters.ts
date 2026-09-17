import Graph from 'graphology'
import type { Attributes } from 'graphology-types'
import type { GraphEdge, GraphNode } from './types'
import { communityColor } from './communities'

// graphNodeAttributes is the node attribute shape Sigma expects: x/y (position,
// added by a layout later), size (by degree), color (by community), label, plus
// our metadata (nodeType/path/degree/community) for tooltips + the legend.
// NOTE: `type` is Sigma's render-program selector. In Sigma v3 the only program
// registered by default is "circle" (resolveSettings merges {circle: CircleProgram});
// "default" does NOT exist, so nodes must use type: 'circle'. The symbol kind is
// stored in `nodeType` to avoid colliding with the program selector.
export interface GraphologyNode extends Attributes {
  size: number
  color: string
  label: string
  type: 'circle'
  nodeType: string
  path: string
  degree: number
  community: number
  uid: string
  // x/y are assigned by the layout (seeded to 0 at build); declared here so
  // setNodeAttribute(n, 'x', ...) type-checks.
  x: number
  y: number
}

export interface GraphologyEdge extends Attributes {
  // `type` is Sigma's edge-program selector — v3 only registers "arrow"/"line"
  // by default, so we use "arrow". The actual edge kind (calls/imports/...) is
  // stored in `edgeType` for tooltips/legend.
  type: 'arrow' | 'line'
  edgeType: string
  weight: number
  size: number
  color: string
  confidence?: string
}

// sizeByDegree maps a node's degree to a Sigma size (px-ish). Bounded so a
// few hub nodes don't dominate the view.
export function sizeByDegree(degree: number): number {
  return 4 + Math.min(24, Math.sqrt(Math.max(0, degree)) * 4)
}

// mnemonicGraphToGraphology converts the flat {nodes, edges} payload to a
// graphology Graph. Edges whose endpoints are absent from the node set are
// dropped (the spec's pruning rule). Directed edges are added with the
// `directed` graph flag; multi (duplicate source/target) edges are allowed.
export function mnemonicGraphToGraphology(
  nodes: GraphNode[],
  edges: GraphEdge[],
): Graph<GraphologyNode, GraphologyEdge> {
  const g = new Graph<GraphologyNode, GraphologyEdge>({ type: 'directed', multi: true })

  const idToKey = new Map<number, string>()
  for (const n of nodes) {
    const key = String(n.id)
    idToKey.set(n.id, key)
    g.addNode(key, {
      label: n.label,
      size: sizeByDegree(n.degree),
      color: communityColor(n.community),
      type: 'circle', // Sigma v3's only default node program; symbol kind is in nodeType
      nodeType: n.type,
      path: n.path,
      degree: n.degree,
      community: n.community,
      uid: n.uid,
      // x/y are assigned by the layout; seed to 0 so Sigma has a valid start.
      x: 0,
      y: 0,
    })
  }

  const seen = new Set<string>()
  for (const e of edges) {
    const s = idToKey.get(e.source)
    const t = idToKey.get(e.target)
    if (!s || !t || s === t) continue // drop dangling / self edges
    // multi is on, but guard against a duplicate identical source->target pair
    // (the backend can emit two edges for the same pair across kinds); keep the
    // first so a double-load under StrictMode can't trip the edge guard.
    const pair = s + '\u0000' + t
    if (seen.has(pair)) continue
    seen.add(pair)
    g.addEdge(s, t, {
      type: 'arrow', // Sigma v3 edge-program selector; the kind is in edgeType
      edgeType: e.type,
      weight: e.weight,
      size: 0.5 + Math.min(2.5, e.weight),
      color: '#cbd5e1',
      confidence: e.confidence,
    })
  }

  return g
}
