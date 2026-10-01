import { fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { PlansPage } from './PlansPage'

const plans = {
  plans: [
    {
      name: '003-fix-login',
      status: 'PENDING',
      progress: 0.5,
      tasksDone: 1,
      tasksTotal: 2,
      hasLedger: true,
    },
  ],
}

const detail = {
  name: '003-fix-login',
  status: 'PENDING',
  progress: 0.5,
  tasksDone: 1,
  tasksTotal: 2,
  hasLedger: true,
  files: [],
  steps: [
    { raw: '- [x] step one', status: 'COMPLETE' },
    { raw: '- [ ] do task-003 then task-007', status: 'PENDING' },
  ],
  briefing: 'See #012 for the design.',
  tasks: '1. Implement task-003',
}

describe('PlansPage linked tasks', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('links task refs in ledger steps to the tracker deep link', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((async (input) => {
      const url = String(input)
      const body = url.includes('/plans/003') ? detail : plans
      const text = JSON.stringify(body)
      return { ok: true, status: 200, text: async () => text, json: async () => body }
    }) as typeof fetch)
    render(<PlansPage />)
    const card = await screen.findByRole('heading', { name: '003-fix-login' })
    fireEvent.click(card.closest('button') ?? card)
    await new Promise((r) => setTimeout(r, 400))
    const links = screen.getAllByRole('link')
    const hrefs = links.map((l) => l.getAttribute('href'))
    expect(hrefs).toContain('/tracker?task=003')
    expect(hrefs).toContain('/tracker?task=007')
    expect(hrefs).toContain('/tracker?task=012')
  })
})
