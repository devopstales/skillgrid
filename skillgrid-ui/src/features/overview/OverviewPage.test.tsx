import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { OverviewPage } from './OverviewPage'

// state.yaml body as /docs/content returns it for .skillgrid/state.yaml
// (pipeline.current_phase / current_change / status + progress.completed_changes).
function stateYaml(
  phase: string,
  change: string,
  status: string,
  completed: number,
): string {
  return [
    'schema: skillgrid/state/v1',
    'pipeline:',
    `  current_phase: "${phase}"`,
    `  current_change: "${change}"`,
    `  status: ${status}`,
    'progress:',
    `  completed_changes: ${completed}`,
    `  blocked_changes: 0`,
  ].join('\n')
}

const STAT_OK = {
  project: 'skillgrid',
  total: 42,
  byType: { commit: 42 },
  activeSessions: 3,
}
const CODE_FRESH = { file_count: 120, chunk_count: 480, last_indexed: '2026-09-30T10:00:00Z', stale: false }
const CODE_STALE = { file_count: 120, chunk_count: 480, last_indexed: '2026-09-01T10:00:00Z', stale: true }
const CODE_NOTINDEXED = { file_count: 0, chunk_count: 0, last_indexed: '', stale: true }
const EVENTS = {
  project: 'skillgrid',
  limit: 10,
  events: [
    {
      id: 1,
      ts: '2026-09-30T10:00:00Z',
      type: 'commit',
      source: 'git',
      severity: 'info',
      summary: 'feat(ui): wave 4 done',
      sessionId: 's1',
      relatedIds: [],
    },
    {
      id: 2,
      ts: '2026-09-30T11:00:00Z',
      type: 'memory',
      source: 'mnemonic',
      severity: 'info',
      summary: 'saved architecture note',
      sessionId: 's1',
      relatedIds: [],
    },
  ],
}
const EMPTY_EVENTS = { project: 'skillgrid', limit: 10, events: [] }

function json(v: unknown): Response {
  return new Response(JSON.stringify(v), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}
function notFound(): Response {
  return new Response(JSON.stringify({ error: 'doc not found' }), {
    status: 404,
    headers: { 'Content-Type': 'application/json' },
  })
}

// Mock the global fetch so resolveProject (project/current → activity/stats →
// /projects) settles and every overview endpoint returns the given fixtures.
function mockFetch(
  overrides: Partial<{
    stats: unknown
    code: unknown
    state: string
    stateStatus: number
    events: unknown
  }> = {},
) {
  const { stats, code, state, stateStatus, events } = overrides
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input.toString()
    if (url.includes('/project/current')) {
      return Promise.resolve(json({ project: 'skillgrid' })) as unknown as Promise<Response>
    }
    if (url.includes('/activity/stats')) {
      const body = stats === undefined ? STAT_OK : stats
      return Promise.resolve(json(body)) as unknown as Promise<Response>
    }
    if (url.includes('/activity/events')) {
      const body = events === undefined ? EVENTS : events
      return Promise.resolve(json(body)) as unknown as Promise<Response>
    }
    if (url.includes('/code/status')) {
      const body = code === undefined ? CODE_FRESH : code
      return Promise.resolve(json(body)) as unknown as Promise<Response>
    }
    if (url.includes('/docs/content')) {
      if (stateStatus === 404) return Promise.resolve(notFound()) as unknown as Promise<Response>
      const body = state === undefined ? stateYaml('reflect', '', 'idle', 19) : state
      return Promise.resolve(json({ path: '.skillgrid/state.yaml', body })) as unknown as Promise<Response>
    }
    return Promise.resolve(json({ projects: ['skillgrid'] })) as unknown as Promise<Response>
  })
}

function textContent(el: Element | null | undefined): string {
  return el?.textContent ?? ''
}

describe('OverviewPage', () => {
  beforeEach(() => vi.restoreAllMocks())
  afterEach(() => vi.restoreAllMocks())

  it('renders the KPI cards with values from the mocked endpoints', async () => {
    mockFetch()
    render(<OverviewPage />)

    expect(await screen.findByTestId('kpi-active-sessions-value')).toBeTruthy()
    expect(textContent(screen.getByTestId('kpi-active-sessions-value'))).toMatch(/\b3\b/)
    expect(textContent(screen.getByTestId('kpi-active-sessions-label'))).toMatch(/active sessions/i)

    expect(textContent(screen.getByTestId('kpi-total-events-value'))).toMatch(/\b42\b/)
    expect(textContent(screen.getByTestId('kpi-total-events-label'))).toMatch(/total events/i)

    expect(textContent(screen.getByTestId('kpi-code-index-value'))).toMatch(/fresh/i)
    expect(textContent(screen.getByTestId('kpi-code-index-label'))).toMatch(/code index/i)

    expect(textContent(screen.getByTestId('kpi-pipeline-phase-value'))).toMatch(/reflect/)
    expect(textContent(screen.getByTestId('kpi-pipeline-phase-label'))).toMatch(/pipeline phase/i)
  })

  it('shows the Code Index card green when the index is fresh', async () => {
    mockFetch({ code: CODE_FRESH })
    render(<OverviewPage />)
    await screen.findByTestId('kpi-code-index-value')
    const el = screen.getByTestId('kpi-code-index-value')
    expect(textContent(el)).toMatch(/fresh/i)
    expect(el.className).toMatch(/text-accent/)
  })

  it('shows the Code Index card amber when the index is stale', async () => {
    mockFetch({ code: CODE_STALE })
    render(<OverviewPage />)
    await screen.findByTestId('kpi-code-index-value')
    const el = screen.getByTestId('kpi-code-index-value')
    expect(textContent(el)).toMatch(/stale/i)
    expect(el.className).toMatch(/text-warn/)
  })

  it('shows the Code Index card red when the code is not indexed', async () => {
    mockFetch({ code: CODE_NOTINDEXED })
    render(<OverviewPage />)
    await screen.findByTestId('kpi-code-index-value')
    const el = screen.getByTestId('kpi-code-index-value')
    expect(textContent(el)).toMatch(/not indexed/i)
    expect(el.className).toMatch(/text-danger/)
  })

  it('shows the Pipeline Phase card with the change name when present', async () => {
    mockFetch({ state: stateYaml('slicing', 'wave-5-overview', 'running', 19) })
    render(<OverviewPage />)
    await screen.findByTestId('kpi-pipeline-phase-value')
    const el = screen.getByTestId('kpi-pipeline-phase-value')
    expect(textContent(el)).toMatch(/slicing/)
    expect(textContent(el)).toMatch(/wave-5-overview/)
  })

  it('renders the Recent Activity list with event summaries', async () => {
    mockFetch()
    render(<OverviewPage />)

    expect(await screen.findByText('feat(ui): wave 4 done')).toBeTruthy()
    expect(screen.getByText('saved architecture note')).toBeTruthy()
    // Event type labels render.
    expect(screen.getByText('commit')).toBeTruthy()
    expect(screen.getByText('memory')).toBeTruthy()
  })

  it('shows the Recent Activity empty state when there are no events', async () => {
    mockFetch({ events: EMPTY_EVENTS })
    render(<OverviewPage />)

    expect(await screen.findByText(/no recent activity/i)).toBeTruthy()
  })

  it('renders the Pipeline Status block with phase, change, and completed count', async () => {
    mockFetch({ state: stateYaml('slicing', 'wave-5-overview', 'running', 19) })
    render(<OverviewPage />)

    await screen.findByTestId('pipeline-status')
    const block = screen.getByTestId('pipeline-status')
    expect(textContent(block)).toMatch(/slicing/)
    expect(textContent(block)).toMatch(/wave-5-overview/)
    expect(textContent(screen.getByTestId('pipeline-completed-value'))).toMatch(/\b19\b/)
  })

  it('shows a Pipeline Status fallback when state.yaml is unavailable', async () => {
    mockFetch({ stateStatus: 404 })
    render(<OverviewPage />)

    expect(await screen.findByText(/pipeline state unavailable/i)).toBeTruthy()
  })
})
