import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { GraphPage } from './GraphPage'
import { fetchGraph } from './graph/api'
import type { GraphResponse } from './graph/types'
import { apiGet } from '../../lib/api'

vi.mock('./graph/api', () => ({
  fetchGraph: vi.fn(),
}))

vi.mock('../../lib/api', () => ({
  apiGet: vi.fn(),
}))

// The page fetches /code/status and /knowledge/graph through apiGet; the mock
// dispatches by path so the concept-toggle / status feeds stay honest.
vi.mocked(apiGet).mockImplementation(async (path: string) => {
  if (path === '/code/status') return null
  if (path === '/knowledge/graph')
    return {
      nodes: [
        {
          id: 'workflows/sdd',
          title: 'SDD workflow',
          type: 'workflow',
          description: 'spec → build → verify',
          tags: ['sdd'],
          status: 'active',
          orphan: false,
          deprecated: false,
        },
        {
          id: 'decisions/adr-0031',
          title: 'Go 1.22 floor',
          type: 'decision',
          description: 'Go 1.22+ to build.',
          tags: ['deps'],
          status: 'active',
          orphan: true,
          deprecated: false,
        },
      ],
      edges: [{ source: 'decisions/adr-0031', target: 'workflows/sdd' }],
      backlinks: { 'workflows/sdd': ['decisions/adr-0031'] },
    }
  return null
})

const node = (
  id: number,
  label: string,
  extra: Record<string, unknown> = {},
): Record<string, unknown> => ({
  id,
  uid: `sym:${label}`,
  label,
  type: 'function',
  language: 'go',
  path: 'demo/a.go',
  degree: 0,
  community: 0,
  ...extra,
})

const graph = {
  nodes: [
    node(1, 'main', {
      degree: 4,
      lines: { start: 3, end: 12 },
      signature: 'func main()',
    }),
    node(2, 'alpha', { degree: 1 }),
    node(3, 'beta', { degree: 1 }),
    node(4, 'gamma', { degree: 1 }),
    node(5, 'delta', { degree: 1 }),
    node(6, 'orphaned', { type: 'type', community: -1 }),
  ],
  edges: [
    { id: 'calls:1->2', source: 1, target: 2, type: 'calls', weight: 1, directed: true, line: 5 },
    { id: 'calls:1->3', source: 1, target: 3, type: 'calls', weight: 1, directed: true, line: 6 },
    { id: 'calls:2->4', source: 2, target: 4, type: 'calls', weight: 1, directed: true, line: 9 },
    { id: 'imports:1->6', source: 1, target: 6, type: 'imports', weight: 1, directed: true, line: 2 },
  ],
  truncated: false,
  degraded: false,
  project: 'skillgrid',
}

// no edges at all (degraded: the server serves node-only + a file list)
const degradedGraph = {
  nodes: [
    { ...node(1, 'alpha'), path: 'demo/a.go' },
    { ...node(2, 'beta'), path: 'demo/b.go' },
  ],
  edges: [],
  truncated: false,
  degraded: true,
  files: ['demo/a.go', 'demo/b.go'],
  project: 'skillgrid',
}

// wait for the page to have finished loading its graph (the tree renders).
const loaded = () => screen.findByText('Files')

beforeEach(() => {
  vi.mocked(fetchGraph).mockResolvedValue(graph as unknown as GraphResponse)
})

describe('GraphPage [teaching-graph-page G14]', () => {
  it('requests at most 500 nodes and renders the page', async () => {
    render(<GraphPage />)
    await waitFor(() => expect(fetchGraph).toHaveBeenCalledWith({ limit: 500 }))
    expect(screen.getByText('Node Inspector')).toBeTruthy()
    await waitFor(() => expect(document.querySelector('svg')).toBeTruthy())
  })

  it('shows an index-status error without hiding the graph', async () => {
    vi.mocked(apiGet).mockRejectedValueOnce(new Error('index down'))
    render(<GraphPage />)
    expect(await screen.findByText('index down')).toBeTruthy()
    expect(screen.getByText('Node Inspector')).toBeTruthy()
  })

  it('shows the file tree from the served nodes and syncs selection both ways', async () => {
    render(<GraphPage />)
    // tree panel lists the served file paths
    expect(await loaded()).toBeTruthy()
    const tree = screen.getByText('Files').parentElement
    expect(tree).toBeTruthy()
    expect(tree!.textContent).toContain('demo/a.go')
    const file = screen.getAllByText('demo/a.go', { exact: false })[0]
    expect(file).toBeTruthy()
    // expanding a file focuses the graph on its nodes (focus label shown)
    fireEvent.click(file)
    await waitFor(() =>
      expect(screen.getByText(/focused on/i)).toBeTruthy(),
    )
  })

  it('inspector shows identity with lines, signature, and typed connections with source lines', async () => {
    render(<GraphPage />)
    await loaded()
    // pick a node via the tree's symbol list (main is in demo/a.go)
    // expand the tree file so its symbol rows render, then pick 'main'
    const tree = screen.getByText('Files').parentElement
    expect(tree!.textContent).toContain('demo/a.go')
    const fileRow = screen.getAllByText('demo/a.go', { exact: false })[0]
    fireEvent.click(fileRow)
    await waitFor(() => expect(screen.getAllByText('main').length).toBeGreaterThan(0))
    const symbols = screen.getAllByText('main')
    expect(symbols.length).toBeGreaterThan(0)
    fireEvent.click(symbols[0])
    // identity block (the signature renders in the identity grid AND the
    // ephemeral-locals preview → expect both)
    expect(await screen.findByText(/lines 3-12/i)).toBeTruthy()
    expect(screen.getAllByText('func main()').length).toBeGreaterThanOrEqual(2)
    // typed connection groups: 'calls' and 'imports' as separate groups
    expect(screen.getByText('calls (2)')).toBeTruthy()
    expect(screen.getByText('imports (1)')).toBeTruthy()
    // per-edge source line on the connection rows
    expect(screen.getAllByText(/L5/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/L6/).length).toBeGreaterThan(0)
  })

  it('search centers a named symbol', async () => {
    render(<GraphPage />)
    await loaded()
    const input = screen.getByPlaceholderText(/search symbols/i)
    fireEvent.change(input, { target: { value: 'beta' } })
    await waitFor(() =>
      expect(screen.getByText(/focused on beta/i)).toBeTruthy(),
    )
  })

  it('filters by kind and caps uncategorized (newcomer) nodes', async () => {
    render(<GraphPage />)
    await loaded()
    // kind filter: functions only → the 'type'-kind node is hidden
    const kindSelect = screen.getByLabelText(/kind/i)
    fireEvent.change(kindSelect, { target: { value: 'function' } })
    await waitFor(() =>
      expect(screen.getByText(/1 hidden/i)).toBeTruthy(),
    )
  })

  it('toggles the concept overlay on and off', async () => {
    render(<GraphPage />)
    await loaded()
    const toggle = screen.getByRole('switch', { name: /concepts/i })
    expect(toggle).toBeTruthy()
    fireEvent.click(toggle)
    await waitFor(() =>
      expect(screen.getByText('Concepts (2)')).toBeTruthy(),
    )
    fireEvent.click(toggle)
    await waitFor(() =>
      expect(screen.queryByText('Concepts (2)')).toBeNull(),
    )
  })

  it('runs a client-side tour with at least 3 stops', async () => {
    render(<GraphPage />)
    await loaded()
    fireEvent.click(screen.getByRole('button', { name: /start tour/i }))
    // stop 1: workflow
    expect(await screen.findByText(/stop 1 of/i)).toBeTruthy()
    expect(screen.getByText(/Start at the entry point/i)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /next stop/i }))
    // stop 2: decisions
    await waitFor(() => expect(screen.getByText(/stop 2 of/i)).toBeTruthy())
    fireEvent.click(screen.getByRole('button', { name: /next stop/i }))
    // stop 3: top centrality
    await waitFor(() => expect(screen.getByText(/stop 3 of/i)).toBeTruthy())
    expect(screen.getByText(/top centrality/i)).toBeTruthy()
    // the tour completes (finishes cleanly)
    fireEvent.click(screen.getByRole('button', { name: /next stop/i }))
    await waitFor(() => expect(screen.queryByText(/stop \d of/i)).toBeNull())
  })

  it('renders a degraded state (no edges) with the file list, not an error or blank', async () => {
    vi.mocked(fetchGraph).mockResolvedValue(degradedGraph as unknown as GraphResponse)
    render(<GraphPage />)
    expect((await screen.findAllByText(/degraded: no edges/i)).length).toBeGreaterThan(0)
    // served file list is rendered (demo/b.go appears in tree + banner)
    expect(screen.getAllByText('demo/b.go', { exact: false }).length).toBeGreaterThan(0)
    // the inspector hint still renders (not blank)
    expect(screen.getByText('Node Inspector')).toBeTruthy()
  })

  it('renders an empty-filters state (all kinds filtered out)', async () => {
    render(<GraphPage />)
    await loaded()
    const kindSelect = screen.getByLabelText(/kind/i)
    // filtering to a kind with no nodes leaves zero visible nodes → a state,
    // not a blank canvas.
    fireEvent.change(kindSelect, { target: { value: 'route' } })
    await waitFor(() =>
      expect(screen.getByText(/no symbols match the filters/i)).toBeTruthy(),
    )
  })

  it('renders a missing-locals state in the inspector (degraded, not an error)', async () => {
    // a node without lines → the inspector degrades the locals section.
    vi.mocked(fetchGraph).mockResolvedValue({
      ...graph,
      nodes: [
        node(1, 'main', { degree: 0, lines: { start: 3, end: 12 } }),
        node(2, 'mystery', { degree: 0, community: -1 }),
      ],
      edges: [],
    } as unknown as GraphResponse)
    render(<GraphPage />)
    await loaded()
    // 'mystery' has no lines field → clicking it shows the degraded state
    const tree = screen.getByText('Files').parentElement
    expect(tree!.textContent).toContain('demo/a.go')
    const fileRow = screen.getAllByText('demo/a.go', { exact: false })[0]
    fireEvent.click(fileRow)
    await waitFor(() => expect(screen.getAllByText('mystery').length).toBeGreaterThan(0))
    fireEvent.click(screen.getAllByText('mystery')[0])
    expect(await screen.findByText(/locals unavailable/i)).toBeTruthy()
  })
})
