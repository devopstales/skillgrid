import { afterEach, describe, expect, it, vi } from 'vitest'

describe('apiOrigin', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('is empty outside DEV so production calls stay same-origin', async () => {
    vi.stubEnv('DEV', false)
    vi.resetModules()
    const { apiOrigin } = await import('./apiBase')
    expect(apiOrigin()).toBe('')
  })

  it('points at the Go API while Vite is in DEV', async () => {
    vi.stubEnv('DEV', true)
    vi.resetModules()
    const { apiOrigin } = await import('./apiBase')
    expect(apiOrigin()).toBe('http://127.0.0.1:7438')
  })
})
