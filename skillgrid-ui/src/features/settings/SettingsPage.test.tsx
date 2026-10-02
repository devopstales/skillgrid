import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { SettingsPage } from './SettingsPage'

const TOKEN_KEY = 'skillgrid.httpToken'

describe('SettingsPage', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the HTTP Token label and description', () => {
    render(<SettingsPage />)
    expect(screen.getByText('HTTP Token')).toBeTruthy()
    expect(screen.getByText(/Used for API authentication to the Skillgrid backend/)).toBeTruthy()
  })

  it('populates the input with the current localStorage token on load', async () => {
    localStorage.setItem(TOKEN_KEY, 'tok-abc-123')
    render(<SettingsPage />)
    const input = (await screen.findByLabelText(/HTTP Token/i)) as HTMLInputElement
    expect(input.value).toBe('tok-abc-123')
  })

  it('saves a new token value to localStorage and shows a confirmation', async () => {
    render(<SettingsPage />)
    const input = (await screen.findByLabelText(/HTTP Token/i)) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'tok-new-456' } })
    fireEvent.click(screen.getByRole('button', { name: /^Save$/i }))

    expect(localStorage.getItem(TOKEN_KEY)).toBe('tok-new-456')
    expect(screen.getByText(/token saved/i)).toBeTruthy()
  })

  it('shows no confirmation message before saving', async () => {
    render(<SettingsPage />)
    await screen.findByLabelText(/HTTP Token/i)
    expect(screen.queryByText(/token saved/i)).toBeNull()
  })

  it('clears the confirmation after it expires', async () => {
    render(<SettingsPage />)
    const input = screen.getByLabelText(/HTTP Token/i) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'tok-temp' } })
    fireEvent.click(screen.getByRole('button', { name: /^Save$/i }))
    expect(screen.getByText(/token saved/i)).toBeTruthy()

    await new Promise((r) => setTimeout(r, 2500))
    expect(screen.queryByText(/token saved/i)).toBeNull()
  })

  it('renders the About section', () => {
    render(<SettingsPage />)
    expect(screen.getByText('About')).toBeTruthy()
    expect(screen.getByText('Skillgrid Dashboard')).toBeTruthy()
    expect(screen.getByText('Mnemonic Admin Console (spike 001)')).toBeTruthy()
    expect(screen.getByText('Powered by Skillgrid')).toBeTruthy()
  })
})
