import { createRootRoute, createRoute, createRouter, redirect, RouterProvider } from '@tanstack/react-router'
import { ActivityPage } from './features/activity/ActivityPage'
import { DocsPage } from './features/docs/DocsPage'
import { GitPage } from './features/git/GitPage'
import { GraphPage } from './features/mnemonic/GraphPage'
import { FilesPage } from './features/mnemonic/FilesPage'
import { MemoriesPage } from './features/mnemonic/MemoriesPage'
import { SessionsPage } from './features/mnemonic/SessionsPage'
import { SearchPage } from './features/mnemonic/SearchPage'
import { PlansPage } from './features/plans/PlansPage'
import { PrototypesPage } from './features/prototypes/PrototypesPage'
import { SettingsPage } from './features/settings/SettingsPage'
import { TrackerPage } from './features/tracker/TrackerPage'
import { AppLayout } from './components/layout/AppLayout'
import { Stub } from './components/Stub'

const rootRoute = createRootRoute({
  component: AppLayout,
})

const trackerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/tracker',
  component: TrackerPage,
})

const graphRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/graph',
  component: GraphPage,
})

const filesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/files',
  component: FilesPage,
})

const memoriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/memories',
  component: MemoriesPage,
})

const sessionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/sessions',
  component: SessionsPage,
})

const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/mnemonic/search',
  component: SearchPage,
})

const docsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/docs',
  component: DocsPage,
})

const plansRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/plans',
  component: PlansPage,
})

const activityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/activity',
  component: ActivityPage,
})

const gitRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/git',
  component: GitPage,
})

const prototypesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/prototypes',
  component: PrototypesPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: SettingsPage,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/tracker' })
  },
})

const notFoundRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/__404__',
  component: () => (
    <Stub title="Not found" note="This route does not exist yet" />
  ),
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
