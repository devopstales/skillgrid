import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchSessions } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
  window.history.replaceState({}, '', '/')
})

describe('sessions api project', () => {
  it('reads sessions from the ?store= project named in the link', async () => {
    window.history.replaceState({}, '', '/mnemonic/sessions?id=sess-decay&store=aiskillgrid')
    const fetchMock = vi.fn(async (url: string) => {
      const body = String(url).startsWith('/project/current')
        ? { project: 'skillgrid' }
        : String(url).startsWith('/activity/stats')
          ? { total: 5 }
          : { project: 'skillgrid', projects: ['skillgrid'], sessions: [] }
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    await fetchSessions()
    const urls = fetchMock.mock.calls.map((c) => String((c as unknown[])[0]))
    expect(urls.some((u) => u.includes('/mnemonic/sessions') && u.includes('project=aiskillgrid'))).toBe(true)
  })
})
