import { render, screen } from '@testing-library/react'
import { createMemoryHistory, createRootRoute, createRoute, createRouter, RouterProvider } from '@tanstack/react-router'
import { describe, it, expect, vi, beforeAll } from 'vitest'
import { AppLayout } from './AppLayout'

// jsdom lacks scrollTo; TanStack router's scroll restoration calls it on route
// change. Stub it to silence the "Not implemented" warning.
beforeAll(() => {
  window.scrollTo = vi.fn() as unknown as typeof window.scrollTo
})

// Stub pages — AppLayout renders <Outlet />, so each nav target needs a route.
function Page({ name }: { name: string }) {
  return <div data-testid={`page-${name}`}>{name} page</div>
}

// Build a real TanStack router with AppLayout as the root component and one
// route per nav target, so NavLink (Link + useLocation) resolves correctly.
function makeRouter() {
  const root = createRootRoute({ component: AppLayout })
  const route = (path: string, name: string) =>
    createRoute({
      getParentRoute: () => root,
      path,
      component: () => <Page name={name} />,
    })
  const routeTree = root.addChildren([
    route('/tracker', 'tracker'),
    route('/mnemonic/graph', 'graph'),
    route('/mnemonic/files', 'files'),
    route('/mnemonic/memories', 'memories'),
    route('/mnemonic/sessions', 'sessions'),
    route('/mnemonic/search', 'search'),
    route('/plans', 'plans'),
    route('/adr', 'adr'),
    route('/decisions', 'decisions'),
    route('/docs', 'docs'),
    route('/git', 'git'),
    route('/prototypes', 'prototypes'),
    route('/settings', 'settings'),
  ])
  const router = createRouter({ routeTree, history: createMemoryHistory({ initialEntries: ['/tracker'] }) })
  return RouterProvider({ router }) as unknown as React.ReactElement
}

describe('AppLayout navigation', () => {
  it('groups nav into Observe / Mnemonic / Pipeline / System', async () => {
    render(makeRouter())
    const text = await screen.findByText('Tracker').then(() => document.body.textContent ?? '')

    // All four group headers are present.
    for (const group of ['Observe', 'Mnemonic', 'Pipeline', 'System']) {
      expect(text).toContain(group)
    }
    // The newly-added items are present (these fail until the regroup lands).
    for (const item of ['Overview', 'ADRs', 'Swagger']) {
      expect(text).toContain(item)
    }
  })

  it('Observe group contains Overview and Tracker', async () => {
    render(makeRouter())
    await screen.findByText('Tracker')

    const overview = screen.getByText('Overview')
    const observeHeader = screen.getByText('Observe')
    // Overview sits inside the Observe group: after the header, before Mnemonic.
    expect(overview.compareDocumentPosition(observeHeader) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
  })

  it('Pipeline group contains Plans, ADRs and Decisions', async () => {
    render(makeRouter())
    await screen.findByText('Tracker')

    for (const label of ['Plans', 'ADRs', 'Decisions']) {
      const el = screen.getByText(label)
      const pipelineHeader = screen.getByText('Pipeline')
      // Each Pipeline item appears after the Pipeline header.
      expect(el.compareDocumentPosition(pipelineHeader) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
    }
  })

  it('System group contains Docs, Git, Prototypes, Settings and Swagger', async () => {
    render(makeRouter())
    await screen.findByText('Tracker')

    for (const label of ['Docs', 'Git', 'Prototypes', 'Settings', 'Swagger']) {
      const el = screen.getByText(label)
      const systemHeader = screen.getByText('System')
      expect(el.compareDocumentPosition(systemHeader) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
    }
  })

  it('renders an /adr nav link that points at the ADRs route', async () => {
    render(makeRouter())
    await screen.findByText('Tracker')

    const link = screen.getByText('ADRs').closest('a')
    expect(link).toBeTruthy()
    expect(link?.getAttribute('href')).toBe('/adr')
  })
})
