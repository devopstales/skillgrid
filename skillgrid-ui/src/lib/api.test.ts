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

  it('omits project when the caller opts out', async () => {
    await apiGet('/security/trivy', {}, { project: false })
    const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]))
    expect(urls).toContain('/security/trivy')
  })
})
