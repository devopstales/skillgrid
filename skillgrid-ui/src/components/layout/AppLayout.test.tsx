import { render, screen } from '@testing-library/react'
import { createMemoryHistory, createRootRoute, createRoute, createRouter, RouterProvider } from '@tanstack/react-router'
import { describe, it, expect, vi, beforeAll } from 'vitest'
import { AppLayout } from './AppLayout'

beforeAll(() => {
  window.scrollTo = vi.fn() as unknown as typeof window.scrollTo
})

function Page({ name }: { name: string }) {
  return <div data-testid={`page-${name}`}>{name} page</div>
}

function makeRouter() {
  const root = createRootRoute({ component: AppLayout })
  const route = (path: string, name: string) =>
    createRoute({
      getParentRoute: () => root,
      path,
      component: () => <Page name={name} />,
    })
  const routeTree = root.addChildren([
    route('/', 'overview'),
    route('/tracker', 'kanban'),
    route('/plans', 'changes'),
    route('/adr', 'adr'),
    route('/project/spikes', 'spikes'),
    route('/mnemonic/graph', 'graph'),
    route('/mnemonic/files', 'files'),
    route('/mnemonic/sessions', 'sessions'),
    route('/observe/telemetry', 'telemetry'),
    route('/observe/compaction', 'compaction'),
    route('/observe/web-cache', 'web-cache'),
    route('/docs', 'docs'),
    route('/system/security', 'security'),
    route('/settings', 'settings'),
    route('/swagger-ui', 'swagger'),
  ])
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ['/tracker'] }),
  })
  return RouterProvider({ router }) as unknown as React.ReactElement
}

describe('AppLayout navigation', () => {
  it('groups nav into Project / Memory / Observe / System', async () => {
    render(makeRouter())
    const text = await screen.findByText('Kanban').then(() => document.body.textContent ?? '')

    for (const group of ['Project', 'Memory', 'Observe', 'System']) {
      expect(text).toContain(group)
    }
    for (const item of ['Overview', 'Code Graph', 'Swagger Docs', 'Security']) {
      expect(text).toContain(item)
    }
  })

  it('Project group contains Kanban and Spikes', async () => {
    render(makeRouter())
    await screen.findByText('Kanban')

    const projectHeader = screen.getByText('Project')
    for (const label of ['Kanban', 'Changes', 'Decisions (ADR)', 'Spikes']) {
      const el = screen.getByText(label)
      expect(el.compareDocumentPosition(projectHeader) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
    }
  })

  it('Observe group contains Telemetry, Compaction, Web Cache', async () => {
    render(makeRouter())
    await screen.findByText('Kanban')

    const observeHeader = screen.getByText('Observe')
    for (const label of ['Telemetry', 'Compaction', 'Web Cache']) {
      const el = screen.getByText(label)
      expect(el.compareDocumentPosition(observeHeader) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
    }
  })

  it('brand reads Mnemonic Enterprise Admin Console', async () => {
    render(makeRouter())
    await screen.findByText('Kanban')
    expect(screen.getAllByText('Mnemonic').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Enterprise Admin Console')).toBeTruthy()
  })
})
