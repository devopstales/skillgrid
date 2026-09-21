import { lazy, Suspense, type ComponentType } from 'react'
import { createRootRoute, createRoute, createRouter, redirect, RouterProvider } from '@tanstack/react-router'
import { AppLayout } from './components/layout/AppLayout'
import { Stub } from './components/Stub'

// Phase 7.3: per-feature-route code-splitting. Each feature page is a lazy
// chunk so the initial bundle stays small; the heavy views (graph, git, docs,
// activity, plans, prototypes, tracker, mnemonic) only load their JS when the
// route is visited. A shared Suspense fallback (spinner) covers the gap.
// Returns a component (not an Element) so it satisfies TanStack's RouteComponent.
function lazyPage(factory: () => Promise<{ default: ComponentType }>) {
  const Lazy = lazy(factory)
  return function LazyRoutePage() {
    return (
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-edge border-t-accent" />
          </div>
        }
      >
        <Lazy />
      </Suspense>
    )
  }
}

const rootRoute = createRootRoute({
  component: AppLayout,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/tracker' })
  },
})

const trackerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/tracker',
  component: lazyPage(() => import('./features/tracker/TrackerPage').then((m) => ({ default: m.TrackerPage }))),
})

const graphRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/graph',
  component: lazyPage(() => import('./features/mnemonic/GraphPage').then((m) => ({ default: m.GraphPage }))),
})

const filesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/files',
  component: lazyPage(() => import('./features/mnemonic/FilesPage').then((m) => ({ default: m.FilesPage }))),
})

const memoriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/memories',
  component: lazyPage(() => import('./features/mnemonic/MemoriesPage').then((m) => ({ default: m.MemoriesPage }))),
})

const sessionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/sessions',
  component: lazyPage(() => import('./features/mnemonic/SessionsPage').then((m) => ({ default: m.SessionsPage }))),
})

const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/search',
  component: lazyPage(() => import('./features/mnemonic/SearchPage').then((m) => ({ default: m.SearchPage }))),
})

const docsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/docs',
  component: lazyPage(() => import('./features/docs/DocsPage').then((m) => ({ default: m.DocsPage }))),
})

const plansRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/plans',
  component: lazyPage(() => import('./features/plans/PlansPage').then((m) => ({ default: m.PlansPage }))),
})

const activityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/activity',
  component: lazyPage(() => import('./features/activity/ActivityPage').then((m) => ({ default: m.ActivityPage }))),
})

const handoffRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/handoff',
  component: lazyPage(() => import('./features/handoff/HandoffHubPage').then((m) => ({ default: m.HandoffHubPage }))),
})

const decisionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/decisions',
  component: lazyPage(() => import('./features/decisions/DecisionsPage').then((m) => ({ default: m.DecisionsPage }))),
})

const gitRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/git',
  component: lazyPage(() => import('./features/git/GitPage').then((m) => ({ default: m.GitPage }))),
})

const prototypesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/prototypes',
  component: lazyPage(() => import('./features/prototypes/PrototypesPage').then((m) => ({ default: m.PrototypesPage }))),
})

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: lazyPage(() => import('./features/settings/SettingsPage').then((m) => ({ default: m.SettingsPage }))),
})

const notFoundRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/__404__',
  component: () => <Stub title="Not found" note="This route does not exist yet" />,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  trackerRoute,
  graphRoute,
  filesRoute,
  memoriesRoute,
  sessionsRoute,
  searchRoute,
  docsRoute,
  plansRoute,
  activityRoute,
  handoffRoute,
  decisionsRoute,
  gitRoute,
  prototypesRoute,
  settingsRoute,
  notFoundRoute,
])

const router = createRouter({ routeTree })

export function App() {
  return <RouterProvider router={router} />
}

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
