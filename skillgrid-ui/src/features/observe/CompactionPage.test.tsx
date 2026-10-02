import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CompactionPage } from './CompactionPage'
import { apiGet } from '../../lib/api'

vi.mock('../../lib/api', () => ({
  apiGet: vi.fn(),
}))

beforeEach(() => {
  vi.mocked(apiGet).mockImplementation(async (path: string) => {
    if (path === '/context') return { project: 'skillgrid', session_id: 'sess-9' }
    throw new Error('compaction down')
  })
})

describe('CompactionPage', () => {
  it('keeps context when compaction fails', async () => {
    render(<CompactionPage />)
    expect(await screen.findByText('compaction down')).toBeTruthy()
    expect(screen.getByText('skillgrid')).toBeTruthy()
  })
})
