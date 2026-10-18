import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { KnowledgeGraphPage } from './KnowledgeGraphPage'
import { fetchKnowledgeGraph } from './knowledge-graph/api'

vi.mock('./knowledge-graph/api', () => ({
  fetchKnowledgeGraph: vi.fn(),
}))

const fixture = {
  nodes: [
    {
      id: 'decisions/d1',
      title: 'D-1',
      type: 'decision',
      description: 'Use Go stdlib only.',
      tags: ['dep'],
      status: 'active',
      orphan: false,
      deprecated: false,
    },
    {
      id: 'invariants/i1',
      title: 'I-1',
      type: 'invariant',
      description: 'Tests never write.',
      tags: [],
      status: 'deprecated',
      orphan: false,
      deprecated: true,
    },
    {
      id: 'reference/r1',
      title: 'R-1',
      type: 'reference',
      description: 'Terminology.',
      tags: [],
      status: 'active',
      orphan: true,
      deprecated: false,
    },
  ],
  edges: [{ source: 'decisions/d1', target: 'invariants/i1' }],
  backlinks: { 'invariants/i1': ['decisions/d1'] },
}

const empty = { nodes: [], edges: [], backlinks: {} }

beforeEach(() => {
  vi.mocked(fetchKnowledgeGraph).mockResolvedValue(fixture)
})

describe('KnowledgeGraphPage', () => {
  it('renders the concept graph (nodes, edges, legend)', async () => {
    render(<KnowledgeGraphPage />)
    await waitFor(() => expect(fetchKnowledgeGraph).toHaveBeenCalled())
    expect(screen.getByText('Knowledge Graph')).toBeTruthy()
    await waitFor(() => expect(document.querySelector('svg')).toBeTruthy())
    // concept type legend entries (rendered as "type (count)")
    expect(screen.getByText('decision (1)')).toBeTruthy()
    expect(screen.getByText('invariant (1)')).toBeTruthy()
    expect(screen.getByText('reference (1)')).toBeTruthy()
    // edge + orphan/deprecated summary
    expect(screen.getByText(/3 concepts \/ 1 links/)).toBeTruthy()
  })

  it('renders the selected concept frontmatter in the inspector', async () => {
    render(<KnowledgeGraphPage />)
    await waitFor(() => expect(document.querySelector('circle')).toBeTruthy())
    // The d3 tooltip lives in a child <title> element, so find the circle by
    // its tooltip text, not by an attribute on the circle itself.
    const d1 = Array.from(document.querySelectorAll('circle')).find(
      (c) => (c.querySelector('title')?.textContent ?? '').startsWith('D-1'),
    )
    expect(d1).toBeTruthy()
    d1?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(await screen.findByText('Use Go stdlib only.')).toBeTruthy()
    expect(screen.getByText('D-1')).toBeTruthy()
  })

  it('renders the inspector for a concept with null tags (omitted frontmatter)', async () => {
    // A concept with no `tags:` frontmatter is JSON null, not [] — the
    // inspector must not crash reading .length on it.
    vi.mocked(fetchKnowledgeGraph).mockResolvedValue({
      nodes: [
        {
          id: 'decisions/d2',
          title: 'D-2',
          type: 'decision',
          description: 'No tags on this one.',
          tags: null as unknown as string[],
          status: 'active',
          orphan: false,
          deprecated: false,
        },
      ],
      edges: [],
      backlinks: {},
    })
    render(<KnowledgeGraphPage />)
    await waitFor(() => expect(document.querySelector('circle')).toBeTruthy())
    const d2 = Array.from(document.querySelectorAll('circle')).find(
      (c) => (c.querySelector('title')?.textContent ?? '').startsWith('D-2'),
    )
    d2?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(await screen.findByText('No tags on this one.')).toBeTruthy()
  })

  it('renders EmptyState for an empty bundle (not an error)', async () => {
    vi.mocked(fetchKnowledgeGraph).mockResolvedValue(empty)
    render(<KnowledgeGraphPage />)
    expect(await screen.findByText(/no knowledge concepts/i)).toBeTruthy()
    expect(document.querySelector('svg')).toBeNull()
    expect(screen.queryByText('Knowledge Graph Error')).toBeNull()
  })

  it('shows an error state when the route fails', async () => {
    vi.mocked(fetchKnowledgeGraph).mockRejectedValue(new Error('graph down'))
    render(<KnowledgeGraphPage />)
    expect(await screen.findByText('graph down')).toBeTruthy()
  })
})
