import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { TelemetryPage } from './TelemetryPage'
import { apiGet } from '../../lib/api'

vi.mock('../../lib/api', () => ({
  apiGet: vi.fn(),
}))

beforeEach(() => {
  vi.mocked(apiGet).mockImplementation(async (path: string) => {
    if (path === '/mnemonic/sessions') return { sessions: [{ id: 'sess-1', status: 'active' }] }
    throw new Error('events down')
  })
})

describe('TelemetryPage', () => {
  it('keeps sessions when the event stream fails', async () => {
    render(<TelemetryPage />)
    expect(await screen.findByText('events down')).toBeTruthy()
    expect(screen.getByText(/sess-1/)).toBeTruthy()
  })
})
