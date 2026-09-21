import { fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { SessionsPage } from './SessionsPage'
import * as api from '../sessions/api'

// The tab nav buttons render their raw value and uppercase it via CSS
// `capitalize`, so the DOM text is the lowercase value ('audit', 'activity').
// Address them by role + exact lowercase name.
const tabButton = (name: string) => screen.getByRole('button', { name })

// The Sessions page is the single "what happened" home. These tests pin the
// consolidation behavior: the session list renders, the session-scoped activity
// endpoint is used when a session is selected, and the Audit tab is a distinct
// (secondary) tab.
const session = (id: string, title: string) => ({
  id,
  title,
  started_at: '2026-09-10T09:00:00Z',
  ended_at: undefined,
  status: 'active',
  memory_count: 2,
  has_summary: true,
})

beforeEach(() => {
  vi.spyOn(api, 'fetchSessions').mockResolvedValue({
    project: 'p',
    sessions: [session('sess-a', 'Session A')],
  })
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
  // The single SSE endpoint is stubbed to a no-op (returns a cleanup fn).
  vi.spyOn(api, 'openActivityStream').mockReturnValue(() => {})
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SessionsPage (unified activity home)', () => {
  it('renders the session list', async () => {
    render(<SessionsPage />)
    expect(await screen.findByText('Session A')).toBeTruthy()
  })

  it('uses the session-scoped activity endpoint when a session is selected', async () => {
    render(<SessionsPage />)
    const listBtn = await screen.findByRole('button', { name: /Session A/ })
    // Selecting the session scopes the Activity tab to that session.
    fireEvent.click(listBtn)
    await screen.findByText('A-chose-sqlite') // proves the scoped feed rendered
    expect(api.fetchSessionActivity).toHaveBeenCalledWith('sess-a', 200)
  })

  it('shows a distinct secondary Audit tab (not the default view)', async () => {
    render(<SessionsPage />)
    await screen.findByRole('button', { name: /Session A/ })
    // Audit is a tab button, present but not active by default.
    expect(tabButton('audit')).toBeTruthy()
    // The audit table is not rendered until the tab is opened.
    expect(screen.queryByText('deadbeefcafe')).toBeFalsy()
  })
})
