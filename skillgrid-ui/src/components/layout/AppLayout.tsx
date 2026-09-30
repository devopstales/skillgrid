import { useEffect, useRef, useState } from 'react'
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
  { to: '/decisions', label: 'Decisions' },
  { to: '/git', label: 'Git' },
  { to: '/prototypes', label: 'Prototypes' },
  { to: '/settings', label: 'Settings' },
]

export function AppLayout() {
  // Phase 7.4 responsive: the sidebar is a slide-over on mobile (hidden by
  // default, opened via the hamburger) and a fixed column on lg+ screens.
  const [navOpen, setNavOpen] = useState(false)
  const hamburgerRef = useRef<HTMLButtonElement>(null)
  const asideRef = useRef<HTMLElement>(null)

  // Review B4: Escape closes the mobile slide-over; focus is moved into the
  // aside on open and returned to the hamburger on close.
  useEffect(() => {
    if (!navOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setNavOpen(false)
    }
    document.addEventListener('keydown', onKey)
    // Move focus into the slide-over (first focusable = the close button).
    const first = asideRef.current?.querySelector<HTMLElement>('[data-nav-close], a, button')
    first?.focus()
    return () => {
      document.removeEventListener('keydown', onKey)
      hamburgerRef.current?.focus()
    }
  }, [navOpen])

  const closeNav = () => setNavOpen(false)
  return (
    <div className="flex h-screen overflow-hidden bg-bg">
      {/* mobile slide-over backdrop */}
      {navOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/50 lg:hidden"
          onClick={closeNav}
          aria-hidden
        />
      )}
      <aside
        ref={asideRef}
        id="app-nav"
        className={[
          'fixed inset-y-0 left-0 z-30 flex w-[220px] shrink-0 flex-col border-r border-edge-soft bg-surface-2 font-mono transition-transform',
          'lg:static lg:translate-x-0',
          navOpen ? 'translate-x-0' : '-translate-x-full',
        ].join(' ')}
      >
        <div className="flex h-14 items-center justify-between gap-2 border-b border-edge-soft px-4">
          <span className="flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-full bg-accent" />
            <span className="text-[13px] font-bold tracking-[0.04em] text-ink">Skillgrid</span>
          </span>
          {/* review B4: visible, labelled close button (mobile only) */}
          <button
            type="button"
            data-nav-close
            onClick={closeNav}
            aria-label="Close navigation"
            className="rounded-md p-1 text-zinc-400 hover:text-zinc-100 lg:hidden"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </div>
        <nav className="flex-1 overflow-y-auto p-3">
          <ul className="space-y-0.5 text-sm">
            <li>
              <div className="px-4 pb-1 pt-1 text-[10px] font-medium uppercase tracking-[0.14em] text-ink-6">
                Observe
              </div>
              <NavLink to="/tracker" label="Tracker" onNavigate={() => setNavOpen(false)} />
            </li>
            <li className="pt-3">
              <div className="px-4 pb-1 text-[10px] font-medium uppercase tracking-[0.14em] text-ink-6">
                Mnemonic
              </div>
            </li>
            {MNEMONIC_ITEMS.map((item) => (
              <li key={item.to}>
                <NavLink to={item.to} label={item.label} nested onNavigate={() => setNavOpen(false)} />
              </li>
            ))}
            <li className="pt-3">
              <div className="px-4 pb-1 text-[10px] font-medium uppercase tracking-[0.14em] text-ink-6">
                Pipeline
              </div>
              {STANDALONE_ITEMS.slice(0, 5).map((item) => (
                <NavLink key={item.to} to={item.to} label={item.label} onNavigate={() => setNavOpen(false)} />
              ))}
            </li>
            <li className="pt-3">
              <div className="px-4 pb-1 text-[10px] font-medium uppercase tracking-[0.14em] text-ink-6">
                System
              </div>
              {STANDALONE_ITEMS.slice(5).map((item) => (
                <NavLink key={item.to} to={item.to} label={item.label} onNavigate={() => setNavOpen(false)} />
              ))}
            </li>
          </ul>
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-edge-soft bg-bg px-4">
          <div className="flex items-center gap-2">
            {/* hamburger — mobile only. Review B4: aria-expanded + aria-controls
                so assistive tech knows whether navigation is open. */}
            <button
              ref={hamburgerRef}
              type="button"
              onClick={() => setNavOpen((v) => !v)}
              className="rounded-md p-1.5 text-ink-4 hover:text-ink lg:hidden"
              aria-label="Toggle navigation"
              aria-expanded={navOpen}
              aria-controls="app-nav"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M3 6h18M3 12h18M3 18h18" />
              </svg>
            </button>
            <div className="hidden items-center gap-2 font-mono text-[12px] text-ink-4 sm:flex">
              <span>~/skillgrid</span>
              <span className="text-ink-6">/</span>
              <span className="font-semibold text-ink">dashboard</span>
            </div>
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
      // Review B5: expose the toggle state (aria-pressed) — the visible text
      // alone does not announce the on/off semantic to assistive tech.
      aria-pressed={density === 'compact'}
      className="rounded border border-edge-soft bg-inset px-2 py-1 text-[11px] text-ink-4 transition-colors hover:text-ink"
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
        'block border-l-2 border-transparent px-4 py-1.5 text-[13px] text-ink-4 transition-colors hover:text-ink-2',
        nested ? 'pl-7' : '',
        active ? 'border-accent bg-accent/7 text-ink' : '',
      ].join(' ')}
    >
      {label}
    </Link>
  )
}
