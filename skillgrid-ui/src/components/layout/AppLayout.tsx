import { useState } from 'react'
import { Link, Outlet, useLocation } from '@tanstack/react-router'
import { ProjectSelector } from './ProjectSelector'
import { getDensity, setDensity, type Density } from '../../lib/density'

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
  // Phase 7.4 responsive: the sidebar is a slide-over on mobile (hidden by
  // default, opened via the hamburger) and a fixed column on lg+ screens.
  const [navOpen, setNavOpen] = useState(false)
  return (
    <div className="flex h-screen overflow-hidden bg-bg">
      {/* mobile slide-over backdrop */}
      {navOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/50 lg:hidden"
          onClick={() => setNavOpen(false)}
          aria-hidden
        />
      )}
      <aside
        className={[
          'fixed inset-y-0 left-0 z-30 flex w-60 shrink-0 flex-col border-r border-edge bg-card transition-transform',
          'lg:static lg:translate-x-0',
          navOpen ? 'translate-x-0' : '-translate-x-full',
        ].join(' ')}
      >
        <div className="flex h-14 items-center gap-2 border-b border-edge px-4">
          <span className="h-2.5 w-2.5 rounded-full bg-accent" />
          <span className="text-sm font-semibold tracking-wide text-zinc-100">Skillgrid</span>
        </div>
        <nav className="flex-1 overflow-y-auto p-3">
          <ul className="space-y-0.5 text-sm">
            <li>
              <NavLink to="/tracker" label="Tracker" onNavigate={() => setNavOpen(false)} />
            </li>
            <li className="pt-3">
              <div className="px-3 pb-1 text-xs font-medium uppercase tracking-widest text-zinc-500">
                Mnemonic
              </div>
            </li>
            {MNEMONIC_ITEMS.map((item) => (
              <li key={item.to}>
                <NavLink to={item.to} label={item.label} nested onNavigate={() => setNavOpen(false)} />
              </li>
            ))}
            <li className="pt-3" />
            {STANDALONE_ITEMS.map((item) => (
              <li key={item.to}>
                <NavLink to={item.to} label={item.label} onNavigate={() => setNavOpen(false)} />
              </li>
            ))}
          </ul>
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between gap-2 border-b border-edge bg-card px-4">
          <div className="flex items-center gap-2">
            {/* hamburger — mobile only */}
            <button
              type="button"
              onClick={() => setNavOpen((v) => !v)}
              className="rounded-md p-1.5 text-zinc-400 hover:text-zinc-100 lg:hidden"
              aria-label="Toggle navigation"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M3 6h18M3 12h18M3 18h18" />
              </svg>
            </button>
            <div className="hidden text-sm text-zinc-500 sm:block">Skillgrid</div>
          </div>
          <div className="flex items-center gap-2">
            <DensityToggle />
            <ProjectSelector />
          </div>
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

// DensityToggle — Phase 7.4 comfortable/compact switch (persisted).
function DensityToggle() {
  const [density, setDensityState] = useState<Density>(getDensity)
  const toggle = () => {
    const next: Density = density === 'comfortable' ? 'compact' : 'comfortable'
    setDensity(next)
    setDensityState(next)
  }
  return (
    <button
      type="button"
      onClick={toggle}
      title={`Density: ${density} (click to switch)`}
      className="rounded-md border border-edge px-2 py-1 text-xs text-zinc-400 transition-colors hover:text-zinc-100"
    >
      {density === 'comfortable' ? 'Comfortable' : 'Compact'}
    </button>
  )
}

function NavLink({
  to,
  label,
  nested = false,
  onNavigate,
}: {
  to: string
  label: string
  nested?: boolean
  onNavigate?: () => void
}) {
  const { pathname } = useLocation()
  const active = pathname === to
  return (
    <Link
      to={to}
      aria-current={active ? 'page' : undefined}
      onClick={onNavigate}
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
