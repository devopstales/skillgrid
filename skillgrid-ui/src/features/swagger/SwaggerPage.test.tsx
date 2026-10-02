import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { SwaggerPage } from './SwaggerPage'

describe('SwaggerPage', () => {
  const originalDev = import.meta.env.DEV

  beforeEach(() => {
    // Vitest runs with DEV=true; pin expectations for the absolute API origin.
    vi.stubEnv('DEV', true)
  })

  afterEach(() => {
    vi.unstubAllEnvs()
    void originalDev
  })

  it('renders an iframe pointing at the Go Swagger UI (absolute in DEV)', () => {
    render(<SwaggerPage />)
    const iframe = document.querySelector('iframe')
    expect(iframe).toBeTruthy()
    expect(iframe!.getAttribute('src')).toBe('http://127.0.0.1:7438/swagger/')
  })

  it('renders an Open in new tab link with target=_blank', () => {
    render(<SwaggerPage />)
    const link = screen.getByText(/Open in new tab/i).closest('a')
    expect(link).toBeTruthy()
    expect(link!.getAttribute('href')).toBe('http://127.0.0.1:7438/swagger/')
    expect(link!.getAttribute('target')).toBe('_blank')
  })

  it('renders the API Documentation title', () => {
    render(<SwaggerPage />)
    expect(screen.getByRole('heading', { name: /API Documentation/i })).toBeTruthy()
  })
})
