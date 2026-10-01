import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { DecisionsPage } from './DecisionsPage'
import * as api from './api'

// The page polls on a 4s interval. To test the interval without holding the
// suite open, we spy on the global setInterval the component installs and
// invoke the captured callback by hand (real timers, so findBy* waits work).
function renderPolling() {
  const intervalSpy = vi
    .spyOn(globalThis, 'setInterval')
    .mockImplementation(() => 1 as unknown as ReturnType<typeof setInterval>)
  const utils = render(<DecisionsPage />)
  const flush = () => {
    for (const [arg] of intervalSpy.mock.calls as unknown as Array<[(cb: unknown) => void]>) {
      if (typeof arg === 'function') (arg as () => void)()
    }
  }
  return { ...utils, flush, intervalSpy }
}

let fetchSpy: Mock

function row(id: number, state: 'pending' | 'answered'): api.DecisionRow {
  return {
    id,
    topicKey: `demo/decision-${id}`,
    title: `Decision ${id}`,
    createdAt: '2026-09-19T10:00:00Z',
    updatedAt: '2026-09-19T10:00:00Z',
    visibility: 'team',
    content: {
      question: `Question ${id}?`,
      options: [
        { id: 'a', label: `Option A ${id}` },
        { id: 'b', label: `Option B ${id}` },
      ],
      recommended: 'a',
      state,
      ...(state === 'answered' ? { answeredOption: 'b' } : {}),
    },
  }
}

describe('DecisionsPage', () => {
  beforeEach(() => {
    fetchSpy = vi.spyOn(api, 'fetchDecisions').mockResolvedValue([])
    vi.spyOn(api, 'answerDecision').mockResolvedValue(undefined)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the pending decision inbox (question + options + created)', async () => {
    fetchSpy.mockResolvedValue([row(1, 'pending')])
    render(<DecisionsPage />)
    expect(await screen.findByText('Question 1?')).toBeTruthy()
    expect(screen.getByText('Option A 1')).toBeTruthy()
    expect(screen.getByText('Option B 1')).toBeTruthy()
    expect(screen.getByText(/2026-09-19/)).toBeTruthy()
  })

  it('fetches pending by default', async () => {
    render(<DecisionsPage />)
    await screen.findByText('No pending decisions.')
    expect(fetchSpy).toHaveBeenCalledWith('pending')
  })

  it('refreshes the inbox on an interval', async () => {
    fetchSpy.mockResolvedValue([row(1, 'pending')])
    const { flush } = renderPolling()
    await screen.findByText('Question 1?')
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    flush()
    expect(fetchSpy).toHaveBeenCalledTimes(2)
  })

  it('shows a stored answer for an answered decision', async () => {
    fetchSpy.mockResolvedValue([row(2, 'answered')])
    render(<DecisionsPage />)
    await screen.findByText('Question 2?')
    expect(screen.getByText(/your choice/i)).toBeTruthy()
  })

  // The "open" tab reads .skillgrid/ASSUMPTIONS.md from /docs/content and
  // renders its Open Questions section. The section in the real file is a
  // numbered list: `1. **Title** — detail`.
  const ASSUMPTIONS_BODY = `# Assumptions

Some other section.

## Open Questions

Questions that are not yet decided and not yet spiked.

1. **v1.1 dashboard prioritization** — Memories+Sessions-first vs. Graph+Tracker-first. Decided at v1.1 scoping, not now.
2. **MCP tool-contract versioning** — is the secondary persona strong enough to warrant stable, versioned MCP tool contract? Assumed no in v1.0.
3. **README alignment** — the README is still installer-first; aligning it to the engine-first PRD is a follow-up ticket.
`

  function stubContent(body: string, opts: { status?: number } = {}) {
    const { status = 200 } = opts
    const payload = status === 200
      ? { path: '.skillgrid/ASSUMPTIONS.md', body }
      : { error: 'doc not found' }
    return vi
      .spyOn(globalThis, 'fetch')
      .mockImplementation((input) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('ASSUMPTIONS.md')) {
          return Promise.resolve(
            new Response(JSON.stringify(payload), {
              status,
              headers: { 'Content-Type': 'application/json' },
            }),
          ) as unknown as Promise<Response>
        }
        // Anything else (the decisions bridge) resolves to nothing.
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200 })) as unknown as Promise<Response>
      })
  }

  it('shows an open tab that lists the Open Questions from ASSUMPTIONS.md', async () => {
    fetchSpy.mockResolvedValue([])
    stubContent(ASSUMPTIONS_BODY)
    render(<DecisionsPage />)

    // The open tab is present.
    const openTab = screen.getByRole('tab', { name: 'open' })
    fireEvent.click(openTab)

    // Each open question's title renders.
    expect(await screen.findByText(/v1.1 dashboard prioritization/)).toBeTruthy()
    expect(screen.getByText(/MCP tool-contract versioning/)).toBeTruthy()
    expect(screen.getByText(/README alignment/)).toBeTruthy()
  })

  it('open tab fetches the ASSUMPTIONS.md content path', async () => {
    fetchSpy.mockResolvedValue([])
    const fetchSpyOpen = stubContent(ASSUMPTIONS_BODY)
    render(<DecisionsPage />)
    const openTab = screen.getByRole('tab', { name: 'open' })
    fireEvent.click(openTab)
    await screen.findByText(/v1.1 dashboard prioritization/)

    const called = fetchSpyOpen.mock.calls.some(([input]) => {
      const url = typeof input === 'string' ? input : input.toString()
      return url.includes('ASSUMPTIONS.md')
    })
    expect(called).toBeTruthy()
  })

  it('open tab shows an empty state when the section has no questions', async () => {
    fetchSpy.mockResolvedValue([])
    stubContent(`# Assumptions\n\n## Open Questions\n\n(decided, nothing left)\n`)
    render(<DecisionsPage />)
    const openTab = screen.getByRole('tab', { name: 'open' })
    fireEvent.click(openTab)

    expect(await screen.findByText(/no open questions/i)).toBeTruthy()
  })

  it('open tab shows an error when ASSUMPTIONS.md is missing', async () => {
    fetchSpy.mockResolvedValue([])
    stubContent('', { status: 404 })
    render(<DecisionsPage />)
    const openTab = screen.getByRole('tab', { name: 'open' })
    fireEvent.click(openTab)

    expect(await screen.findByText(/doc not found/i)).toBeTruthy()
  })
})
