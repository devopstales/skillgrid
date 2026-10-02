import { lazy, Suspense, type ComponentType } from 'react'
import { createRootRoute, createRoute, createRouter, RouterProvider } from '@tanstack/react-router'
import { AppLayout } from './components/layout/AppLayout'
import { Stub } from './components/Stub'

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
  component: lazyPage(() => import('./features/overview/OverviewPage').then((m) => ({ default: m.OverviewPage }))),
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

const decisionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/decisions',
  component: lazyPage(() => import('./features/decisions/DecisionsPage').then((m) => ({ default: m.DecisionsPage }))),
})

const adrRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/adr',
  component: lazyPage(() => import('./features/adr/AdrsPage').then((m) => ({ default: m.AdrsPage }))),
})

const gitRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/git',
  component: lazyPage(() => import('./features/git/GitPage').then((m) => ({ default: m.GitPage }))),
})

const swaggerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/swagger-ui',
  component: lazyPage(() => import('./features/swagger/SwaggerPage').then((m) => ({ default: m.SwaggerPage }))),
})

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: lazyPage(() => import('./features/settings/SettingsPage').then((m) => ({ default: m.SettingsPage }))),
})

const prototypesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/project/prototypes',
  component: lazyPage(() => import('./features/prototypes/PrototypesPage').then((m) => ({ default: m.PrototypesPage }))),
})

const telemetryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/observe/telemetry',
  component: lazyPage(() => import('./features/observe/TelemetryPage').then((m) => ({ default: m.TelemetryPage }))),
})

const agentStatsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/observe/agents',
  component: lazyPage(() => import('./features/observe/AgentStatsPage').then((m) => ({ default: m.AgentStatsPage }))),
})

const compactionRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/observe/compaction',
  component: lazyPage(() => import('./features/observe/CompactionPage').then((m) => ({ default: m.CompactionPage }))),
})

const webCacheRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/observe/web-cache',
  component: lazyPage(() => import('./features/observe/WebCachePage').then((m) => ({ default: m.WebCachePage }))),
})

const teamsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/observe/teams',
  component: lazyPage(() => import('./features/observe/TeamsPage').then((m) => ({ default: m.TeamsPage }))),
})

const securityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system/security',
  component: lazyPage(() => import('./features/security/SecurityPage').then((m) => ({ default: m.SecurityPage }))),
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
  decisionsRoute,
  adrRoute,
  gitRoute,
  swaggerRoute,
  settingsRoute,
  prototypesRoute,
  telemetryRoute,
  agentStatsRoute,
  compactionRoute,
  webCacheRoute,
  teamsRoute,
  securityRoute,
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
