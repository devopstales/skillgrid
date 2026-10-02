import { useEffect, useMemo, useRef, useState } from 'react'
import * as d3 from 'd3'
import { fetchGraph } from './graph/api'
import type { GraphEdge, GraphNode, GraphResponse } from './graph/types'
import { apiGet } from '../../lib/api'
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

export function GraphPage() {
  const [graph, setGraph] = useState<GraphResponse | null>(null)
  const [codeStatus, setCodeStatus] = useState<CodeStatus | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<SimNode | null>(null)
  const [zoom, setZoom] = useState(1)
  const graphRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchGraph({ limit: 500 })
      .then(setGraph)
      .catch((err: Error) => setError(err.message))
    apiGet<CodeStatus>('/code/status')
      .then(setCodeStatus)
      .catch((err: Error) => setStatusError(err.message))
  }, [])

  const adjacency = useMemo(() => {
    const map: Record<number, { incoming: number[]; outgoing: number[] }> = {}
    for (const e of graph?.edges ?? []) {
      if (!map[e.source]) map[e.source] = { incoming: [], outgoing: [] }
      if (!map[e.target]) map[e.target] = { incoming: [], outgoing: [] }
      map[e.target].incoming.push(e.source)
      map[e.source].outgoing.push(e.target)
    }
    return map
  }, [graph])

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
    const nodes: SimNode[] = graph.nodes.map((n) => ({ ...n }))
    const edges: SimLink[] = graph.edges.map((e: GraphEdge) => ({
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
      })

    node.append('title').text((d) => `${d.label} (${d.type})\n${d.path}`)

    const gNode = g.node()
    const clear = () => setSelected(null)
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
  }, [graph])

  useEffect(() => {
    const container = graphRef.current
    if (!container || !selected || !graph) return
    const svg = d3.select(container).select('svg')
    const node = svg.selectAll<SVGCircleElement, SimNode>('circle')
    const link = svg.selectAll<SVGLineElement, SimLink>('line')
    const selId = selected.id
    const neighborIds = new Set<number>([selId])
    for (const e of graph.edges) {
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
  }, [selected, graph])

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

  const selConn = selected ? adjacency[selected.id] || { incoming: [], outgoing: [] } : null

  const connRow = (dir: 'incoming' | 'outgoing', id: number) => {
    const other = nodeById[id]
    if (!other) return null
    const verb = dir === 'incoming' ? '← called by' : '→ calls'
    return (
      <div key={dir + id} className="flex items-start gap-1.5 py-0.5">
        <span className={`shrink-0 text-[10px] ${dir === 'incoming' ? 'text-warn' : 'text-accent'}`}>
          {verb}
        </span>
        <div className="min-w-0">
          <div className="truncate font-mono text-[11px] text-ink">{other.label}</div>
          <div className="truncate font-mono text-[10px] text-ink-5">{other.path}</div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-edge p-4">
        <div>
          <h2 className="text-sm font-semibold">Code Graph</h2>
          <p className="text-xs text-ink-4">
            {graph.nodes.length} nodes / {graph.edges.length} edges (limited to 500)
            {graph.truncated ? ' — truncated' : ''}
            {graph.degraded ? ' — degraded' : ''}
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
      <div className="flex flex-1">
        <div ref={graphRef} className="min-h-[400px] flex-1 bg-surface-900" />
        <div className="w-64 space-y-4 overflow-y-auto border-l border-edge p-4">
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
                    <span className="col-span-2 text-ink-4">File</span>
                    <span className="col-span-2 break-all font-mono text-[10px] text-accent">
                      {selected.path}
                    </span>
                  </div>
                </div>
                {selConn && (selConn.incoming.length > 0 || selConn.outgoing.length > 0) ? (
                  <div>
                    <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-4">
                      Connections ({selConn.incoming.length + selConn.outgoing.length})
                    </div>
                    <div className="max-h-64 space-y-0.5 overflow-y-auto">
                      {selConn.incoming.slice(0, 30).map((id) => connRow('incoming', id))}
                      {selConn.incoming.length > 30 && (
                        <div className="text-[10px] text-ink-4">
                          … {selConn.incoming.length - 30} more callers
                        </div>
                      )}
                      {selConn.outgoing.slice(0, 30).map((id) => connRow('outgoing', id))}
                      {selConn.outgoing.length > 30 && (
                        <div className="text-[10px] text-ink-4">
                          … {selConn.outgoing.length - 30} more callees
                        </div>
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="text-xs text-ink-4">
                    No edges in the loaded window (graph limited to 500 nodes).
                  </div>
                )}
              </div>
            ) : (
              <div className="text-xs text-ink-4">
                <p>Click a node to inspect symbol, type, file, and callers / callees.</p>
                <p className="mt-2 text-ink-5">
                  Edges are directed <code className="font-mono">calls</code> relations.
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
              <div>Drag nodes to reposition</div>
              <div>Scroll to zoom, drag to pan</div>
              <div>Click canvas to clear selection</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
