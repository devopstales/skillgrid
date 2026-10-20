import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import * as d3 from 'd3'
import { fetchGraph } from './graph/api'
import type { GraphEdge, GraphNode, GraphResponse } from './graph/types'
import {
  apiGet,
} from '../../lib/api'
import {
  EmptyState,
  ErrorState,
  LoadingState,
  SectionTitle,
  StatusBadge,
} from '../../components/ui/Badges'

const GRAPH_TYPE_COLORS: Record<string, string> = {
  function: '#60a5fa',
  method: '#34d399',
  type: '#fbbf24',
  class: '#f472b6',
  route: '#c084fc',
  reference: '#94a3b8',
}

type SimNode = GraphNode & d3.SimulationNodeDatum
type SimLink = {
  source: number | SimNode
  target: number | SimNode
  type: string
}

interface CodeStatus {
  stale?: boolean
  file_count?: number
  chunk_count?: number
  last_indexed?: string
}

// ConceptNode is the shape of /knowledge/graph nodes (the concept overlay).
interface ConceptNode {
  id: string
  title: string
  type: string
  description: string
  tags: string[]
  status: string
  orphan: boolean
  deprecated: boolean
}
interface ConceptGraphResponse {
  nodes: ConceptNode[]
  edges: { source: string; target: string }[]
}

// ---------------------------------------------------------------------------
// Local (non-component) helpers — pure, testable in isolation.
// ---------------------------------------------------------------------------

// graphLines reads the additive `lines` field (TICKET-12 enrichment) from a
// node, tolerating its absence (old payloads / unlocated symbols).
function graphLines(n: GraphNode): { start: number; end: number } | null {
  const l = (n as unknown as { lines?: { start: number; end: number } }).lines
  if (!l || typeof l.start !== 'number' || typeof l.end !== 'number') return null
  return l
}

// fileTree groups nodes by their file path (the tree panel source).
function fileTree(nodes: GraphNode[]): Map<string, GraphNode[]> {
  const m = new Map<string, GraphNode[]>()
  for (const n of nodes) {
    const arr = m.get(n.path) ?? []
    arr.push(n)
    m.set(n.path, arr)
  }
  for (const arr of m.values()) arr.sort((a, b) => a.label.localeCompare(b.label))
  return m
}

// applyFilters keeps the nodes matching the kind/community/layer filters and
// the newcomer cap (uncategorized nodes, community -1, are capped). Returns
// the visible node set and the number hidden.
function applyFilters(
  nodes: GraphNode[],
  kind: string,
  community: string,
  layer: string,
  newcomerCap: number,
): { visible: Set<number>; hidden: number } {
  const visible = new Set<number>()
  let uncategorized = 0
  for (const n of nodes) {
    if (kind !== 'all' && n.type !== kind) continue
    if (community !== 'all' && String(n.community) !== community) continue
    if (layer !== 'all' && !n.path.startsWith(layer)) continue
    if (n.community === -1) {
      // newcomer cap: limit uncategorized nodes to the first `cap`
      if (uncategorized >= newcomerCap) continue
      uncategorized++
    }
    visible.add(n.id)
  }
  return { visible, hidden: nodes.length - visible.size }
}

// TOUR_STOPS is the client-side teaching tour: workflow → decisions →
// top-centrality (at least 3 stops, no endpoint involved).
interface TourStop {
  title: string
  body: string
  focus: (g: GraphResponse) => GraphNode | null
}
const TOUR_STOPS: TourStop[] = [
  {
    title: 'Workflow',
    body: 'Start at the entry point: main orchestrates the call graph. Its outgoing calls are the workflow edges.',
    focus: (g) => {
      const main = g.nodes.find((n) => n.label === 'main' || n.label === 'Main' || n.type === 'route')
      return main ?? g.nodes[0] ?? null
    },
  },
  {
    title: 'Decisions',
    body: 'Community boundaries mark the architectural decisions: tightly-coupled clusters stay together. The concept overlay anchors ADRs onto these clusters.',
    focus: (g) => {
      // the most connected community is where the decisions live
      const byCommunity = new Map<number, number>()
      for (const n of g.nodes) {
        if (n.community >= 0) byCommunity.set(n.community, (byCommunity.get(n.community) ?? 0) + n.degree)
      }
      let best: number | null = null
      let bestScore = -1
      for (const [c, s] of byCommunity) {
        if (s > bestScore) {
          bestScore = s
          best = c
        }
      }
      return g.nodes.find((n) => n.community === best) ?? g.nodes[0] ?? null
    },
  },
  {
    title: 'Top centrality',
    body: 'This symbol has the highest degree in the graph: it is the hub. Change it first, and expect the widest blast radius.',
    focus: (g) =>
      [...g.nodes].sort((a, b) => b.degree - a.degree)[0] ?? null,
  },
]

export function GraphPage() {
  const [graph, setGraph] = useState<GraphResponse | null>(null)
  const [codeStatus, setCodeStatus] = useState<CodeStatus | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<SimNode | null>(null)
  const [zoom, setZoom] = useState(1)
  const [focusLabel, setFocusLabel] = useState<string | null>(null)

  // teaching-page state (TICKET-12)
  const [treeExpanded, setTreeExpanded] = useState<Set<string>>(new Set())
  const [search, setSearch] = useState('')
  const [kindFilter, setKindFilter] = useState('all')
  const [communityFilter, setCommunityFilter] = useState('all')
  const [layerFilter, setLayerFilter] = useState('all')
  const [newcomerCap, setNewcomerCap] = useState(50)
  const [conceptsOn, setConceptsOn] = useState(false)
  const [concepts, setConcepts] = useState<ConceptGraphResponse | null>(null)
  const [tourStep, setTourStep] = useState<number | null>(null)
  const graphRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchGraph({ limit: 500 })
      .then(setGraph)
      .catch((err: Error) => setError(err.message))
    apiGet<CodeStatus>('/code/status')
      .then(setCodeStatus)
      .catch((err: Error) => setStatusError(err.message))
    apiGet<ConceptGraphResponse>('/knowledge/graph')
      .then(setConcepts)
      .catch(() => setConcepts(null)) // degraded: no overlay, never an error
  }, [])

  const tree = useMemo(() => fileTree(graph?.nodes ?? []), [graph])
  const pathList = useMemo(() => [...tree.keys()].sort(), [tree])

  const kinds = useMemo(
    () => [...new Set((graph?.nodes ?? []).map((n) => n.type))].sort(),
    [graph],
  )
  const communities = useMemo(
    () =>
      [...new Set((graph?.nodes ?? []).map((n) => n.community))]
        .filter((c) => c >= 0)
        .sort((a, b) => a - b),
    [graph],
  )
  const layers = useMemo(() => {
    const ls = new Set<string>()
    for (const p of graph?.nodes ?? []) {
      const dir = p.path.includes('/') ? p.path.slice(0, p.path.lastIndexOf('/')) : ''
      if (dir) ls.add(dir)
    }
    return [...ls].sort()
  }, [graph])

  const { visible, hidden } = useMemo(
    () =>
      applyFilters(graph?.nodes ?? [], kindFilter, communityFilter, layerFilter, newcomerCap),
    [graph, kindFilter, communityFilter, layerFilter, newcomerCap],
  )

  const filteredNodes = useMemo(
    () => (graph?.nodes ?? []).filter((n) => visible.has(n.id)),
    [graph, visible],
  )
  const filteredEdges = useMemo(
    () =>
      (graph?.edges ?? []).filter((e) => visible.has(e.source) && visible.has(e.target)),
    [graph, visible],
  )

  // both-ways sync: selecting a node in the graph reveals its file in the tree.
  useEffect(() => {
    if (!selected) return
    setTreeExpanded((prev) => new Set(prev).add(selected.path))
  }, [selected])

  const selectNode = useCallback((id: number) => {
    const n = graph?.nodes.find((x) => x.id === id)
    if (!n) return
    setSelected(n)
    setFocusLabel(n.label)
  }, [graph])

  // search-to-focus: type a symbol → the graph centers on it.
  useEffect(() => {
    if (!search) {
      setFocusLabel(null)
      return
    }
    const q = search.toLowerCase()
    const match = (graph?.nodes ?? []).find(
      (n) => n.label.toLowerCase() === q || n.label.toLowerCase().includes(q),
    )
    if (match) {
      setSelected(match)
      setFocusLabel(match.label)
      setTreeExpanded((prev) => new Set(prev).add(match.path))
    } else {
      setFocusLabel(null)
    }
  }, [search, graph])

  // focus the graph on a file's nodes (tree → graph direction).
  const focusFile = useCallback(
    (path: string) => {
      const nodes = tree.get(path) ?? []
      if (nodes.length === 0) {
        setFocusLabel(path)
        return
      }
      const hub = [...nodes].sort((a, b) => b.degree - a.degree)[0]
      setSelected(hub)
      setFocusLabel(path)
    },
    [tree],
  )

  const startTour = useCallback(() => {
    setTourStep(0)
  }, [])
  const nextTour = useCallback(() => {
    setTourStep((s) => (s == null ? null : s + 1 >= TOUR_STOPS.length ? null : s + 1))
  }, [])

  const tourStop = tourStep != null && tourStep < TOUR_STOPS.length ? TOUR_STOPS[tourStep] : null

  // tour: center the graph on the stop's focus symbol.
  useEffect(() => {
    if (!tourStop || !graph) return
    const n = tourStop.focus(graph)
    if (n) setSelected(n)
  }, [tourStop, graph])

  const nodeById = useMemo(() => {
    const m: Record<number, GraphNode> = {}
    for (const n of graph?.nodes ?? []) m[n.id] = n
    return m
  }, [graph])

  useEffect(() => {
    if (!graph || !graphRef.current) return
    const container = graphRef.current
    const width = container.clientWidth || 800
    const height = container.clientHeight || 500
    const nodes: SimNode[] = filteredNodes.map((n) => ({ ...n }))
    const edges: SimLink[] = filteredEdges.map((e: GraphEdge) => ({
      source: e.source,
      target: e.target,
      type: e.type,
    }))

    const svg = d3.select(container).append('svg').attr('width', width).attr('height', height)
    const g = svg.append('g')
    svg.call(
      d3
        .zoom<SVGSVGElement, unknown>()
        .scaleExtent([0.1, 5])
        .on('zoom', (event) => {
          g.attr('transform', event.transform.toString())
          setZoom(event.transform.k)
        }),
    )

    const simulation = d3
      .forceSimulation(nodes)
      .force(
        'link',
        d3
          .forceLink<SimNode, SimLink>(edges)
          .id((d) => d.id)
          .distance(60),
      )
      .force('charge', d3.forceManyBody().strength(-2))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(4))

    const link = g
      .append('g')
      .selectAll('line')
      .data(edges)
      .enter()
      .append('line')
      .attr('stroke', '#2e2e3e')
      .attr('stroke-width', 0.5)

    const node = g
      .append('g')
      .selectAll('circle')
      .data(nodes)
      .enter()
      .append('circle')
      .attr('r', 3)
      .attr('fill', (d) => GRAPH_TYPE_COLORS[d.type] || '#64748b')
      .style('cursor', 'pointer')
      .call(
        d3
          .drag<SVGCircleElement, SimNode>()
          .on('start', (event, d) => {
            if (!event.active) simulation.alphaTarget(0.3).restart()
            d.fx = d.x
            d.fy = d.y
          })
          .on('drag', (event, d) => {
            d.fx = event.x
            d.fy = event.y
          })
          .on('end', (event, d) => {
            if (!event.active) simulation.alphaTarget(0)
            d.fx = null
            d.fy = null
          }),
      )
      .on('click', (event, d) => {
        event.stopPropagation()
        setSelected(d)
        setFocusLabel(d.label)
      })

    node.append('title').text((d) => `${d.label} (${d.type})\n${d.path}`)

    const gNode = g.node()
    const clear = () => {
      setSelected(null)
      setFocusLabel(null)
    }
    gNode?.addEventListener('click', clear)

    simulation.on('tick', () => {
      link
        .attr('x1', (d) => (d.source as SimNode).x ?? 0)
        .attr('y1', (d) => (d.source as SimNode).y ?? 0)
        .attr('x2', (d) => (d.target as SimNode).x ?? 0)
        .attr('y2', (d) => (d.target as SimNode).y ?? 0)
      node.attr('cx', (d) => d.x ?? 0).attr('cy', (d) => d.y ?? 0)
    })

    return () => {
      simulation.stop()
      gNode?.removeEventListener('click', clear)
      svg.remove()
    }
  }, [filteredNodes, filteredEdges])

  // selection highlight (neighbor emphasis).
  useEffect(() => {
    const container = graphRef.current
    if (!container || !selected || !graph) return
    const svg = d3.select(container).select('svg')
    const node = svg.selectAll<SVGCircleElement, SimNode>('circle')
    const link = svg.selectAll<SVGLineElement, SimLink>('line')
    const selId = selected.id
    const neighborIds = new Set<number>([selId])
    for (const e of filteredEdges) {
      if (e.source === selId || e.target === selId) {
        neighborIds.add(e.source)
        neighborIds.add(e.target)
      }
    }
    node
      .attr('stroke', (d) => (d.id === selId ? '#fff' : null))
      .attr('stroke-width', (d) => (d.id === selId ? 2 : null))
      .attr('r', (d) => (d.id === selId ? 5 : neighborIds.has(d.id) ? 4 : 3))
      .attr('opacity', (d) => (neighborIds.has(d.id) ? 1 : 0.15))
    link
      .attr('stroke', (e) => {
        const s = typeof e.source === 'object' ? e.source.id : e.source
        const t = typeof e.target === 'object' ? e.target.id : e.target
        return s === selId || t === selId ? '#60a5fa' : '#2e2e3e'
      })
      .attr('stroke-width', (e) => {
        const s = typeof e.source === 'object' ? e.source.id : e.source
        const t = typeof e.target === 'object' ? e.target.id : e.target
        return s === selId || t === selId ? 1.5 : 0.5
      })
      .attr('opacity', (e) => {
        const s = typeof e.source === 'object' ? e.source.id : e.source
        const t = typeof e.target === 'object' ? e.target.id : e.target
        return s === selId || t === selId ? 1 : 0.1
      })
  }, [selected, graph, filteredEdges])

  if (error) {
    return (
      <div className="p-6">
        <ErrorState error={error} />
      </div>
    )
  }
  if (!graph) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  const typeCounts: Record<string, number> = {}
  for (const n of graph.nodes) typeCounts[n.type] = (typeCounts[n.type] || 0) + 1

  // typed connection groups for the inspector (per-edge source lines).
  const selConn = selected
    ? filteredEdges
        .filter((e) => e.source === selected.id || e.target === selected.id)
        .map((e) => {
          const dir: 'incoming' | 'outgoing' = e.source === selected.id ? 'outgoing' : 'incoming'
          const otherId = dir === 'outgoing' ? e.target : e.source
          return { dir, otherId, type: e.type, line: e.line }
        })
        .reduce<Record<string, { dir: 'incoming' | 'outgoing'; otherId: number; type: string; line?: number }[]>>((acc, c) => {
          (acc[c.type] ??= []).push(c)
          return acc
        }, {})
    : null

  const degraded = graph.degraded && filteredEdges.length === 0
  const emptyFilters = filteredNodes.length === 0

  return (
    <div className="flex h-full flex-col">
      {/* toolbar: search, filters, newcomer cap, concept toggle, tour */}
      <div className="flex flex-wrap items-center gap-2 border-b border-edge p-3 text-xs">
        <input
          aria-label="Search symbols"
          placeholder="Search symbols…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-40 rounded border border-edge bg-surface-800 px-2 py-1 text-xs text-ink placeholder:text-ink-5"
        />
        <select
          aria-label="Kind"
          value={kindFilter}
          onChange={(e) => setKindFilter(e.target.value)}
          className="rounded border border-edge bg-surface-800 px-2 py-1 text-xs text-ink"
        >
          <option value="all">kind: all</option>
          {kinds.map((k) => (
            <option key={k} value={k}>
              kind: {k}
            </option>
          ))}
        </select>
        <select
          aria-label="Community"
          value={communityFilter}
          onChange={(e) => setCommunityFilter(e.target.value)}
          className="rounded border border-edge bg-surface-800 px-2 py-1 text-xs text-ink"
        >
          <option value="all">community: all</option>
          {communities.map((c) => (
            <option key={c} value={String(c)}>
              community {c}
            </option>
          ))}
        </select>
        <select
          aria-label="Layer"
          value={layerFilter}
          onChange={(e) => setLayerFilter(e.target.value)}
          className="rounded border border-edge bg-surface-800 px-2 py-1 text-xs text-ink"
        >
          <option value="all">layer: all</option>
          {layers.map((l) => (
            <option key={l} value={l}>
              {l}
            </option>
          ))}
        </select>
        <label className="flex items-center gap-1 text-ink-4">
          newcomer cap
          <input
            aria-label="Newcomer cap"
            type="number"
            min={0}
            value={newcomerCap}
            onChange={(e) => setNewcomerCap(Number(e.target.value) || 0)}
            className="w-14 rounded border border-edge bg-surface-800 px-1 py-1 text-xs text-ink"
          />
        </label>
        <button
          role="switch"
          aria-checked={conceptsOn}
          aria-label="Concepts overlay"
          onClick={() => setConceptsOn((v) => !v)}
          className={`rounded px-2 py-1 ${
            conceptsOn ? 'bg-accent/20 text-accent' : 'bg-surface-800 text-ink-4'
          }`}
        >
          Concepts {conceptsOn ? 'on' : 'off'}
        </button>
        <button
          onClick={startTour}
          disabled={tourStep != null}
          className="rounded bg-accent/20 px-2 py-1 text-accent disabled:opacity-40"
        >
          Start tour
        </button>
        {focusLabel && <span className="text-ink-4">focused on {focusLabel}</span>}
        {hidden > 0 && <span className="text-ink-5">{hidden} hidden</span>}
      </div>

      {/* header: stats + legend */}
      <div className="flex items-center justify-between border-b border-edge px-4 py-2">
        <div>
          <h2 className="text-sm font-semibold">Code Graph</h2>
          <p className="text-xs text-ink-4">
            {filteredNodes.length}/{graph.nodes.length} nodes / {filteredEdges.length} edges
            (limited to 500)
            {graph.truncated ? ' — truncated' : ''}
            {graph.degraded ? ' — degraded: no edges' : ''}
          </p>
        </div>
        <div className="flex gap-4 text-xs">
          {Object.entries(typeCounts)
            .sort((a, b) => b[1] - a[1])
            .map(([type, count]) => (
              <span key={type} className="flex items-center gap-1">
                <span
                  className="h-2 w-2 rounded-full"
                  style={{ background: GRAPH_TYPE_COLORS[type] || '#64748b' }}
                />
                {type} ({count})
              </span>
            ))}
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* tree panel (two-way synced) */}
        <div className="w-56 shrink-0 overflow-y-auto border-r border-edge p-3">
          <SectionTitle>Files</SectionTitle>
          {pathList.length === 0 ? (
            <div className="text-xs text-ink-4">No files in the loaded window.</div>
          ) : (
            <div className="space-y-1">
              {pathList.map((path) => {
                const expanded = treeExpanded.has(path)
                const focused = selected?.path === path
                return (
                  <div key={path}>
                    <div
                      className={`cursor-pointer truncate rounded px-1 py-0.5 font-mono text-[11px] ${
                        focused ? 'bg-accent/20 text-accent' : 'text-ink-3 hover:bg-surface-800'
                      }`}
                      onClick={() => focusFile(path)}
                      title={path}
                    >
                      {expanded ? '▾' : '▸'} {path}
                    </div>
                    {expanded && (
                      <div className="ml-3 space-y-0.5">
                        {(tree.get(path) ?? []).map((n) => (
                          <div
                            key={n.id}
                            className={`cursor-pointer truncate rounded px-1 py-0.5 font-mono text-[11px] ${
                              selected?.id === n.id
                                ? 'bg-accent/20 text-accent'
                                : 'text-ink-4 hover:bg-surface-800'
                            }`}
                            onClick={() => selectNode(n.id)}
                            title={n.label}
                          >
                            {n.label}
                            {graphLines(n) ? ` (${graphLines(n)!.start})` : ''}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* graph canvas + degraded states */}
        <div className="relative min-h-[400px] flex-1 bg-surface-900">
          <div ref={graphRef} className="h-full w-full" />
          {degraded && (
            <div className="absolute left-3 top-3 max-w-xs rounded border border-edge bg-surface-800/90 p-3">
              <div className="text-xs font-semibold text-warn">degraded: no edges</div>
              <div className="mt-1 text-xs text-ink-4">
                Node-only view. Served file list:
              </div>
              <div className="mt-1 max-h-32 overflow-y-auto font-mono text-[10px] text-ink-3">
                {(graph.files ?? pathList).map((p) => (
                  <div key={p}>{p}</div>
                ))}
              </div>
            </div>
          )}
          {emptyFilters && !degraded && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="rounded border border-edge bg-surface-800/90 p-4 text-center">
                <div className="text-xs text-ink-3">no symbols match the filters</div>
                <div className="mt-1 text-[11px] text-ink-5">
                  Reset kind / community / layer to see the graph.
                </div>
              </div>
            </div>
          )}
          {tourStop && (
            <div className="absolute bottom-3 left-3 right-3 mx-auto max-w-md rounded border border-accent/40 bg-surface-800/95 p-4">
              <div className="text-xs font-semibold text-accent">
                {tourStop.title} — stop {(tourStep ?? 0) + 1} of {TOUR_STOPS.length}
              </div>
              <p className="mt-2 text-xs text-ink-3">{tourStop.body}</p>
              <div className="mt-3 flex justify-end gap-2">
                <button onClick={() => setTourStep(null)} className="rounded px-2 py-1 text-xs text-ink-4">
                  End tour
                </button>
                <button
                  onClick={nextTour}
                  className="rounded bg-accent/20 px-2 py-1 text-xs text-accent"
                >
                  Next stop
                </button>
              </div>
            </div>
          )}
          {conceptsOn && concepts && concepts.nodes.length > 0 && (
            <div className="absolute bottom-3 right-3 max-h-64 w-52 overflow-y-auto rounded border border-edge bg-surface-800/95 p-3">
              <div className="text-xs font-semibold text-ink-3">Concepts ({concepts.nodes.length})</div>
              <div className="mt-1 space-y-1">
                {concepts.nodes.map((c) => (
                  <div key={c.id} className="text-[11px]">
                    <span className="font-mono text-ink-3">{c.title}</span>
                    <span className="text-ink-5"> · {c.type}</span>
                    {c.orphan && <span className="text-warn"> · orphan</span>}
                    {c.deprecated && <span className="text-warn"> · deprecated</span>}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* inspector */}
        <div className="w-72 shrink-0 space-y-4 overflow-y-auto border-l border-edge p-4">
          <div>
            <SectionTitle>Node Inspector</SectionTitle>
            {selected ? (
              <div className="space-y-3">
                <div className="rounded bg-surface-800 p-2">
                  <div className="flex items-center gap-2">
                    <span
                      className="h-2.5 w-2.5 shrink-0 rounded-full"
                      style={{ background: GRAPH_TYPE_COLORS[selected.type] || '#64748b' }}
                    />
                    <span className="break-all font-mono text-sm text-ink">{selected.label}</span>
                  </div>
                  <div className="mt-2 grid grid-cols-2 gap-y-1 text-[11px]">
                    <span className="text-ink-4">Type</span>
                    <span className="text-ink-3">{selected.type}</span>
                    <span className="text-ink-4">Language</span>
                    <span className="text-ink-3">{selected.language}</span>
                    <span className="text-ink-4">Degree</span>
                    <span className="text-ink-3">{selected.degree} connections</span>
                    <span className="text-ink-4">Community</span>
                    <span className="text-ink-3">{selected.community}</span>
                    {graphLines(selected) && (
                      <>
                        <span className="text-ink-4">Lines</span>
                        <span className="text-ink-3">
                          lines {graphLines(selected)!.start}-{graphLines(selected)!.end}
                        </span>
                      </>
                    )}
                    {(selected as unknown as { signature?: string }).signature && (
                      <>
                        <span className="text-ink-4">Signature</span>
                        <span className="break-all font-mono text-[10px] text-ink-3">
                          {(selected as unknown as { signature?: string }).signature}
                        </span>
                      </>
                    )}
                    <span className="col-span-2 text-ink-4">File</span>
                    <span className="col-span-2 break-all font-mono text-[10px] text-accent">
                      {selected.path}
                    </span>
                  </div>
                </div>

                {/* typed connection groups with per-edge source lines */}
                {selConn && Object.keys(selConn).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(selConn).map(([kind, conns]) => (
                      <div key={kind}>
                        <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-4">
                          {kind} ({conns.length})
                        </div>
                        <div className="max-h-40 space-y-0.5 overflow-y-auto">
                          {conns.slice(0, 20).map((c) => {
                            const other = nodeById[c.otherId]
                            if (!other) return null
                            const verb = c.dir === 'incoming' ? '← from' : '→ to'
                            return (
                              <div key={c.dir + c.otherId + c.type} className="flex items-start gap-1.5 py-0.5">
                                <span
                                  className={`shrink-0 text-[10px] ${
                                    c.dir === 'incoming' ? 'text-warn' : 'text-accent'
                                  }`}
                                >
                                  {verb}
                                </span>
                                <div className="min-w-0">
                                  <div className="truncate font-mono text-[11px] text-ink">
                                    {other.label}
                                    {c.line != null && (
                                      <span className="text-ink-5"> L{c.line}</span>
                                    )}
                                  </div>
                                  <div className="truncate font-mono text-[10px] text-ink-5">
                                    {other.path}
                                  </div>
                                </div>
                              </div>
                            )
                          })}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-ink-4">
                    No edges in the loaded window (graph limited to 500 nodes).
                  </div>
                )}

                {/* ephemeral locals (on-demand, read-only) */}
                <div>
                  <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-4">Locals</div>
                  {graphLines(selected) &&
                  (selected as unknown as { signature?: string }).signature ? (
                    <div className="rounded border border-edge bg-surface-800 p-2 font-mono text-[10px] text-ink-3">
                      {(selected as unknown as { signature?: string }).signature}
                    </div>
                  ) : (
                    <div className="text-xs text-ink-5">
                      locals unavailable for this symbol (degraded: no source range)
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="text-xs text-ink-4">
                <p>Click a node to inspect symbol, lines, typed connections, and locals.</p>
                <p className="mt-2 text-ink-5">
                  Edges are directed <code className="font-mono">calls</code> /{' '}
                  <code className="font-mono">imports</code> relations with source lines.
                </p>
              </div>
            )}
          </div>

          <div>
            <SectionTitle>Index Status</SectionTitle>
            {statusError ? (
              <ErrorState error={statusError} />
            ) : codeStatus ? (
              <div className="space-y-1 text-xs">
                <div className="flex justify-between">
                  <span className="text-ink-4">Status</span>
                  <StatusBadge status={codeStatus.stale ? 'stale' : 'fresh'} />
                </div>
                <div className="flex justify-between">
                  <span className="text-ink-4">Files</span>
                  <span className="font-mono">{codeStatus.file_count}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-ink-4">Chunks</span>
                  <span className="font-mono">{codeStatus.chunk_count?.toLocaleString()}</span>
                </div>
              </div>
            ) : (
              <EmptyState message="No index status" />
            )}
          </div>

          <div>
            <SectionTitle>Controls</SectionTitle>
            <div className="space-y-1 text-xs text-ink-4">
              <div>Zoom: {zoom.toFixed(1)}x</div>
              <div>Click node → inspect + highlight</div>
              <div>Tree ↔ graph sync both ways</div>
              <div>Search → centers the graph</div>
              <div>Drag nodes / scroll to zoom</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
