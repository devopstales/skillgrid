import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SwaggerPage } from './SwaggerPage'

describe('SwaggerPage', () => {
  it('renders an iframe pointing at /swagger/', () => {
    render(<SwaggerPage />)
    const iframe = document.querySelector('iframe')
    expect(iframe).toBeTruthy()
    expect(iframe!.getAttribute('src')).toBe('/swagger/')
  })

  it('renders an Open in new tab link with target=_blank', () => {
    render(<SwaggerPage />)
    const link = screen.getByText(/Open in new tab/i).closest('a')
    expect(link).toBeTruthy()
    expect(link!.getAttribute('href')).toBe('/swagger/')
    expect(link!.getAttribute('target')).toBe('_blank')
  })

  it('renders the API Documentation title', () => {
    render(<SwaggerPage />)
    expect(screen.getByRole('heading', { name: /API Documentation/i })).toBeTruthy()
  })
})
