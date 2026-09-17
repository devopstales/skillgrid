import { useCallback, useEffect, useState } from 'react'
import type Graph from 'graphology'
import { mnemonicGraphToGraphology, type GraphologyEdge, type GraphologyNode } from '../converters'
import { applyLayout, type LayoutKind } from '../layouts'
import { fetchGraph } from '../api'
import type { GraphResponse } from '../types'

export interface UseMnemonicGraph {
  g: Graph<GraphologyNode, GraphologyEdge> | null
  raw: GraphResponse | null
  loading: boolean
  // layingOut is true once the data has arrived but before the (synchronous)
  // layout has been applied — lets the UI show an "organizing" indicator during
  // the force/tree computation, which blocks the main thread.
  layingOut: boolean
  error: string | null
  layout: LayoutKind
  setLayout: (k: LayoutKind) => void
  recenter: (nodeId: number | null, depth?: number) => void
  degraded: boolean
  truncated: boolean
}

// useMnemonicGraph fetches the graph, converts it to a FRESH graphology
// instance per load (so React StrictMode's double-mount never double-loads the
// same graph and trips the multi-edge guard), and runs the active layout.
// Switching layout re-triggers a load (fresh graph + layout). `recenter`
// re-fetches a depth-limited neighborhood (the search "focus" flow).
export function useMnemonicGraph(): UseMnemonicGraph {
  const [g, setG] = useState<Graph<GraphologyNode, GraphologyEdge> | null>(null)
  const [raw, setRaw] = useState<GraphResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [layingOut, setLayingOut] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [layout, setLayoutState] = useState<LayoutKind>('force')
  const [focus, setFocus] = useState<{ id: number | null; depth: number }>({ id: null, depth: 2 })

  useEffect(() => {
    let cancelled = false
    const run = async () => {
      setLoading(true)
      setLayingOut(false)
      setError(null)
      try {
        const res = await fetchGraph(focus.id != null ? { nodeId: focus.id, depth: focus.depth } : undefined)
        if (cancelled) return
        setRaw(res)
        // Data arrived; the layout computation below is synchronous and blocks
        // the main thread. Flip layingOut so the indicator can render first
        // (one frame), then compute, then publish the graph.
        setLayingOut(true)
        await new Promise((r) => requestAnimationFrame(() => r(null)))
        const graph = mnemonicGraphToGraphology(res.nodes, res.edges)
        applyLayout(graph, layout)
        if (cancelled) return
        setG(graph)
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      } finally {
        if (!cancelled) {
          setLoading(false)
          setLayingOut(false)
        }
      }
    }
    run()
    return () => {
      cancelled = true
    }
  }, [focus, layout])

  const setLayout = useCallback((kind: LayoutKind) => {
    setLayoutState(kind)
  }, [])

  const recenter = useCallback((nodeId: number | null, depth = 2) => {
    setFocus({ id: nodeId, depth })
  }, [])

  return {
    g,
    raw,
    loading,
    layingOut,
    error,
    layout,
    setLayout,
    recenter,
    degraded: raw?.degraded ?? false,
    truncated: raw?.truncated ?? false,
  }
}
