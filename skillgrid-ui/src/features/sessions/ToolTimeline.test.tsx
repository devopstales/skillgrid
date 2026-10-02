import { render, screen, waitFor, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ToolTimeline } from './ToolTimeline'
import * as api from './api'

const registerTool = vi.fn()
const registerObservation = vi.fn()
let onObservationLive: ((o: api.ObservationRow) => void) | null = null

const tool = (over: Partial<api.ToolEvent> = {}): api.ToolEvent => ({
  id: 1,
  ts: '2026-09-10T09:00:00Z',
  sessionId: 'sess-a',
  sequence: 1,
  agent: 'cursor',
  action: 'file_read',
  tool: 'Read',
  path: 'early.go',
  command: '',
  preview: '',
  result: 'success',
  sensitive: false,
  mcp: false,
  ...over,
})

const observation = (over: Partial<api.ObservationRow> = {}): api.ObservationRow => ({
  id: 50,
  type: 'decision',
  title: 'chose-sqlite',
  created_at: '2026-09-10T09:05:00Z',
  ...over,
})

beforeEach(() => {
  onObservationLive = null
  registerTool.mockImplementation(() => undefined)
  registerObservation.mockImplementation((h) => {
    onObservationLive = h
    return undefined
  })
  vi.spyOn(api, 'fetchSessionEvents').mockResolvedValue({
    events: [
      tool({ id: 2, ts: '2026-09-10T09:10:00Z', path: 'late.go' }),
      tool({ id: 1, ts: '2026-09-10T09:00:00Z', path: 'early.go' }),
    ],
    observations: [observation()],
  })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ToolTimeline observations', () => {
  it('interleaves an observation row between tool rows by timestamp', async () => {
    render(
      <ToolTimeline
        sessionId="sess-a"
        agents={['cursor']}
        live
        registerTool={registerTool}
        registerObservation={registerObservation}
      />,
    )
    const log = await screen.findByRole('log', { name: 'Tool calls' })
    await waitFor(() => expect(within(log).getByText('late.go')).toBeTruthy())
    const rows = within(log).getAllByRole('listitem')
    expect(rows).toHaveLength(3)
    expect(rows[0].textContent).toContain('late.go')
    expect(rows[1].textContent).toContain('chose-sqlite')
    expect(rows[1].textContent).toContain('decision')
    expect(rows[2].textContent).toContain('early.go')
  })

  it('links an observation row to the memory detail route', async () => {
    render(
      <ToolTimeline
        sessionId="sess-a"
        agents={['cursor']}
        live
        registerTool={registerTool}
        registerObservation={registerObservation}
      />,
    )
    const link = await screen.findByRole('link', { name: /chose-sqlite/i })
    expect(link.getAttribute('href')).toBe('/mnemonic/memories?id=50')
  })

  it('merges a live observation without duplicates', async () => {
    render(
      <ToolTimeline
        sessionId="sess-a"
        agents={['cursor']}
        live
        registerTool={registerTool}
        registerObservation={registerObservation}
      />,
    )
    await screen.findByText('chose-sqlite')
    onObservationLive?.(observation({ id: 51, title: 'live-obs', created_at: '2026-09-10T09:11:00Z' }))
    await waitFor(() => expect(screen.getByText('live-obs')).toBeTruthy())
    onObservationLive?.(observation({ id: 51, title: 'live-obs', created_at: '2026-09-10T09:11:00Z' }))
    expect(screen.getAllByText('live-obs')).toHaveLength(1)
  })
})
