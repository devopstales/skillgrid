import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TeamsPage } from './TeamsPage'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('TeamsPage', () => {
  it('shows ledger rows from /sdd/runs', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            source: '.skillgrid/sdd',
            count: 1,
            in_flight: 1,
            done: 0,
            runs: [
              {
                name: '2026-10-02-memory',
                heading: 'Parallel ledger — memory wave',
                updated: '2026-10-02T09:00:00Z',
                in_flight: 1,
                done: 0,
                members: [
                  {
                    agent: 'decay-config',
                    task: 'TASK-030.05 TICKET-03',
                    status: 'dispatched',
                    owns: 'config/load.go',
                  },
                ],
              },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    )

    render(<TeamsPage />)
    const change = await screen.findByRole('link', { name: '2026-10-02-memory' })
    expect(change.getAttribute('href')).toBe('/plans?change=2026-10-02-memory')
    expect(screen.getByText('decay-config')).toBeTruthy()
    const link = screen.getByRole('link', { name: 'TASK-030.05' })
    expect(link.getAttribute('href')).toBe('/tracker?task=TASK-030.05')
    expect(screen.getByText(/TICKET-03/)).toBeTruthy()
    expect(screen.getByText('dispatched')).toBeTruthy()
  })

  it('shows the harness, live status, and session link on each ledger row', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string) => {
        const u = String(url)
        const body = u.includes('/project/current')
          ? { project: 'skillgrid' }
          : u.includes('/mnemonic/sessions')
          ? { sessions: [] }
          : {
              count: 1,
              runs: [
                {
                  name: '2026-09-24-mnemonic-memory-improvements',
                  members: [
                    {
                      agent: 'decay-config',
                      task: 'TASK-030.05 TICKET-03',
                      status: 'dispatched',
                      session: {
                        id: 'sess-decay',
                        project: 'aiskillgrid',
                        agent: 'cursor',
                        status: 'active',
                        title: 'TICKET-03 mnemonic.decay config',
                        live: 'working',
                      },
                    },
                    { agent: 'query-cache', task: 'TASK-030.02 TICKET-04', status: 'dispatched' },
                  ],
                },
              ],
            }
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }),
    )

    render(<TeamsPage />)
    const harness = await screen.findByRole('link', { name: 'cursor' })
    expect(harness.getAttribute('href')).toBe('/mnemonic/sessions?id=sess-decay&store=aiskillgrid')
    const row = harness.closest('tr') as HTMLElement
    expect(row.textContent).toContain('decay-config')
    expect(row.textContent).toContain('working')
    const unmatched = screen.getByText('query-cache').closest('tr') as HTMLElement
    expect(unmatched.querySelector('a[href^="/mnemonic/sessions"]')).toBeNull()
  })

  it('links a live harness to its session', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string) => {
        const u = String(url)
        const body = u.includes('/project/current')
          ? { project: 'skillgrid' }
          : u.includes('/mnemonic/sessions')
          ? {
              sessions: [
                {
                  id: 'sess-cursor',
                  title: 'Decay work',
                  status: 'active',
                  agent: 'cursor',
                  started_at: '2026-10-02T10:00:00Z',
                  memory_count: 1,
                  has_summary: false,
                },
              ],
            }
          : { count: 0, runs: [] }
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }),
    )

    render(<TeamsPage />)
    const link = await screen.findByRole('link', { name: 'cursor' })
    expect(link.getAttribute('href')).toBe('/mnemonic/sessions?id=sess-cursor')
    expect(screen.getByText('idle')).toBeTruthy()
  })

  it('marks a recent harness working and an ended one finished', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string) => {
        const u = String(url)
        const body = u.includes('/project/current')
          ? { project: 'skillgrid' }
          : u.includes('/mnemonic/sessions')
          ? {
              sessions: [
                {
                  id: 'sess-work',
                  title: 'In progress',
                  status: 'active',
                  agent: 'opencode',
                  last_active: new Date().toISOString(),
                },
                {
                  id: 'sess-done',
                  title: 'Closed',
                  status: 'ended',
                  agent: 'cursor',
                  last_active: '2026-10-01T10:00:00Z',
                },
              ],
            }
          : { count: 0, runs: [] }
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }),
    )

    render(<TeamsPage />)
    expect(await screen.findByText('working')).toBeTruthy()
    expect(screen.getByText('finished')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'opencode' }).getAttribute('href')).toBe(
      '/mnemonic/sessions?id=sess-work',
    )
  })

  it('shows an empty state when no ledgers exist', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        new Response(JSON.stringify({ source: '.skillgrid/sdd', count: 0, runs: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )

    render(<TeamsPage />)
    expect(await screen.findByText(/No execution ledgers/)).toBeTruthy()
  })
})
