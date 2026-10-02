import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { WebCachePage } from './WebCachePage'
import { apiGet } from '../../lib/api'

vi.mock('../../lib/api', () => ({
  apiGet: vi.fn(),
}))

beforeEach(() => {
  vi.mocked(apiGet).mockImplementation(async (path: string) => {
    if (path === '/web/status') return { total_entries: 3, by_source: { exa: 3 } }
    throw new Error('search down')
  })
})

describe('WebCachePage', () => {
  it('keeps the status cards when search fails', async () => {
    render(<WebCachePage />)
    expect(await screen.findByText('search down')).toBeTruthy()
    expect(screen.getByText('Total Entries')).toBeTruthy()
    expect(screen.queryByText('No entries match filter')).toBeNull()
    expect(screen.queryByText('Cache is empty — no web research cached yet')).toBeNull()
  })
})
