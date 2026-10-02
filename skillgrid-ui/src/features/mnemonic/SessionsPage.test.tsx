import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { SessionsPage } from './SessionsPage'
import * as api from '../sessions/api'

// The tab nav buttons render their lowercase label ('tool calls', 'memory',
// 'audit'). Address them by role + exact name.
const tabButton = (name: string) => screen.getByRole('button', { name })

// The Sessions page is the single "what happened" home. These tests pin: the
// list renders with the owning harness, selecting a session loads its tool
// timeline, live `tool` frames land without a refresh, and memory/audit are
// secondary tabs.
const session = (id: string, title: string, agent?: string) => ({
  id,
  title,
  started_at: '2026-09-10T09:00:00Z',
  ended_at: undefined,
  status: 'active',
  memory_count: 2,
  has_summary: true,
  agent,
  tool_calls: 3,
  last_tool: 'Read',
})

const toolEvent = (over: Partial<api.ToolEvent> = {}): api.ToolEvent => ({
  id: 1,
  ts: '2026-09-10T09:05:00Z',
  sessionId: 'sess-a',
  sequence: 2,
  agent: 'cursor',
  action: 'file_read',
  tool: 'Read',
  path: 'src/main.go',
  command: '',
  preview: '',
  result: 'success',
  sensitive: false,
  mcp: false,
  ...over,
})

let streamCallbacks: api.StreamCallbacks | null = null

beforeEach(() => {
  streamCallbacks = null
  vi.spyOn(api, 'fetchSessions').mockResolvedValue({
    project: 'p',
    sessions: [session('sess-a', 'Session A', 'cursor')],
  })
  vi.spyOn(api, 'fetchSessionEvents').mockResolvedValue({
    events: [toolEvent()],
  })
  vi.spyOn(api, 'fetchEvents').mockResolvedValue({ events: [] })
  vi.spyOn(api, 'fetchSessionActivity').mockResolvedValue({
    project: 'p',
    events: [{
      id: 1,
      ts: '2026-09-10T09:05:00Z',
      type: 'decision',
      source: 'agent',
      actor: null,
      severity: 'medium',
      summary: 'A-chose-sqlite',
      topicKey: null,
      sessionId: 'sess-a',
      relatedIds: [],
    }],
    limit: 200,
  })
  vi.spyOn(api, 'fetchActivityEvents').mockResolvedValue({ project: 'p', events: [], limit: 200 })
  vi.spyOn(api, 'fetchActivityStats').mockResolvedValue({
    project: 'p',
    total: 1,
    byType: { decision: 1 },
    activeSessions: 1,
  })
  vi.spyOn(api, 'fetchSessionSummary').mockResolvedValue({
    id: 'sess-a',
    summary: '## Goal A\nwe chose sqlite',
    status: 'active',
    ended_at: '',
  })
  vi.spyOn(api, 'fetchAudit').mockResolvedValue({
    project: 'p',
    entries: [{ seq: 1, observation_id: 1, revision: 1, created_at: '2026-09-10T09:05:00Z', hash: 'deadbeefcafe', prev_hash: '0000000000' }],
    chain_valid: true,
  })
  vi.spyOn(api, 'openActivityStream').mockImplementation((cb) => {
    streamCallbacks = cb
    return () => {}
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  window.history.replaceState({}, '', '/')
})

describe('SessionsPage (live session home)', () => {
  it('renders the session list with the agent badge', async () => {
    render(<SessionsPage />)
    const row = await screen.findByRole('button', { name: /Session A/ })
    expect(row.textContent).toContain('cursor')
  })

  it('loads the session tool timeline when a session is clicked', async () => {
    render(<SessionsPage />)
    fireEvent.click(await screen.findByRole('button', { name: /Session A/ }))
    await waitFor(() => expect(api.fetchSessionEvents).toHaveBeenCalledWith('sess-a', {}))
    expect(await screen.findByText('src/main.go')).toBeTruthy()
    expect(window.location.search).toContain('id=sess-a')
  })

  it('opens the session named in ?id= and the legacy ?session=', async () => {
    for (const q of ['?id=sess-a', '?session=sess-a']) {
      window.history.replaceState({}, '', `/mnemonic/sessions${q}`)
      const { unmount } = render(<SessionsPage />)
      const row = await screen.findByRole('button', { name: /Session A/ })
      expect(row.className).toContain('border-accent')
      await waitFor(() => expect(api.fetchSessionEvents).toHaveBeenCalledWith('sess-a', {}))
      unmount()
    }
  })

  it('appends a live tool event without a refresh', async () => {
    window.history.replaceState({}, '', '/mnemonic/sessions?id=sess-a')
    render(<SessionsPage />)
    await screen.findByText('src/main.go')
    streamCallbacks?.onTool?.(
      toolEvent({ id: 9, tool: 'mcp__mnemonic__mem_search', action: 'tool_use', path: '', command: 'mem_search q', mcp: true }),
    )
    const log = screen.getByRole('log', { name: 'Tool calls' })
    await waitFor(() => expect(within(log).getByText('mcp__mnemonic__mem_search')).toBeTruthy())
    expect(within(log).getByText('MCP')).toBeTruthy()
    // The list row picks up the new last tool live, too.
    const row = screen.getByRole('button', { name: /Session A/ })
    expect(within(row).getByText('mcp__mnemonic__mem_search')).toBeTruthy()
  })

  it('shows a live policy block in red with its rule and counts it', async () => {
    window.history.replaceState({}, '', '/mnemonic/sessions?id=sess-a')
    render(<SessionsPage />)
    await screen.findByText('src/main.go')
    streamCallbacks?.onTool?.(
      toolEvent({
        id: 11,
        tool: 'Write',
        action: 'file_write',
        path: 'secrets/db.env',
        result: 'blocked',
        preview: 'no-secret-writes: secrets are read-only',
      }),
    )
    const log = screen.getByRole('log', { name: 'Tool calls' })
    const note = await within(log).findByTestId('policy-note')
    expect(note.textContent).toContain('no-secret-writes: secrets are read-only')
    expect(note.className).toContain('text-danger')
    expect(within(log).getByText('blocked').className).toContain('text-danger')
    await waitFor(() => expect(screen.getByTestId('policy-count').textContent).toBe('1 policy'))
    expect(screen.getByText('1 blocked')).toBeTruthy()
    expect(screen.queryByText(/^4 calls/)).toBeNull()
  })

  it('reloads the list when a tool event names an unknown session', async () => {
    render(<SessionsPage />)
    await screen.findByRole('button', { name: /Session A/ })
    const calls = vi.mocked(api.fetchSessions).mock.calls.length
    streamCallbacks?.onTool?.(toolEvent({ id: 10, sessionId: 'sess-new', newSession: true }))
    await waitFor(() => expect(vi.mocked(api.fetchSessions).mock.calls.length).toBeGreaterThan(calls))
  })

  it('keeps memory activity on a secondary tab', async () => {
    window.history.replaceState({}, '', '/mnemonic/sessions?id=sess-a')
    render(<SessionsPage />)
    await screen.findByRole('button', { name: /Session A/ })
    fireEvent.click(tabButton('memory'))
    await screen.findByText('A-chose-sqlite')
    expect(api.fetchSessionActivity).toHaveBeenCalledWith('sess-a', 200)
  })

  it('shows a distinct secondary Audit tab (not the default view)', async () => {
    render(<SessionsPage />)
    await screen.findByRole('button', { name: /Session A/ })
    expect(tabButton('audit')).toBeTruthy()
    expect(screen.queryByText('deadbeefcafe')).toBeFalsy()
  })
})
