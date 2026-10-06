import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchMilestones, fetchProviders, fetchTasks, TrackerError } from './api'
import type { Milestone, LoadState, ProviderName } from './types'

// useTrackerBoard loads the board for one provider and keeps it live via the
// /tracker/stream SSE endpoint. A file change on disk (Backlog.md) triggers a
// re-fetch, so the board updates without a manual reload.
export function useTrackerBoard(provider: ProviderName) {
  const [state, setState] = useState<LoadState>({ status: 'loading' })
  const [milestones, setMilestones] = useState<Milestone[]>([])
  const [live, setLive] = useState(false)
  const reqRef = useRef(0)

  const load = useCallback(async () => {
    const req = ++reqRef.current
    try {
      const info = await fetchProviders(provider)
      if (req !== reqRef.current) return
      if (!info.connected) {
        setState({
          status: 'degraded',
          provider,
          reason: info.reason ?? 'not connected',
          httpStatus: 503,
        })
        return
      }
      const [list, ms] = await Promise.all([
        fetchTasks(provider),
        fetchMilestones(provider),
      ])
      if (req !== reqRef.current) return
      setState({ status: 'ready', tasks: list.tasks, provider: list.provider })
      setMilestones(ms)
    } catch (err) {
      if (req !== reqRef.current) return
      if (err instanceof TrackerError && (err.status === 501 || err.status === 503)) {
        setState({
          status: 'degraded',
          provider,
          reason: err.message,
          httpStatus: err.status,
        })
      } else {
        setState({
          status: 'error',
          message: err instanceof Error ? err.message : 'failed to load',
        })
      }
    }
  }, [provider])

  useEffect(() => {
    setState({ status: 'loading' })
    void load()
  }, [load])

  // Live updates: subscribe to the SSE stream; on any tasks-changed event,
  // re-fetch the board. Only the Backlog.md provider emits file-change events
  // (fsnotify on .backlog/tasks/), so other providers simply stay static.
  useEffect(() => {
    let es: EventSource | null = null
    let disposed = false
    try {
      es = new EventSource('/tracker/stream')
      es.addEventListener('ready', () => {
        if (!disposed) setLive(true)
      })
      es.addEventListener('tasks-changed', () => {
        void load()
      })
      es.onerror = () => {
        // EventSource auto-reconnects; mark not-live if it stays closed.
        if (es?.readyState === EventSource.CLOSED) setLive(false)
      }
    } catch {
      setLive(false)
    }
    return () => {
      disposed = true
      es?.close()
    }
  }, [load])

  return { state, milestones, live, reload: load }
}

// useTaskDrawer holds the selected task id (deep-linked via ?task=).
export function useTaskDrawer() {
  const params = new URLSearchParams(window.location.search)
  const [selected, setSelected] = useState<string | null>(
    params.get('task') ?? null,
  )

  const open = useCallback((id: string) => {
    setSelected(id)
    const url = new URL(window.location.href)
    url.searchParams.set('task', id)
    window.history.replaceState(null, '', url.toString())
  }, [])

  const close = useCallback(() => {
    setSelected(null)
    const url = new URL(window.location.href)
    url.searchParams.delete('task')
    window.history.replaceState(null, '', url.toString())
  }, [])

  return { selected, open, close }
}
