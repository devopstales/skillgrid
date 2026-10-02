import { useEffect, useRef, useState } from 'react'
import { Link, Outlet, useLocation } from '@tanstack/react-router'
import { ProjectSelector } from './ProjectSelector'
import { getDensity, setDensity, type Density } from '../../lib/density'
import { resolveProject } from '../../lib/api'

type NavEntry =
  | { group: string }
  | { id: string; to: string; label: string; icon: string; top?: boolean }

const NAV: NavEntry[] = [
  { id: 'overview', to: '/', label: 'Overview', icon: '◉', top: true },
  { group: 'Project' },
  { id: 'kanban', to: '/tracker', label: 'Kanban', icon: '▦' },
  { id: 'changes', to: '/plans', label: 'Changes', icon: '📋' },
  { id: 'decisions', to: '/adr', label: 'Decisions', icon: '⚖' },
  { id: 'prototypes', to: '/project/prototypes', label: 'Prototypes', icon: '🔬' },
  { group: 'Memory' },
  { id: 'sessions', to: '/mnemonic/sessions', label: 'Sessions', icon: '💬' },
  { id: 'code-graph', to: '/mnemonic/graph', label: 'Code Graph', icon: '⚙' },
  { id: 'memfs', to: '/mnemonic/files', label: 'MemFS', icon: '📁' },
  { group: 'Observe' },
  { id: 'telemetry', to: '/observe/telemetry', label: 'Telemetry', icon: '📊' },
  { id: 'agent-stats', to: '/observe/agents', label: 'Agent Stats', icon: '📈' },
  { id: 'compaction', to: '/observe/compaction', label: 'Compaction', icon: '📦' },
  { id: 'web-cache', to: '/observe/web-cache', label: 'Web Cache', icon: '🌐' },
  { id: 'teams', to: '/observe/teams', label: 'Teams', icon: '👥' },
  { id: 'docs', to: '/docs', label: 'Docs', icon: '📄', top: true },
  { group: 'System' },
  { id: 'security', to: '/system/security', label: 'Security', icon: '🛡' },
  { id: 'settings', to: '/settings', label: 'Settings', icon: '⚙' },
  { id: 'swagger', to: '/swagger-ui', label: 'Swagger Docs', icon: '📖' },
]

export function AppLayout() {
  const [navOpen, setNavOpen] = useState(false)
  const [project, setProject] = useState('…')
  const hamburgerRef = useRef<HTMLButtonElement>(null)
  const asideRef = useRef<HTMLElement>(null)

  useEffect(() => {
    resolveProject().then(setProject).catch(() => setProject('project'))
  }, [])

  useEffect(() => {
    if (!navOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setNavOpen(false)
    }
    document.addEventListener('keydown', onKey)
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
      {navOpen && (
        <div className="fixed inset-0 z-20 bg-black/50 lg:hidden" onClick={closeNav} aria-hidden />
      )}
      <aside
        ref={asideRef}
        id="app-nav"
        className={[
          'fixed inset-y-0 left-0 z-30 flex w-56 shrink-0 flex-col border-r border-edge bg-surface-800 transition-transform',
          'lg:static lg:translate-x-0',
          navOpen ? 'translate-x-0' : '-translate-x-full',
        ].join(' ')}
      >
        <div className="flex items-center justify-between border-b border-edge p-4">
          <div>
            <div className="text-sm font-bold text-accent">Mnemonic</div>
            <div className="mt-0.5 text-[10px] text-ink-4">Enterprise Admin Console</div>
          </div>
          <button
            type="button"
            data-nav-close
            onClick={closeNav}
            aria-label="Close navigation"
            className="rounded-md p-1 text-ink-4 hover:text-ink lg:hidden"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </div>
        <nav className="flex-1 overflow-y-auto py-2">
          {NAV.map((item) => {
            if ('group' in item) {
              return (
                <div key={item.group} className="nav-group-label">
                  {item.group}
                </div>
              )
            }
            return (
              <NavLink
                key={item.id}
                to={item.to}
                label={item.label}
                icon={item.icon}
                top={item.top}
                onNavigate={closeNav}
              />
            )
          })}
        </nav>
        <div className="border-t border-edge p-3 text-[10px] text-ink-4">
          <div>
            Project: <span className="text-ink-3">{project}</span>
          </div>
          <div className="mt-1 flex items-center justify-between gap-2">
            <span>
              API: <span className="font-mono">127.0.0.1:7438</span>
            </span>
            <DensityToggle />
          </div>
          <div className="mt-2">
            <ProjectSelector />
          </div>
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-12 shrink-0 items-center gap-2 border-b border-edge bg-bg px-4 lg:hidden">
          <button
            ref={hamburgerRef}
            type="button"
            onClick={() => setNavOpen((v) => !v)}
            className="rounded-md p-1.5 text-ink-4 hover:text-ink"
            aria-label="Toggle navigation"
            aria-expanded={navOpen}
            aria-controls="app-nav"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M3 6h18M3 12h18M3 18h18" />
            </svg>
          </button>
          <span className="text-sm font-semibold text-accent">Mnemonic</span>
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto bg-surface-900">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

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
      title={`Density: ${density}`}
      aria-pressed={density === 'compact'}
      className="rounded border border-edge bg-inset px-1.5 py-0.5 text-[10px] text-ink-4 hover:text-ink"
    >
      {density === 'comfortable' ? 'Comfort' : 'Compact'}
    </button>
  )
}

function NavLink({
  to,
  label,
  icon,
  top = false,
  onNavigate,
}: {
  to: string
  label: string
  icon: string
  top?: boolean
  onNavigate?: () => void
}) {
  const { pathname } = useLocation()
  const active = pathname === to || (to !== '/' && pathname.startsWith(to + '/'))
  const cls = `${top ? 'nav-item top' : 'nav-item sub'} ${active ? 'active' : ''}`
  return (
    <Link to={to} aria-current={active ? 'page' : undefined} onClick={onNavigate} className={cls}>
      <span className="nav-icon text-sm">{icon}</span>
      <span>{label}</span>
    </Link>
  )
}
