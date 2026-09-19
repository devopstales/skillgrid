import { useEffect, useMemo, useState } from 'react'
import {
  SigmaContainer,
  useLoadGraph,
  useRegisterEvents,
  useSigma,
  useSetSettings,
} from '@react-sigma/core'
import '@react-sigma/core/lib/style.css'
import type Graph from 'graphology'
import { useMnemonicGraph } from './hooks/useMnemonicGraph'
import { filterGraphByDepth, searchNode, visibleLabels } from './filters'
import { LayoutSwitcher } from './controls/LayoutSwitcher'
import { SearchPanel } from './controls/SearchPanel'
import { Legend } from './controls/Legend'
import { DepthSlider } from './controls/DepthSlider'
import { StopLayoutButton } from './controls/StopLayoutButton'
import type { GraphologyNode, GraphologyEdge } from './converters'

// Loader must render inside SigmaContainer and load the graphology instance.
function Loader({ g }: { g: Graph<GraphologyNode, GraphologyEdge> | null }) {
  const loadGraph = useLoadGraph()
  useEffect(() => {
    if (g) loadGraph(g)
  }, [g, loadGraph])
  return null
}

// HoverEffects highlights the hovered node + its neighbors.
function HoverEffects() {
  const sigma = useSigma()
  const setSettings = useSetSettings()
  const registerEvents = useRegisterEvents()
  const [hovered, setHovered] = useState<string | null>(null)

  useEffect(() => {
    registerEvents({
      enterNode: (e) => setHovered(e.node),
      leaveNode: () => setHovered(null),
    })
  }, [registerEvents])

  useEffect(() => {
    setSettings({
      nodeReducer: (node, data) => {
        if (!hovered) return data
        const g = sigma.getGraph()
        if (node === hovered || g.neighbors(hovered).includes(node)) {
          return { ...data, highlighted: true }
        }
        return { ...data, color: '#475569' }
      },
    })
  }, [hovered, setSettings, sigma])
  return null
}

// SearchHighlight dims everything outside the searched node + its neighborhood.
function SearchHighlight({ center }: { center: string | null }) {
  const sigma = useSigma()
  const setSettings = useSetSettings()
  useEffect(() => {
    setSettings({
      nodeReducer: (node, data) => {
        if (!center) return data
        const g = sigma.getGraph()
        const near = node === center || g.neighbors(center).includes(node)
        return near ? { ...data, highlighted: true } : { ...data, color: '#334155', label: '' }
      },
      edgeReducer: (edge, data) => {
        if (!center) return data
        const [s, t] = sigma.getGraph().extremities(edge)
        return s === center || t === center ? data : { ...data, color: '#1e293b' }
      },
    })
  }, [center, setSettings, sigma])
  return null
}

// HoverTooltip shows node type / path / degree / community on hover.
function HoverTooltip() {
  const sigma = useSigma()
  const registerEvents = useRegisterEvents()
  const [tip, setTip] = useState<{ x: number; y: number; text: string } | null>(null)

  useEffect(() => {
    registerEvents({
      enterNode: (e) => {
        const g = sigma.getGraph()
        const n = g.getNodeAttributes(e.node) as unknown as Record<string, unknown>
        const offsetX = e.event.original instanceof MouseEvent ? e.event.original.offsetX : 0
        setTip({
          x: offsetX,
          y: 0,
          text: `${n.label} · ${n.nodeType} · degree ${n.degree} · ${
            (n.community as number) < 0 ? 'no community' : `community ${n.community}`
          }`,
        })
      },
      mousemove: (e) => {
        const offsetX = e.original instanceof MouseEvent ? e.original.offsetX : 0
        setTip((prev) => (prev ? { ...prev, x: offsetX, y: 0 } : prev))
      },
      leaveNode: () => setTip(null),
    })
  }, [registerEvents, sigma])

  if (!tip) return null
  return (
    <div
      className="pointer-events-none absolute top-2 z-10 rounded bg-slate-950/90 px-2 py-1 text-[11px] text-slate-200 shadow"
      style={{ left: tip.x + 12 }}
    >
      {tip.text}
    </div>
  )
}

// exportHtml produces a standalone Sigma.js (CDN) file for the current graph —
// CDN is used ONLY in the exported artifact, never in the app.
function exportHtml(g: Graph<GraphologyNode, GraphologyEdge>): void {
  const nodes: { id: string; label: string; x: number; y: number; size: number; color: string; title: string }[] = []
  for (const n of g.nodes()) {
    const a = g.getNodeAttributes(n) as unknown as Record<string, unknown>
    nodes.push({
      id: n,
      label: String(a.label ?? n),
      x: Number(a.x ?? 0),
      y: Number(a.y ?? 0),
      size: Number(a.size ?? 4),
      color: String(a.color ?? '#6366f1'),
      title: `${a.label} · ${a.nodeType} · ${a.path}`,
    })
  }
  const edges: { id: string; source: string; target: string }[] = []
  for (const e of g.edges()) {
    const [s, t] = g.extremities(e)
    edges.push({ id: e, source: s, target: t })
  }
  const html = `<!doctype html><html><head><meta charset="utf-8"><title>Skillgrid Graph</title>
<style>html,body{margin:0;height:100%;background:#0f172a}#g{width:100vw;height:100vh}</style>
<script src="https://unpkg.com/sigma@3/build/sigma.min.js"></script>
<script src="https://unpkg.com/graphology/dist/graphology.umd.min.js"></script></head>
<body><div id="g"></div><script>
var g = new graphology.Graph();
${JSON.stringify(nodes)}.forEach(function(n){g.addNode(n.id,{label:n.label,x:n.x,y:n.y,size:n.size,color:n.color,title:n.title})});
${JSON.stringify(edges)}.forEach(function(e){g.addEdge(e.source,e.target,{size:1})});
var s = new Sigma.default(g, document.getElementById('g'));
</script></body></html>`
  const blob = new Blob([html], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'skillgrid-graph.html'
  a.click()
  URL.revokeObjectURL(url)
}

// LoadingState shows a simple spinner while the graph payload is fetched.
// (The force layout no longer blocks the main thread — it animates in place
// after the graph is published, so there's no separate "organizing" full-block
// phase. tree/circles are instant.)
function LoadingState() {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-3 bg-slate-950 p-6">
      <div className="relative h-10 w-10">
        <div className="absolute inset-0 animate-spin rounded-full border-2 border-slate-700 border-t-indigo-400" />
      </div>
      <div className="text-sm text-slate-300">Loading graph…</div>
    </div>
  )
}

export function VectorGraph() {
  const { g, raw, loading, layoutStatus, stopLayout, error, layout, setLayout, degraded, truncated } =
    useMnemonicGraph()
  const [searchCenter, setSearchCenter] = useState<string | null>(null)
  const [depth, setDepth] = useState(2)
  const optimizing = layoutStatus === 'optimizing'

  const labels = useMemo(
    () => (g ? visibleLabels(g, filterGraphByDepth(g, searchCenter, depth)) : []),
    [g, searchCenter, depth],
  )

  const onSearch = (query: string) => {
    if (!g) return
    const found = searchNode(g, query)
    if (found) {
      setSearchCenter(found)
    }
  }

  if (loading) return <LoadingState />
  if (error) return <div className="p-6 text-sm text-red-400">Graph error: {error}</div>
  if (!g || !raw) return <div className="p-6 text-sm text-slate-400">No graph data.</div>

  // degraded: 005/008 edge data absent → node-only + file list fallback.
  if (degraded) {
    return (
      <div className="p-6">
        <div className="mb-3 rounded-lg border border-amber-700/50 bg-amber-900/20 px-3 py-2 text-xs text-amber-300">
          Degraded mode: no edge/community data (005/008). Showing nodes + file
          list fallback.
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
              Nodes ({raw.nodes.length})
            </div>
            <ul className="max-h-96 overflow-auto rounded-lg border border-slate-800 text-xs text-slate-300">
              {raw.nodes.map((n) => (
                <li key={n.id} className="truncate border-b border-slate-800/50 px-2 py-1">
                  {n.label} <span className="text-slate-500">· {n.type}</span>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
              Files ({raw.files?.length ?? 0})
            </div>
            <ul className="max-h-96 overflow-auto rounded-lg border border-slate-800 text-xs text-slate-300">
              {(raw.files ?? []).map((f) => (
                <li key={f} className="truncate border-b border-slate-800/50 px-2 py-1">
                  {f}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="relative h-full w-full overflow-hidden bg-slate-950">
      <SigmaContainer
        className="mnemonic-sigma h-full w-full"
        // Sigma reads the canvas background from the --sigma-background-color
        // CSS var (its style.css sets background: var(--sigma-background-color)).
        // Setting it inline on the co-occurring .react-sigma element overrides
        // the shipped default (light) without a specificity fight.
        style={{ ['--sigma-background-color' as string]: '#0f172a' }}
        settings={{
          allowInvalidContainer: true,
          defaultEdgeType: 'arrow',
          defaultNodeColor: '#0f172a',
          labelColor: { color: '#e2e8f0' },
          labelRenderedSizeThreshold: 6,
        }}
      >
        <Loader g={g} />
        <HoverEffects />
        <SearchHighlight center={searchCenter} />
        <HoverTooltip />
      </SigmaContainer>

      {/* top-left: search */}
      <div className="absolute left-3 top-3 z-10">
        <SearchPanel labels={labels} onSelect={onSearch} onClear={() => setSearchCenter(null)} />
      </div>

      {/* top-right: layout + stop + depth + export */}
      <div className="absolute right-3 top-3 z-10 flex flex-col items-end gap-2">
        <LayoutSwitcher value={layout} onChange={setLayout} />
        <StopLayoutButton visible={optimizing} onStop={stopLayout} />
        {searchCenter && (
          <DepthSlider value={depth} onChange={(d) => { setDepth(d) }} />
        )}
        <button
          type="button"
          onClick={() => exportHtml(g)}
          className="rounded-lg border border-slate-700 bg-slate-900/80 px-3 py-1 text-xs text-slate-200 hover:bg-slate-800"
        >
          Export HTML
        </button>
      </div>

      {/* bottom-left: legend */}
      <div className="absolute bottom-3 left-3 z-10">
        <Legend nodes={raw.nodes} />
      </div>

      {/* bottom-right: status */}
      <div className="absolute bottom-3 right-3 z-10 flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-900/80 px-3 py-1 text-[11px] text-slate-400">
        <span>
          {raw.nodes.length} nodes · {raw.edges.length} edges
          {truncated && <span className="ml-2 text-amber-400">truncated</span>}
        </span>
        <span
          className={
            optimizing
              ? 'flex items-center gap-1.5 text-amber-300'
              : 'text-emerald-400'
          }
        >
          {optimizing && (
            <span className="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-amber-300" aria-hidden />
          )}
          {optimizing ? 'Layout optimizing…' : 'Ready'}
        </span>
      </div>
    </div>
  )
}
