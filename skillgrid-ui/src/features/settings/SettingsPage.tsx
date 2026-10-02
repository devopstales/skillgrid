import { useEffect, useRef, useState, type ReactNode } from 'react'
import { apiGet } from '../../lib/api'
import { LoadingState, PageHeader, SectionTitle } from '../../components/ui/Badges'

const TOKEN_KEY = 'skillgrid.httpToken'

function yamlClean(val: string) {
  let v = val.trim()
  v = v.replace(/\s+#.*$/, '')
  if (v.startsWith('"') && v.endsWith('"') && v.length >= 2) v = v.slice(1, -1)
  return v
}

function parseYamlFlat(text: string): Record<string, string> {
  if (!text) return {}
  const flat: Record<string, string> = {}
  const stack: { indent: number; key: string }[] = []
  for (const line of text.split('\n')) {
    if (line.startsWith('#') || line.trim() === '') continue
    const indent = line.match(/^ */)![0].length
    const m = line.match(/^(\s*)([a-zA-Z_][\w-]*):\s*(.*)$/)
    if (!m) continue
    while (stack.length > 0 && stack[stack.length - 1].indent >= indent) stack.pop()
    const parent = stack.length > 0 ? stack[stack.length - 1].key : ''
    const fullKey = parent ? `${parent}.${m[2]}` : m[2]
    stack.push({ indent, key: fullKey })
    const v = m[3].trim()
    if (v && !v.startsWith('>') && !v.startsWith('|') && v !== '[]' && v !== '{}') {
      flat[fullKey] = yamlClean(v)
    }
  }
  return flat
}

function BoolPill({ value }: { value: string | boolean | undefined }) {
  const on = value === 'true' || value === true
  return <span className={`badge ${on ? 'badge-config' : 'badge-unknown'}`}>{on ? 'enabled' : 'off'}</span>
}

export function SettingsPage() {
  const [token, setToken] = useState('')
  const [saved, setSaved] = useState(false)
  const [config, setConfig] = useState<string | null>(null)
  const [state, setState] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    setToken(localStorage.getItem(TOKEN_KEY) ?? '')
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  useEffect(() => {
    Promise.all([
      apiGet<{ body?: string; content?: string }>(
        '/docs/content',
        { path: '.skillgrid/config.yaml' },
        { project: false },
      ).catch((): { body: string; content?: string } => ({ body: '' })),
      apiGet<{ body?: string; content?: string }>(
        '/docs/content',
        { path: '.skillgrid/state.yaml' },
        { project: false },
      ).catch((): { body: string; content?: string } => ({ body: '' })),
    ])
      .then(([c, s]) => {
        setConfig(c.body || c.content || '')
        setState(s.body || s.content || '')
      })
      .finally(() => setLoaded(true))
  }, [])

  function save() {
    localStorage.setItem(TOKEN_KEY, token)
    setSaved(true)
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setSaved(false), 2000)
  }

  const c = parseYamlFlat(config || '')
  const s = parseYamlFlat(state || '')
  const phase = (s['pipeline.current_phase'] || '').replace(/"/g, '')
  const currentChange = (s['pipeline.current_change'] || '').replace(/"/g, '')
  const pipelineStatus = (s['pipeline.status'] || '').replace(/"/g, '')

  const phaseBadge: Record<string, string> = {
    explore: 'badge-discovery',
    propose: 'badge-decision',
    spec: 'badge-learning',
    apply: 'badge-config',
    verify: 'badge-pattern',
    review: 'badge-correction',
    archive: 'badge-session_log',
    reflect: 'badge-standing',
  }

  const rows: { label: string; value: ReactNode }[] = [
    { label: 'Project', value: <span className="text-sm font-semibold">{c['project'] || '—'}</span> },
    { label: 'Schema', value: <span className="font-mono text-xs text-ink-3">{c['schema'] || '—'}</span> },
    {
      label: 'Phase',
      value: phase ? (
        <span className={`badge ${phaseBadge[phase] || 'badge-unknown'}`}>{phase}</span>
      ) : (
        <span className="text-xs text-ink-4">—</span>
      ),
    },
    {
      label: 'Pipeline',
      value: pipelineStatus ? (
        <span className="text-xs">{pipelineStatus}</span>
      ) : (
        <span className="text-xs text-ink-4">—</span>
      ),
    },
    {
      label: 'Current Change',
      value: currentChange ? (
        <span className="font-mono text-xs">{currentChange}</span>
      ) : (
        <span className="text-xs text-ink-4">idle</span>
      ),
    },
    {
      label: 'Completed',
      value: (
        <span className="font-mono text-xs text-green-400">
          {s['progress.completed_changes'] || '0'}
        </span>
      ),
    },
    { label: 'TDD', value: <BoolPill value={c['testing.tdd']} /> },
    { label: 'Mnemonic', value: <BoolPill value={c['mnemonic.enabled']} /> },
    {
      label: 'Rigor Tier',
      value: <span className="font-mono text-xs">{c['rules.tiers.default'] || '—'}</span>,
    },
  ]

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-6">
      <PageHeader title="Settings" subtitle="Project configuration and client preferences" />

      <section className="kpi-card max-w-xl">
        <h2 className="text-[13px] font-semibold text-ink">HTTP Token</h2>
        <p className="mt-1 text-[12px] text-ink-4">
          Used for API authentication to the Skillgrid backend.
        </p>
        <div className="mt-3 flex items-center gap-2">
          <input
            type="password"
            aria-label="HTTP Token"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            className="min-w-0 flex-1 rounded border border-edge bg-inset px-3 py-1.5 text-[13px] text-ink-2 outline-none focus:border-accent/60"
          />
          <button
            type="button"
            onClick={save}
            className="shrink-0 rounded border border-accent-ink bg-accent/10 px-3 py-1.5 text-[12px] font-medium text-accent hover:bg-accent/20"
          >
            Save
          </button>
        </div>
        {saved && (
          <p role="status" className="mt-2 text-[12px] text-accent">
            Token saved
          </p>
        )}
      </section>

      {!loaded ? (
        <LoadingState />
      ) : (
        <div className="kpi-card">
          <SectionTitle>Project</SectionTitle>
          {config || state ? (
            <div className="grid grid-cols-1 gap-x-8 gap-y-2 md:grid-cols-2">
              {rows.map((r) => (
                <div key={r.label} className="flex items-center justify-between py-1">
                  <span className="text-xs text-ink-4">{r.label}</span>
                  {r.value}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-ink-4">
              Failed to load <code className="rounded bg-surface-700 px-1 font-mono text-xs">.skillgrid/config.yaml</code> or{' '}
              <code className="rounded bg-surface-700 px-1 font-mono text-xs">.skillgrid/state.yaml</code>.
            </p>
          )}
        </div>
      )}

      <section className="kpi-card max-w-xl">
        <h2 className="text-[13px] font-semibold text-ink">About</h2>
        <ul className="mt-1 list-none space-y-0.5 text-[12px] text-ink-4">
          <li>Skillgrid Dashboard</li>
          <li>Mnemonic Admin Console (prototype 001)</li>
          <li>Powered by Skillgrid</li>
        </ul>
      </section>
    </div>
  )
}
