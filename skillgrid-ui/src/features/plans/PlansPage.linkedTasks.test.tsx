import { fireEvent, render, screen, within } from '@testing-library/react'
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
    {
      name: '002-shipped-thing',
      status: 'done',
      progress: 1,
      tasksDone: 4,
      tasksTotal: 4,
      hasLedger: false,
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

function mockFetch() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((async (input) => {
    const url = String(input)
    const body = url.includes('/plans/003') ? detail : plans
    const text = JSON.stringify(body)
    return { ok: true, status: 200, text: async () => text, json: async () => body }
  }) as typeof fetch)
}

describe('PlansPage changes list', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    window.history.replaceState({}, '', '/plans')
  })

  it('renders each plan as a clickable row with a gate chip', async () => {
    mockFetch()
    render(<PlansPage />)
    const list = await screen.findByRole('group', { name: 'Changes' })
    const rows = within(list).getAllByTestId('change-row')
    expect(rows).toHaveLength(2)
    expect(rows.every((r) => r.tagName === 'BUTTON')).toBe(true)
    const chips = within(list).getAllByTestId('plan-status-chip')
    expect(chips.map((c) => c.getAttribute('data-kind'))).toEqual(['pending', 'pass'])
  })

  it('clicking a row selects it and loads the detail pane', async () => {
    mockFetch()
    render(<PlansPage />)
    const list = await screen.findByRole('group', { name: 'Changes' })
    const row = within(list).getByRole('button', { name: /003-fix-login/ })
    fireEvent.click(row)
    expect(row.getAttribute('aria-current')).toBe('true')
    expect(await screen.findByText(/step one/)).toBeTruthy()
    expect(window.location.search).toBe('?change=003-fix-login')
  })

  it('shows the pipeline when nothing is selected and its nodes are clickable', async () => {
    mockFetch()
    render(<PlansPage />)
    const pipeline = await screen.findByTestId('plan-pipeline')
    const node = within(pipeline).getByRole('button', { name: /003-fix-login/ })
    fireEvent.click(node)
    expect(await screen.findByText(/step one/)).toBeTruthy()
    expect(screen.queryByTestId('plan-pipeline')).toBeNull()
  })

  it('links task refs in ledger steps to the tracker deep link', async () => {
    mockFetch()
    render(<PlansPage />)
    const list = await screen.findByRole('group', { name: 'Changes' })
    fireEvent.click(within(list).getByRole('button', { name: /003-fix-login/ }))
    await screen.findByText(/step one/)
    const links = screen.getAllByRole('link')
    const hrefs = links.map((l) => l.getAttribute('href'))
    expect(hrefs).toContain('/tracker?task=003')
    expect(hrefs).toContain('/tracker?task=007')
    expect(hrefs).toContain('/tracker?task=012')
  })

  it('opens an archived change named in ?change=', async () => {
    window.history.replaceState({}, '', '/plans?change=2026-09-04-hermes-memory')
    const hermes = {
      ...detail,
      name: '2026-09-04-hermes-memory',
      briefing: 'Hermes Fact Memory',
    }
    vi.spyOn(globalThis, 'fetch').mockImplementation((async (input) => {
      const url = String(input)
      const body = url.includes('/plans/2026-09-04-hermes-memory') ? hermes : { plans: [] }
      return { ok: true, status: 200, text: async () => JSON.stringify(body), json: async () => body }
    }) as typeof fetch)
    render(<PlansPage />)
    expect(await screen.findByText('Hermes Fact Memory')).toBeTruthy()
  })
})
