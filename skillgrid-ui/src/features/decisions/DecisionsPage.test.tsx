import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { DecisionsPage } from './DecisionsPage'
import * as api from './api'

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
    vi.useFakeTimers()
    vi.spyOn(api, 'fetchDecisions').mockResolvedValue([])
    vi.spyOn(api, 'answerDecision').mockResolvedValue()
  })
  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('renders the pending decision inbox (question + options + created)', async () => {
    api.fetchDecisions.mockResolvedValue([row(1, 'pending')])
    render(<DecisionsPage />)
    expect(await screen.findByText('Question 1?')).toBeTruthy()
    expect(screen.getByText('Option A 1')).toBeTruthy()
    expect(screen.getByText('Option B 1')).toBeTruthy()
    expect(screen.getByText(/2026-09-19/)).toBeTruthy()
  })

  it('fetches pending by default', async () => {
    render(<DecisionsPage />)
    await waitFor(() => expect(api.fetchDecisions).toHaveBeenCalledWith('pending'))
  })

  it('refreshes the inbox on an interval', async () => {
    render(<DecisionsPage />)
    await waitFor(() => expect(api.fetchDecisions).toHaveBeenCalledTimes(1))
    vi.advanceTimersByTime(4000)
    await waitFor(() => expect(api.fetchDecisions).toHaveBeenCalledTimes(2))
  })

  it('shows a stored answer for an answered decision', async () => {
    api.fetchDecisions.mockResolvedValue([row(2, 'answered')])
    render(<DecisionsPage />)
    await screen.findByText('Question 2?')
    expect(screen.getByText(/your choice/i)).toBeTruthy()
  })
})
