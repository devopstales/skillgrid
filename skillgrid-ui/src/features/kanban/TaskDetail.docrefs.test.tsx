import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { TaskDetail } from './TaskDetail'

const baseTask = {
  id: '003',
  title: 'Fix login',
  status: 'in-progress',
  type: 'bug',
  priority: 'high',
  board: 'in_progress',
  provider: 'backlogmd',
} as const

function mockResponse(body: unknown) {
  const text = JSON.stringify(body)
  return {
    ok: true,
    status: 200,
    text: async () => text,
    json: async () => body,
  }
}

function mockFetches(docRefs?: string[]) {
  vi.spyOn(globalThis, 'fetch').mockImplementation((async (input) => {
    const url = String(input)
    if (url.includes('/tasks/003/deps')) {
      return mockResponse({ task_id: '003', deps_in: [], deps_out: [] })
    }
    if (url.includes('/tasks/003')) {
      return mockResponse({ ...baseTask, doc_refs: docRefs })
    }
    return mockResponse({})
  }) as typeof fetch)
}

describe('TaskDetail doc_refs', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders doc_refs as links to the docs viewer', async () => {
    mockFetches(['.skillgrid/specs/003/briefing.md', '.skillgrid/ARCHITECTURE.md'])
    render(<TaskDetail task="003" onClose={() => {}} />)
    expect(await screen.findByText(/Documents/i)).toBeTruthy()
    const links = screen.getAllByRole('link')
    const hrefs = links.map((l) => l.getAttribute('href'))
    expect(hrefs).toContain(`/docs?file=${encodeURIComponent('.skillgrid/specs/003/briefing.md')}`)
    expect(hrefs).toContain(`/docs?file=${encodeURIComponent('.skillgrid/ARCHITECTURE.md')}`)
  })

  it('does not render a Documents section when doc_refs is absent', async () => {
    mockFetches(undefined)
    render(<TaskDetail task="003" onClose={() => {}} />)
    await screen.findByText('Fix login')
    expect(screen.queryByText(/Documents/i)).toBeNull()
  })
})
