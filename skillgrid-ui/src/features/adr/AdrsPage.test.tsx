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

// The in-force set as ASSUMPTIONS.md carries it after ADR-0019: the Record
// column is a backticked repo-relative path, not a markdown link, and the
// table sits under "### In-force set" among other sections.
const ASSUMPTIONS_BODY = `# ASSUMPTIONS

## VERIFIED

- some fact

## LOCKED

### In-force set

| # | Title | Status | Supersedes | Date | In force | Record |
|---|-------|--------|------------|------|----------|--------|
| 0001 | PRD scope is the whole Hub Product, not the binary alone | accepted | — | 2026-09-16 | yes | \`.skillgrid/artifacts/04-adr-0001-prd-scope-whole-hub.md\` |
| 0014 | (removed 2026-09-30) | superseded | — | 2026-09-29 | no | \`.skillgrid/artifacts/04-adr-0014-removed.md\` |
| 0019 | Locked decisions are ADR files; ASSUMPTIONS.md holds the path | accepted | — | 2026-10-02 | yes | \`.skillgrid/artifacts/04-adr-0019-decisions-are-files.md\` |

### Locked constraints

- Go 1.22+ minimum to build.
`

// The post-ADR-0019 index file: a pointer with no table.
const STUB_INDEX_BODY = `# ADR Index

The in-force set is the table in \`.skillgrid/ASSUMPTIONS.md\` § \`### In-force set\`. This file is not a second table.
`

function content(path: string, body: string, title?: string): DocsContent {
  return { path, body, title }
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('AdrsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reads the in-force set from ASSUMPTIONS.md with backticked Record paths', async () => {
    const fetched: string[] = []
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.toString()
      fetched.push(url)
      if (url.includes('ASSUMPTIONS.md')) {
        return Promise.resolve(
          jsonResponse(content('.skillgrid/ASSUMPTIONS.md', ASSUMPTIONS_BODY)),
        ) as unknown as Promise<Response>
      }
      if (url.includes('04-adr-0019')) {
        return Promise.resolve(
          jsonResponse(
            content(
              '.skillgrid/artifacts/04-adr-0019-decisions-are-files.md',
              '# Locked decisions are ADR files\n\n## Decision\n\nEach decision is one file.',
            ),
          ),
        ) as unknown as Promise<Response>
      }
      return Promise.reject(new Error(`unexpected fetch ${url}`)) as unknown as Promise<Response>
    })

    render(<AdrsPage />)

    expect(await screen.findByText(/PRD scope is the whole Hub Product/)).toBeTruthy()
    expect(screen.getByText(/\(removed 2026-09-30\)/)).toBeTruthy()
    expect(screen.getByText(/Locked decisions are ADR files/)).toBeTruthy()
    expect(screen.getByRole('tab', { name: /ADRs/ }).textContent).toContain('3')

    // The backticked Record path resolves to the full repo-relative path.
    fireEvent.click(screen.getByText(/Locked decisions are ADR files/))
    expect(await screen.findByText(/Each decision is one file\./)).toBeTruthy()
    expect(
      fetched.some((u) =>
        u.includes(encodeURIComponent('.skillgrid/artifacts/04-adr-0019-decisions-are-files.md')),
      ),
    ).toBe(true)
  })

  it('falls back to 03-adr-index.md when ASSUMPTIONS.md has no in-force table', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('ASSUMPTIONS.md')) {
        return Promise.resolve(
          jsonResponse(content('.skillgrid/ASSUMPTIONS.md', '# ASSUMPTIONS\n\n## VERIFIED\n\n- fact\n')),
        ) as unknown as Promise<Response>
      }
      if (url.includes('03-adr-index.md')) {
        return Promise.resolve(
          jsonResponse(content('.skillgrid/artifacts/03-adr-index.md', INDEX_BODY)),
        ) as unknown as Promise<Response>
      }
      return Promise.reject(new Error(`unexpected fetch ${url}`)) as unknown as Promise<Response>
    })

    render(<AdrsPage />)

    expect(await screen.findByText(/PRD scope is the whole Hub Product/)).toBeTruthy()
    expect(screen.getByRole('tab', { name: /ADRs/ }).textContent).toContain('3')
  })

  it('lists the Open Questions section from ASSUMPTIONS.md beside the records', async () => {
    const body = `${ASSUMPTIONS_BODY}
## Open Questions

Questions that are not yet decided and not yet prototyped.

1. **v1.1 dashboard prioritization** — Memories-first vs Graph-first.
2. **README alignment** — the README is still installer-first.
`
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('ASSUMPTIONS.md')) {
        return Promise.resolve(
          jsonResponse(content('.skillgrid/ASSUMPTIONS.md', body)),
        ) as unknown as Promise<Response>
      }
      return Promise.reject(new Error(`unexpected fetch ${url}`)) as unknown as Promise<Response>
    })

    render(<AdrsPage />)

    // The ADRs tab is the default; open questions live behind their own tab.
    expect(await screen.findByText(/PRD scope is the whole Hub Product/)).toBeTruthy()
    const adrTab = screen.getByRole('tab', { name: /ADRs/ })
    const openTab = screen.getByRole('tab', { name: /Open questions/ })
    expect(adrTab.getAttribute('aria-selected')).toBe('true')
    expect(openTab.textContent).toContain('2')
    expect(screen.queryByText(/v1.1 dashboard prioritization/)).toBeNull()

    fireEvent.click(openTab)
    expect(await screen.findByText(/v1.1 dashboard prioritization/)).toBeTruthy()
    expect(screen.getByText(/README alignment/)).toBeTruthy()
    expect(screen.queryByText(/PRD scope is the whole Hub Product/)).toBeNull()

    fireEvent.click(screen.getByText(/v1.1 dashboard prioritization/))
    expect(await screen.findByText(/Memories-first vs Graph-first/)).toBeTruthy()

    // Switching back restores the ADR list.
    fireEvent.click(adrTab)
    expect(await screen.findByText(/PRD scope is the whole Hub Product/)).toBeTruthy()
  })

  it('shows the empty state when neither source has a table', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('ASSUMPTIONS.md')) {
        return Promise.resolve(jsonResponse({ error: 'doc not found' }, 404)) as unknown as Promise<Response>
      }
      return Promise.resolve(
        jsonResponse(content('.skillgrid/artifacts/03-adr-index.md', STUB_INDEX_BODY)),
      ) as unknown as Promise<Response>
    })

    render(<AdrsPage />)

    expect(await screen.findByText(/no ADR index/i)).toBeTruthy()
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
      if (url.includes('ASSUMPTIONS.md') || url.includes('03-adr-index.md')) {
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
      if (url.includes('ASSUMPTIONS.md') || url.includes('03-adr-index.md')) {
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
