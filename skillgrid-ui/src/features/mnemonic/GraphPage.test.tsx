import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { GraphPage } from './GraphPage'
import { fetchGraph } from './graph/api'
import { apiGet } from '../../lib/api'

vi.mock('./graph/api', () => ({
  fetchGraph: vi.fn(),
}))

vi.mock('../../lib/api', () => ({
  apiGet: vi.fn(),
}))

const graph = {
  nodes: [
    {
      id: 1,
      uid: 'fn:main',
      label: 'main',
      type: 'function',
      language: 'go',
      path: 'main.go',
      degree: 0,
      community: 0,
    },
  ],
  edges: [],
  truncated: false,
  degraded: false,
  project: 'skillgrid',
}

beforeEach(() => {
  vi.mocked(fetchGraph).mockResolvedValue(graph)
  vi.mocked(apiGet).mockResolvedValue(null)
})

describe('GraphPage', () => {
  it('requests at most 500 nodes and renders the inspector', async () => {
    render(<GraphPage />)
    await waitFor(() => expect(fetchGraph).toHaveBeenCalledWith({ limit: 500 }))
    expect(screen.getByText('Node Inspector')).toBeTruthy()
    await waitFor(() => expect(document.querySelector('svg')).toBeTruthy())
  })
})
