import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { AdrsPage } from './AdrsPage'
import type { DocsContent } from '../docs/types'

// The ADR index as the /docs/content endpoint returns it for
// .skillgrid/artifacts/03-adr-index.md: a GFM table of the in-force set.
const INDEX_BODY = `# ADR Index

Single source for the ADR in-force set.

| # | Title | Status | Supersedes | Date | In force | Record |
|---|-------|--------|------------|------|----------|--------|
| 0001 | PRD scope is the whole Hub Product, not the binary alone | accepted | — | 2026-09-16 | yes | [04-adr-0001](04-adr-0001-prd-scope-whole-hub.md) |
| 0002 | PRD framing is engine-first, not installer-first | accepted | — | 2026-09-16 | yes | [04-adr-0002](04-adr-0002-prd-engine-first-framing.md) |
| 0009 | Vector search: in-SQL sqlite-vec latency corrected | superseded | — | 2026-09-24 | no | [04-adr-0009](04-adr-0009-vector-search-in-sql-latency-viant-deferred.md) |
`

function content(path: string, body: string, title?: string): DocsContent {
  return { path, body, title }
}

describe('AdrsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('parses the ADR index table into rows (id, title, status, date, path)', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(content('.skillgrid/artifacts/03-adr-index.md', INDEX_BODY)), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    ) as unknown as Response

    render(<AdrsPage />)

    // All three ADR titles render from the parsed table.
    expect(await screen.findByText(/PRD scope is the whole Hub Product/)).toBeTruthy()
    expect(screen.getByText(/PRD framing is engine-first/)).toBeTruthy()
    expect(screen.getByText(/Vector search: in-SQL sqlite-vec latency corrected/)).toBeTruthy()

    // The id column renders.
    expect(screen.getByText('0001')).toBeTruthy()
    expect(screen.getByText('0002')).toBeTruthy()
    expect(screen.getByText('0009')).toBeTruthy()

    // The date column renders (two rows share 2026-09-16, so getAllByText).
    expect(screen.getAllByText('2026-09-16').length).toBe(2)
    expect(screen.getByText('2026-09-24')).toBeTruthy()
  })

  it('clicking an ADR row fetches and displays the full record', async () => {
    const adrRecord = content(
      '.skillgrid/artifacts/04-adr-0001-prd-scope-whole-hub.md',
      '# PRD scope is the whole Hub Product\n\n## Decision Outcome\n\nChosen option: "The whole Hub Product".',
    )
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('03-adr-index.md')) {
        return Promise.resolve(
          new Response(JSON.stringify(content('.skillgrid/artifacts/03-adr-index.md', INDEX_BODY)), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }),
        ) as unknown as Promise<Response>
      }
      if (url.includes('04-adr-0001')) {
        return Promise.resolve(
          new Response(JSON.stringify(adrRecord), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }),
        ) as unknown as Promise<Response>
      }
      return Promise.reject(new Error(`unexpected fetch ${url}`)) as unknown as Promise<Response>
    })

    render(<AdrsPage />)
    await screen.findByText(/PRD scope is the whole Hub Product/)

    // Click the first ADR row to expand its full record.
    fireEvent.click(screen.getByText(/PRD scope is the whole Hub Product/))

    // The full record body is now displayed in the detail pane.
    expect(await screen.findByText(/Chosen option: "The whole Hub Product"/)).toBeTruthy()
  })

  it('shows an empty state when the index file does not exist', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: 'not found' }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
      }),
    ) as unknown as Response

    render(<AdrsPage />)

    expect(await screen.findByText(/no ADR index/i)).toBeTruthy()
  })

  it('renders status badges with appropriate color per status', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(content('.skillgrid/artifacts/03-adr-index.md', INDEX_BODY)), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    ) as unknown as Response

    const { container } = render(<AdrsPage />)
    await screen.findByText(/PRD scope is the whole Hub Product/)

    const badges = container.querySelectorAll('[data-testid="adr-status-badge"]')
    expect(badges.length).toBe(3)

    const accepted = container.querySelector('[data-testid="adr-status-badge"][data-status="accepted"]')
    expect(accepted).toBeTruthy()
    expect(accepted?.textContent).toBe('accepted')
    // accepted → green accent styling
    expect(accepted?.className).toMatch(/bg-accent|text-accent/)

    const superseded = container.querySelector('[data-testid="adr-status-badge"][data-status="superseded"]')
    expect(superseded).toBeTruthy()
    expect(superseded?.textContent).toBe('superseded')
    // superseded → neutral / muted styling (not the accent)
    expect(superseded?.className).not.toMatch(/text-accent/)
  })

  it('renders a warn-colored badge for a proposed ADR (different status input)', async () => {
    const proposedBody = `# ADR Index

| # | Title | Status | Supersedes | Date | In force | Record |
|---|-------|--------|------------|------|----------|--------|
| 0017 | A brand-new proposed decision | proposed | — | 2026-09-30 | no | [04-adr-0017](04-adr-0017-new-decision.md) |
`
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(content('.skillgrid/artifacts/03-adr-index.md', proposedBody)), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    ) as unknown as Response

    const { container } = render(<AdrsPage />)
    await screen.findByText(/A brand-new proposed decision/)

    const proposed = container.querySelector('[data-testid="adr-status-badge"][data-status="proposed"]')
    expect(proposed).toBeTruthy()
    expect(proposed?.className).toMatch(/text-warn/)
  })

  it('shows a record error when the full ADR fetch fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('03-adr-index.md')) {
        return Promise.resolve(
          new Response(JSON.stringify(content('.skillgrid/artifacts/03-adr-index.md', INDEX_BODY)), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }),
        ) as unknown as Promise<Response>
      }
      // Match the real /docs/content 404 body exactly (docs_markdown.go:243):
      // a 404 with { error: "doc not found" } — no path/body fields.
      return Promise.resolve(
        new Response(JSON.stringify({ error: 'doc not found' }), {
          status: 404,
          headers: { 'Content-Type': 'application/json' },
        }),
      ) as unknown as Promise<Response>
    })

    render(<AdrsPage />)
    await screen.findByText(/PRD scope is the whole Hub Product/)
    fireEvent.click(screen.getByText(/PRD scope is the whole Hub Product/))

    // The detail pane shows the server error message in a danger banner.
    expect(await screen.findByText(/doc not found/i)).toBeTruthy()
  })
})
