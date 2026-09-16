import { Link, Outlet, useLocation } from '@tanstack/react-router'
import { ProjectSelector } from './ProjectSelector'

const MNEMONIC_ITEMS: Array<{ to: string; label: string }> = [
  { to: '/mnemonic/graph', label: 'Graph' },
  { to: '/mnemonic/files', label: 'Files' },
  { to: '/mnemonic/memories', label: 'Memories' },
  { to: '/mnemonic/sessions', label: 'Sessions' },
  { to: '/mnemonic/search', label: 'Search' },
]

const STANDALONE_ITEMS: Array<{ to: string; label: string }> = [
  { to: '/docs', label: 'Docs' },
  { to: '/plans', label: 'Plans' },
  { to: '/activity', label: 'Activity' },
  { to: '/git', label: 'Git' },
  { to: '/prototypes', label: 'Prototypes' },
  { to: '/settings', label: 'Settings' },
]

export function AppLayout() {
  return (
    <div className="flex h-screen overflow-hidden bg-bg">
      <aside className="flex w-60 shrink-0 flex-col border-r border-edge bg-card">
        <div className="flex h-14 items-center gap-2 border-b border-edge px-4">
          <span className="h-2.5 w-2.5 rounded-full bg-accent" />
          <span className="text-sm font-semibold tracking-wide text-zinc-100">
            Skillgrid
          </span>
        </div>
        <nav className="flex-1 overflow-y-auto p-3">
          <ul className="space-y-0.5 text-sm">
            <li>
              <NavLink to="/tracker" label="Tracker" />
            </li>
            <li className="pt-3">
              <div className="px-3 pb-1 text-xs font-medium uppercase tracking-widest text-zinc-500">
                Mnemonic
              </div>
            </li>
            {MNEMONIC_ITEMS.map((item) => (
              <li key={item.to}>
                <NavLink to={item.to} label={item.label} nested />
              </li>
            ))}
            <li className="pt-3" />
            {STANDALONE_ITEMS.map((item) => (
              <li key={item.to}>
                <NavLink to={item.to} label={item.label} />
              </li>
            ))}
          </ul>
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between border-b border-edge bg-card px-4">
          <div className="text-sm text-zinc-500">
            Skillgrid — SPA shell (Phase 1 stubs)
          </div>
          <ProjectSelector />
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function NavLink({ to, label, nested = false }: { to: string; label: string; nested?: boolean }) {
  const { pathname } = useLocation()
  const active = pathname === to
  return (
    <Link
      to={to}
      aria-current={active ? 'page' : undefined}
      className={[
        'block rounded-md px-3 py-1.5 text-zinc-400 transition-colors hover:bg-edge/40 hover:text-zinc-200',
        nested ? 'pl-6' : '',
        active ? 'bg-accent/15 text-zinc-100' : '',
      ].join(' ')}
    >
      {label}
    </Link>
  )
}
