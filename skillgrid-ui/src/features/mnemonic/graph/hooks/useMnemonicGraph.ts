import { useCallback, useEffect, useRef, useState } from 'react'
import type Graph from 'graphology'
import { mnemonicGraphToGraphology, type GraphologyEdge, type GraphologyNode } from '../converters'
import { applyLayout, forceLayoutAnimated, type AnimatedLayoutHandle, type LayoutKind } from '../layouts'
import { fetchGraph } from '../api'
import type { GraphResponse } from '../types'

export type LayoutStatus = 'idle' | 'optimizing' | 'ready'

export interface UseMnemonicGraph {
  g: Graph<GraphologyNode, GraphologyEdge> | null
  raw: GraphResponse | null
  loading: boolean
  // layingOut is true once the data has arrived but before the layout has been
  // applied (the fetch→first-frame gap) — lets the UI show a "loading" indicator.
  layingOut: boolean
  // layoutStatus reflects the active layout's progress: 'optimizing' while the
  // force simulation is settling, 'ready' when it's done (or for instant
  // tree/circles layouts, set immediately). 'idle' before any layout runs.
  layoutStatus: LayoutStatus
  // stopLayout interrupts the running force simulation, freezing it in place.
  // No-op when nothing is animating.
  stopLayout: () => void
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
// `force` runs the ANIMATED simulation (settle in real time + interruptible);
// `tree`/`circles` are instant (synchronous applyLayout, status → ready).
// Switching layout re-triggers a load (fresh graph + layout). `recenter`
// re-fetches a depth-limited neighborhood (the search "focus" flow).
export function useMnemonicGraph(): UseMnemonicGraph {
  const [g, setG] = useState<Graph<GraphologyNode, GraphologyEdge> | null>(null)
  const [raw, setRaw] = useState<GraphResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [layingOut, setLayingOut] = useState(false)
  const [layoutStatus, setLayoutStatus] = useState<LayoutStatus>('idle')
  const [error, setError] = useState<string | null>(null)
  const [layout, setLayoutState] = useState<LayoutKind>('force')
  const [focus, setFocus] = useState<{ id: number | null; depth: number }>({ id: null, depth: 2 })

  // The running animated-layout handle (force only). Kept in a ref so stopLayout
  // can reach the latest one without re-running the load effect.
  const animRef = useRef<AnimatedLayoutHandle | null>(null)

  const stopLayout = useCallback(() => {
    animRef.current?.stop()
  }, [])

  useEffect(() => {
    let cancelled = false
    const run = async () => {
      setLoading(true)
      setLayingOut(false)
      setLayoutStatus('idle')
      setError(null)
      // Stop any in-flight simulation from a previous load.
      animRef.current?.stop()
      animRef.current = null
      try {
        const res = await fetchGraph(focus.id != null ? { nodeId: focus.id, depth: focus.depth } : undefined)
        if (cancelled) return
        setRaw(res)
        // Data arrived; the layout below runs. Flip layingOut so the indicator
        // can render first (one frame), then build the graph + apply the layout.
        setLayingOut(true)
        await new Promise((r) => requestAnimationFrame(() => r(null)))
        const graph = mnemonicGraphToGraphology(res.nodes, res.edges)

        if (layout === 'force') {
          // Animated: publish the graph immediately (seeded positions) and let
          // the simulation settle in real time. The graph object is mutated in
          // place by the layout, so Sigma re-renders as it moves.
          setLayoutStatus('optimizing')
          const handle = forceLayoutAnimated(graph, {
            // SKILLGRID_GRAPH_MAXITER (default 300) bounds the sim; raise it for
            // slow, visibly-settling animation on large graphs (or to test the
            // Stop button, which freezes it mid-flight).
            maxIterations: Number(new URLSearchParams(location.search).get('maxiter')) || 300,
          })
          animRef.current = handle
          setG(graph)
          void handle.done.then(() => {
            if (!cancelled) setLayoutStatus('ready')
          })
        } else {
          // Instant layouts (tree/circles): synchronous, ready immediately.
          applyLayout(graph, layout)
          setLayoutStatus('ready')
          if (cancelled) return
          setG(graph)
        }
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
      animRef.current?.stop()
      animRef.current = null
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
    layoutStatus,
    stopLayout,
    error,
    layout,
    setLayout,
    recenter,
    degraded: raw?.degraded ?? false,
    truncated: raw?.truncated ?? false,
  }
}
