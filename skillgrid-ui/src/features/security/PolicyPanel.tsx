import { useEffect, useState } from 'react'
import { SectionTitle } from '../../components/ui/Badges'
import { fetchPolicy, type PolicyRule, type PolicyView } from '../sessions/api'

const EFFECT_STYLE: Record<PolicyRule['effect'], string> = {
  block: 'bg-red-900/60 text-red-300 border-red-700',
  warn: 'bg-amber-900/60 text-amber-300 border-amber-700',
  guide: 'bg-blue-900/60 text-blue-300 border-blue-700',
  allow: 'bg-green-900/60 text-green-300 border-green-700',
}

function matchSummary(m: PolicyRule['match']): string {
  const parts: string[] = []
  const add = (k: string, v?: string[]) => {
    if (v && v.length > 0) parts.push(`${k}: ${v.join(' | ')}`)
  }
  add('action', m.action)
  add('tool', m.tool)
  add('path', m.path)
  add('command', m.command)
  add('agent', m.agent)
  add('project', m.project)
  for (const [k, v] of Object.entries(m.counters ?? {})) parts.push(`${k} ${v}`)
  return parts.length > 0 ? parts.join(' · ') : 'any call'
}

// PolicyPanel is the read-only view of the pre-tool policy (GET /policy).
// Edit .skillgrid/policy.yaml; `skillgrid policy test` checks a call.
export function PolicyPanel() {
  const [view, setView] = useState<PolicyView | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    fetchPolicy()
      .then((v) => alive && setView(v))
      .catch((e) => alive && setError(e instanceof Error ? e.message : String(e)))
    return () => {
      alive = false
    }
  }, [])

  return (
    <section className="kpi-card" aria-label="Tool policy" data-testid="policy-panel">
      <div className="flex items-center justify-between">
        <SectionTitle>Tool policy</SectionTitle>
        {view && !view.error && (
          <span
            data-testid="policy-state"
            className={`rounded border px-2 py-0.5 text-[10px] uppercase tracking-wider ${
              view.enabled ? EFFECT_STYLE.allow : 'border-edge bg-surface-700 text-ink-4'
            }`}
          >
            {view.enabled ? 'enforced' : 'disabled'}
          </span>
        )}
      </div>
      {error && <div className="text-xs text-red-400">{error}</div>}
      {!view && !error && <div className="text-xs text-ink-4">Loading…</div>}
      {view?.error && (
        <div className="text-xs text-red-400" data-testid="policy-error">
          Policy file error (hooks fail open, every call is allowed): {view.error}
        </div>
      )}
      {view && !view.error && view.files.length === 0 && (
        <div className="text-xs text-ink-4">
          No policy file. Run <code className="font-mono">skillgrid policy init</code> to write a disabled
          starter at <code className="font-mono">{view.repoFile || '.skillgrid/policy.yaml'}</code>.
        </div>
      )}
      {view && !view.error && view.rules.length > 0 && (
        <table className="mt-2 w-full text-left text-xs" data-testid="policy-rules">
          <thead className="text-[10px] uppercase tracking-wider text-ink-4">
            <tr>
              <th className="py-1 pr-2">#</th>
              <th className="py-1 pr-2">Rule</th>
              <th className="py-1 pr-2">Effect</th>
              <th className="py-1 pr-2">Match</th>
              <th className="py-1">Message</th>
            </tr>
          </thead>
          <tbody>
            {view.rules.map((r, i) => (
              <tr key={`${r.source}-${r.name}-${i}`} className="border-t border-edge align-top">
                <td className="py-1 pr-2 text-ink-4">{i + 1}</td>
                <td className="py-1 pr-2 font-mono" title={r.source}>
                  {r.name}
                </td>
                <td className="py-1 pr-2">
                  <span className={`rounded border px-1.5 py-0.5 text-[10px] ${EFFECT_STYLE[r.effect]}`}>
                    {r.effect}
                  </span>
                </td>
                <td className="py-1 pr-2 font-mono text-ink-4">{matchSummary(r.match)}</td>
                <td className="py-1">{r.message || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {view && !view.error && view.files.length > 0 && (
        <div className="mt-2 text-[10px] text-ink-4">
          {view.files.join(' · ')} · first match wins · hooks fail open when skillgrid serve is down
        </div>
      )}
    </section>
  )
}
