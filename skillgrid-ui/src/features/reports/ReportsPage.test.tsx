import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ReportsPage } from './ReportsPage'
import * as reportsApi from './api'
import type { ReportJSON } from './api'

const fixture: ReportJSON = {
  static: [
    {
      tool: 'trivy',
      status: 'ok',
      findings: [
        {
          severity: 'HIGH',
          title: 'test vuln',
          cve_id: 'CVE-2024-0001',
          rule_id: 'CVE-2024-0001',
          package: 'example.com/lib',
          version: '1.0.0',
          fixed_version: '1.0.1',
          file: 'go.mod',
          line: 0,
        },
      ],
    },
  ],
  dynamic: [],
  coverage: [],
  generated_at: '2026-10-17T12:00:00Z',
}

afterEach(() => vi.restoreAllMocks())

describe('ReportsPage', () => {
  it('renders the CVE panel with the trivy findings', async () => {
    vi.spyOn(reportsApi, 'fetchReport').mockResolvedValue(fixture)
    render(<ReportsPage />)
    expect(await screen.findByText('Quality Report')).toBeTruthy()
    expect(screen.getByText('CVE (Trivy)')).toBeTruthy()
    expect(screen.getByText('CVE-2024-0001')).toBeTruthy()
    expect(screen.getByTestId('cve-package').textContent).toBe('example.com/lib')
    expect(screen.getByText('fixed in 1.0.1')).toBeTruthy()
  })

  it('shows an empty CVE state when trivy found nothing', async () => {
    vi.spyOn(reportsApi, 'fetchReport').mockResolvedValue({ ...fixture, static: [{ tool: 'trivy', status: 'ok', findings: [] }] })
    render(<ReportsPage />)
    expect(await screen.findByText('No vulnerabilities found')).toBeTruthy()
  })

  it('renders an error state when the endpoint fails', async () => {
    vi.spyOn(reportsApi, 'fetchReport').mockRejectedValue(new Error('boom'))
    render(<ReportsPage />)
    expect((await screen.findByText('boom')).textContent).toContain('boom')
  })

  it('shows the failed-source notice when trivy errors', async () => {
    vi.spyOn(reportsApi, 'fetchReport').mockResolvedValue({ ...fixture, static: [{ tool: 'trivy', status: 'error', findings: [] }] })
    render(<ReportsPage />)
    expect(await screen.findByTestId('trivy-source-error')).toBeTruthy()
  })
})
