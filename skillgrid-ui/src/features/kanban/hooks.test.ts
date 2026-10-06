import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { useTrackerBoard } from './hooks'

// Fake EventSource: records listeners by event name so a test can fire a
// synthetic SSE event and assert the hook re-fetches.
type ESListener = () => void
const CONNECTING = 0
const OPEN = 1
const CLOSED = 2

class FakeEventSource {
  static instances: FakeEventSource[] = []
  static CONNECTING = CONNECTING
  static OPEN = OPEN
  static CLOSED = CLOSED
  listeners: Record<string, ESListener[]> = {}
  readyState = OPEN
  closed = false
  onerror: (() => void) | null = null

  constructor(_url: string) {
    FakeEventSource.instances.push(this)
  }
  addEventListener(name: string, fn: ESListener) {
    ;(this.listeners[name] ??= []).push(fn)
  }
  removeEventListener(name: string, fn: ESListener) {
    this.listeners[name] = (this.listeners[name] ?? []).filter((l) => l !== fn)
  }
  emit(name: string) {
    for (const fn of this.listeners[name] ?? []) fn()
  }
  close() {
    this.closed = true
    this.readyState = CLOSED
  }
}

const TASKS = { tasks: [], provider: 'backlogmd' }
const MILESTONES: { milestones: any[]; provider: string } = {
  milestones: [{ id: 'm-6', title: 'kanban-milestones' }],
  provider: 'backlogmd',
}

function ok(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    text: async () => JSON.stringify(body),
  } as Response
}

beforeAll(() => {
  vi.stubGlobal('EventSource', FakeEventSource)
})

afterEach(() => {
  vi.restoreAllMocks()
  FakeEventSource.instances = []
})

describe('useTrackerBoard SSE re-fetch', () => {
  it('re-fetches tasks and milestones on a milestones-changed event', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(ok({ connected: true, reason: '' })) // initial fetchProviders
        .mockResolvedValueOnce(ok(TASKS)) // initial fetchTasks
        .mockResolvedValueOnce(ok(MILESTONES)) // initial fetchMilestones
        .mockResolvedValueOnce(ok({ connected: true, reason: '' })) // re-fetch fetchProviders
        .mockResolvedValueOnce(ok(TASKS)) // re-fetch fetchTasks
        .mockResolvedValueOnce(ok(MILESTONES)), // re-fetch fetchMilestones
    )

    const { result } = renderHook(() => useTrackerBoard('backlogmd'))
    await act(async () => {
      await Promise.resolve()
    })

    // Initial load consumed 3 fetches.
    const initialFetches = vi.mocked(fetch).mock.calls.length
    expect(initialFetches).toBe(3)

    const es = FakeEventSource.instances[0]
    expect(es).toBeDefined()

    // Fire a synthetic milestones-changed event.
    await act(async () => {
      es.emit('milestones-changed')
      await Promise.resolve()
    })

    // The hook must have re-fetched (fetchProviders + tasks + milestones).
    expect(vi.mocked(fetch).mock.calls.length).toBe(initialFetches + 3)
    // The re-fetch re-asserted the ready state.
    expect(result.current.state.status).toBe('ready')
    if (result.current.state.status === 'ready') {
      expect(result.current.state.provider).toBe('backlogmd')
    }
  })

  it('still re-fetches on a tasks-changed event (no regression)', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(ok({ connected: true, reason: '' }))
        .mockResolvedValueOnce(ok(TASKS))
        .mockResolvedValueOnce(ok(MILESTONES))
        .mockResolvedValueOnce(ok({ connected: true, reason: '' }))
        .mockResolvedValueOnce(ok(TASKS))
        .mockResolvedValueOnce(ok(MILESTONES)),
    )

    const { result } = renderHook(() => useTrackerBoard('backlogmd'))
    await act(async () => {
      await Promise.resolve()
    })
    const initialFetches = vi.mocked(fetch).mock.calls.length
    expect(initialFetches).toBe(3)

    await act(async () => {
      FakeEventSource.instances[0].emit('tasks-changed')
      await Promise.resolve()
    })
    expect(vi.mocked(fetch).mock.calls.length).toBe(initialFetches + 3)
    expect(result.current.state.status).toBe('ready')
  })

  it('does not re-fetch on an unrelated event', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(ok({ connected: true, reason: '' }))
        .mockResolvedValueOnce(ok(TASKS))
        .mockResolvedValueOnce(ok(MILESTONES)),
    )

    renderHook(() => useTrackerBoard('backlogmd'))
    await act(async () => {
      await Promise.resolve()
    })
    const initialFetches = vi.mocked(fetch).mock.calls.length

    await act(async () => {
      FakeEventSource.instances[0].emit('unrelated-event')
      await Promise.resolve()
    })
    expect(vi.mocked(fetch).mock.calls.length).toBe(initialFetches)
  })
})
