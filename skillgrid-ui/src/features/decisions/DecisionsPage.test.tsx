import { render, screen } from '@testing-library/react'
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
})
