import { render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PolicyPanel } from './PolicyPanel'
import * as api from '../sessions/api'

afterEach(() => vi.restoreAllMocks())

describe('PolicyPanel', () => {
  it('lists the merged rules with effect and match summary', async () => {
    vi.spyOn(api, 'fetchPolicy').mockResolvedValue({
      project: 'p',
      enabled: true,
      files: ['/repo/.skillgrid/policy.yaml'],
      rules: [
        {
          name: 'no-secret-writes',
          match: { action: ['file_write'], path: ['secrets/**', '**/*.pem'] },
          effect: 'block',
          message: 'secrets are read-only',
          source: '/repo/.skillgrid/policy.yaml',
        },
        { name: 'prefer-mem', match: { tool: ['WebSearch'] }, effect: 'guide', message: 'try mem_search', source: 'x' },
      ],
    })
    render(<PolicyPanel />)
    const table = await screen.findByTestId('policy-rules')
    expect(screen.getByTestId('policy-state').textContent).toBe('enforced')
    expect(within(table).getByText('no-secret-writes')).toBeTruthy()
    expect(within(table).getByText('block')).toBeTruthy()
    expect(within(table).getByText('action: file_write · path: secrets/** | **/*.pem')).toBeTruthy()
    expect(within(table).getByText('guide')).toBeTruthy()
  })

  it('points at policy init when there is no file', async () => {
    vi.spyOn(api, 'fetchPolicy').mockResolvedValue({
      project: 'p',
      enabled: false,
      files: [],
      rules: [],
      repoFile: '/repo/.skillgrid/policy.yaml',
    })
    render(<PolicyPanel />)
    expect(await screen.findByText('skillgrid policy init')).toBeTruthy()
    expect(screen.getByTestId('policy-state').textContent).toBe('disabled')
  })

  it('shows a broken file as fail-open', async () => {
    vi.spyOn(api, 'fetchPolicy').mockResolvedValue({
      project: 'p',
      enabled: false,
      files: [],
      rules: [],
      error: 'rule 1: unknown effect "deny"',
    })
    render(<PolicyPanel />)
    expect((await screen.findByTestId('policy-error')).textContent).toContain('fail open')
  })
})
