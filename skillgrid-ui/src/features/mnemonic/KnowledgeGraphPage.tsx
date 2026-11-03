import { useEffect, useMemo, useRef, useState } from 'react'
import * as d3 from 'd3'
import { fetchKnowledgeGraph } from './knowledge-graph/api'
import type {
  KnowledgeConceptEdge,
  KnowledgeConceptNode,
  KnowledgeGraphResponse,
} from './knowledge-graph/types'
import { EmptyState, ErrorState, LoadingState, SectionTitle } from '../../components/ui/Badges'

// Concept type colors: each knowledge type renders distinctly. Unknown types
// fall back to gray (the code-graph default).
const CONCEPT_TYPE_COLORS: Record<string, string> = {
  decision: '#60a5fa',
  invariant: '#f472b6',
  workflow: '#34d399',
  glossary: '#c084fc',
  reference: '#94a3b8',
}

type SimNode = KnowledgeConceptNode & d3.SimulationNodeDatum
type SimLink = {
  source: string | SimNode
  target: string | SimNode
}

export function KnowledgeGraphPage() {
  const [graph, setGraph] = useState<KnowledgeGraphResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<KnowledgeConceptNode | null>(null)
  const [zoom, setZoom] = useState(1)
  const graphRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchKnowledgeGraph()
      .then(setGraph)
      .catch((err: Error) => setError(err.message))
  }, [])

  // The selected concept's backlinks (who links to it) — straight from the
  // server-derived backlinks map.
  const backlinksFor = useMemo(() => {
    return (id: string): string[] => graph?.backlinks?.[id] ?? []
  }, [graph])

  useEffect(() => {
    if (!graph || !graphRef.current) return
    const container = graphRef.current
    const width = container.clientWidth || 800
    const height = container.clientHeight || 500
    const nodes: SimNode[] = graph.nodes.map((n) => ({ ...n }))
    const edges: SimLink[] = graph.edges.map((e: KnowledgeConceptEdge) => ({
      source: e.source,
      target: e.target,
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
          .distance(70),
      )
      .force('charge', d3.forceManyBody().strength(-30))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(12))

    const link = g
      .append('g')
      .selectAll('line')
      .data(edges)
      .enter()
      .append('line')
      .attr('stroke', '#2e2e3e')
      .attr('stroke-width', 1)

    const node = g
      .append('g')
      .selectAll('circle')
      .data(nodes)
      .enter()
      .append('circle')
      .attr('r', (d) => (d.deprecated ? 5 : 6))
      .attr('fill', (d) => CONCEPT_TYPE_COLORS[d.type] || '#64748b')
      // Orphans + deprecated render with distinct styling: orphans get a
      // dashed ring (no inbound links), deprecated get reduced opacity.
      .attr('stroke', (d) => (d.orphan ? '#f59e0b' : null))
      .attr('stroke-width', (d) => (d.orphan ? 2 : null))
      .attr('stroke-dasharray', (d) => (d.orphan ? '4 3' : null))
      .attr('opacity', (d) => (d.deprecated ? 0.45 : 1))
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

    node.append('title').text(
      (d) =>
        `${d.title} (${d.type})` +
        (d.deprecated ? ' [deprecated]' : '') +
        (d.orphan ? ' [orphan]' : '') +
        `\n${d.id}.md`,
    )

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
  if (graph.nodes.length === 0) {
    return (
      <div className="flex h-full flex-col">
        <div className="border-b border-edge p-4">
          <h2 className="text-sm font-semibold">Knowledge Graph</h2>
        </div>
        <div className="flex-1">
          <EmptyState message="No knowledge concepts in this bundle yet" />
        </div>
      </div>
    )
  }

  const typeCounts: Record<string, number> = {}
  for (const n of graph.nodes) typeCounts[n.type] = (typeCounts[n.type] || 0) + 1
  const orphanCount = graph.nodes.filter((n) => n.orphan).length
  const deprecatedCount = graph.nodes.filter((n) => n.deprecated).length
  const backlinks = selected ? backlinksFor(selected.id) : []

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-edge p-4">
        <div>
          <h2 className="text-sm font-semibold">Knowledge Graph</h2>
          <p className="text-xs text-ink-4">
            {graph.nodes.length} concepts / {graph.edges.length} links
            {orphanCount > 0 ? ` · ${orphanCount} orphan` : ''}
            {deprecatedCount > 0 ? ` · ${deprecatedCount} deprecated` : ''}
          </p>
        </div>
        <div className="flex gap-4 text-xs">
          {Object.entries(typeCounts)
            .sort((a, b) => b[1] - a[1])
            .map(([type, count]) => (
              <span key={type} className="flex items-center gap-1">
                <span
                  className="h-2 w-2 rounded-full"
                  style={{ background: CONCEPT_TYPE_COLORS[type] || '#64748b' }}
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
            <SectionTitle>Concept Inspector</SectionTitle>
            {selected ? (
              <div className="space-y-3">
                <div className="rounded bg-surface-800 p-2">
                  <div className="flex items-center gap-2">
                    <span
                      className="h-2.5 w-2.5 shrink-0 rounded-full"
                      style={{ background: CONCEPT_TYPE_COLORS[selected.type] || '#64748b' }}
                    />
                    <span className="break-all font-mono text-sm text-ink">{selected.title}</span>
                  </div>
                  <div className="mt-2 grid grid-cols-2 gap-y-1 text-[11px]">
                    <span className="text-ink-4">Type</span>
                    <span className="text-ink-3">{selected.type}</span>
                    <span className="text-ink-4">Status</span>
                    <span className={selected.deprecated ? 'text-warn' : 'text-ink-3'}>
                      {selected.status}
                    </span>
                    <span className="text-ink-4">Flags</span>
                    <span className="text-ink-3">
                      {[selected.deprecated && 'deprecated', selected.orphan && 'orphan']
                        .filter(Boolean)
                        .join(', ') || 'none'}
                    </span>
                    <span className="col-span-2 text-ink-4">Description</span>
                    <span className="col-span-2 text-ink-3">{selected.description || '—'}</span>
                    <span className="col-span-2 text-ink-4">Tags</span>
                    <span className="col-span-2 text-ink-3">
                      {/* A concept with no `tags:` frontmatter arrives as JSON
                          null (Go nil slice), so guard the length. */}
                      {selected.tags && selected.tags.length > 0
                        ? selected.tags.join(', ')
                        : '—'}
                    </span>
                    <span className="col-span-2 text-ink-4">File</span>
                    <span className="col-span-2 break-all font-mono text-[10px] text-accent">
                      {selected.id}.md
                    </span>
                  </div>
                </div>
                <div>
                  <div className="mb-1 text-[10px] uppercase tracking-wide text-ink-4">
                    Backlinks ({backlinks.length})
                  </div>
                  {backlinks.length > 0 ? (
                    <div className="max-h-48 space-y-0.5 overflow-y-auto">
                      {backlinks.map((id) => (
                        <div key={id} className="truncate font-mono text-[11px] text-ink-3">
                          ← {id}.md
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-xs text-ink-4">
                      {selected.orphan ? 'No inbound links (orphan).' : 'No backlinks.'}
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="text-xs text-ink-4">
                <p>Click a concept to inspect its frontmatter and backlinks.</p>
                <p className="mt-2 text-ink-5">
                  Dashed amber ring = orphan · dimmed = deprecated
                </p>
              </div>
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
