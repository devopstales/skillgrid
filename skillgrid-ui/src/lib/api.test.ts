import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiGet, clearProjectCache } from './api'

beforeEach(() => {
  clearProjectCache()
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      if (String(url) === '/project/current') {
        return { ok: true, json: async () => ({ project: 'skillgrid' }) }
      }
      return { ok: true, statusText: 'OK', json: async () => ({ ok: true }) }
    }),
  )
})

describe('apiGet', () => {
  it('appends the resolved project on a relative URL', async () => {
    await apiGet('/prototypes')
    const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]))
    expect(urls).toContain('/prototypes?project=skillgrid')
  })

  it('does not cache a failed project resolve', async () => {
    clearProjectCache()
    vi.mocked(fetch).mockImplementation(async (url: string) => {
      if (String(url) === '/project/current') {
        return { ok: false, status: 503, statusText: 'unavailable', json: async () => ({}) }
      }
      return { ok: true, statusText: 'OK', json: async () => ({ ok: true }) }
    })
    await expect(apiGet('/prototypes')).rejects.toThrow('unavailable')
    await expect(apiGet('/prototypes')).rejects.toThrow('unavailable')
    const projectCalls = vi.mocked(fetch).mock.calls.filter((c) => String(c[0]) === '/project/current')
    expect(projectCalls).toHaveLength(2)
  })

  it('omits project when the caller opts out', async () => {
    await apiGet('/security/trivy', {}, { project: false })
    const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]))
    expect(urls).toContain('/security/trivy')
  })
})
