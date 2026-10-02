import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AgentStatsPage } from './AgentStatsPage'
import * as api from '../sessions/api'

const stats: api.EventStats = {
  project: 'p',
  since: '',
  total: 5,
  sessions: 2,
  mcp: 1,
  errors: 1,
  sensitive: 0,
  blocked: 0,
  byAction: { command_exec: 2, file_read: 1 },
  byAgent: [
    { agent: 'cursor', sessions: 1, events: 3, reads: 1, writes: 0, commands: 1, mcp: 1, errors: 0, blocked: 0, inputTokens: 1200, outputTokens: 300, costUsd: 0.0042 },
    { agent: 'opencode', sessions: 1, events: 2, reads: 0, writes: 1, commands: 1, mcp: 0, errors: 1, blocked: 0, inputTokens: 0, outputTokens: 0, costUsd: null },
  ],
  byModel: [
    { model: 'claude-sonnet-4-5', sessions: 1, inputTokens: 1200, outputTokens: 300, cacheTokens: 0, costUsd: 0.0042 },
    { model: 'cursor-private', sessions: 1, inputTokens: 0, outputTokens: 0, cacheTokens: 0, costUsd: null },
  ],
  topFiles: [{ name: 'src/auth/login.go', count: 1 }],
  topCommands: [{ name: 'go test ./...', count: 1 }],
  topTools: [{ name: 'mem_search', count: 1, mcp: true }],
}

beforeEach(() => {
  vi.spyOn(api, 'fetchEventStats').mockResolvedValue(stats)
})
afterEach(() => vi.restoreAllMocks())

describe('AgentStatsPage', () => {
  it('renders per-agent rows, top lists, and cost', async () => {
    render(<AgentStatsPage />)
    const table = await screen.findByTestId('agent-table')
    expect(table.textContent).toContain('cursor')
    expect(table.textContent).toContain('opencode')
    expect(table.textContent).toContain('$0.0042')
    expect(table.textContent).toContain('n/a')
    const models = screen.getByTestId('model-table')
    expect(models.textContent).toContain('claude-sonnet-4-5')
    expect(models.textContent).toContain('1.2k')
    expect(models.textContent).toContain('n/a')
    expect(screen.getByText('src/auth/login.go')).toBeTruthy()
    expect(screen.getByText('go test ./...')).toBeTruthy()
    expect(screen.getByText('mem_search').parentElement?.textContent).toContain('MCP')
  })

  it('refetches when the window changes', async () => {
    render(<AgentStatsPage />)
    await screen.findByTestId('agent-table')
    fireEvent.change(screen.getByLabelText('window'), { target: { value: '7d' } })
    await waitFor(() => expect(api.fetchEventStats).toHaveBeenCalledWith({ since: '7d', agent: '' }))
  })
})
