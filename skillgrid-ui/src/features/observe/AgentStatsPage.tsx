import { useEffect, useState } from 'react'
import {
  EmptyState,
  ErrorState,
  KpiCard,
  LoadingState,
  PageHeader,
  SectionTitle,
} from '../../components/ui/Badges'
import { fetchEventStats, type EventStats } from '../sessions/api'
import { AgentBadge, McpBadge } from '../sessions/AgentBadge'
import { formatCost, formatTokens } from '../sessions/format'

const WINDOWS = [
  { v: '1h', label: 'Last hour' },
  { v: '24h', label: 'Today (24h)' },
  { v: '7d', label: '7 days' },
  { v: '30d', label: '30 days' },
  { v: '', label: 'All time' },
]

const REFRESH_MS = 15_000

// AgentStatsPage — Gryph-style `gryph stats`: what every coding agent did in
// the window, per harness, with the hottest files, commands, and tools, plus
// token and cost totals. Rows link to the filtered Sessions timeline.
export function AgentStatsPage() {
  const [since, setSince] = useState('24h')
  const [agent, setAgent] = useState('')
  const [stats, setStats] = useState<EventStats | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    const load = () =>
      fetchEventStats({ since, agent })
        .then((s) => {
          if (!cancelled) {
            setStats(s)
            setError('')
          }
        })
        .catch((e: Error) => {
          if (!cancelled) setError(e.message)
        })
    load()
    const t = setInterval(load, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(t)
    }
  }, [since, agent])

  const select =
    'rounded border border-edge-soft bg-inset px-2 py-1 font-mono text-[12px] text-ink-3 focus:border-accent focus:outline-none'

  return (
    <div className="p-6">
      <div className="flex items-start justify-between gap-4">
        <PageHeader title="Agent Stats" subtitle="Every tool call, file, command, and MCP call by agent" />
        <div className="flex gap-2">
          <select aria-label="agent" value={agent} onChange={(e) => setAgent(e.target.value)} className={select}>
            <option value="">all agents</option>
            {(stats?.byAgent ?? []).map((a) => (
              <option key={a.agent} value={a.agent === 'mnemonic' ? '' : a.agent}>
                {a.agent}
              </option>
            ))}
          </select>
          <select aria-label="window" value={since} onChange={(e) => setSince(e.target.value)} className={select}>
            {WINDOWS.map((w) => (
              <option key={w.v} value={w.v}>
                {w.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {error ? (
        <ErrorState error={error} />
      ) : !stats ? (
        <LoadingState />
      ) : (
        <>
          <div className="mb-4 grid grid-cols-2 gap-4 md:grid-cols-4 xl:grid-cols-8">
            <KpiCard label="Tool calls" value={stats.total} />
            <KpiCard label="Sessions" value={stats.sessions} />
            <KpiCard label="MCP calls" value={stats.mcp} />
            <KpiCard label="Errors" value={stats.errors} />
            <KpiCard label="Sensitive" value={stats.sensitive} />
            <KpiCard label="Blocked" value={stats.blocked} />
            <KpiCard label="Warned" value={stats.warned ?? 0} />
            <KpiCard label="Guided" value={stats.guided ?? 0} />
          </div>

          <div className="kpi-card mb-4">
            <SectionTitle>By agent</SectionTitle>
            {stats.byAgent.length === 0 ? (
              <EmptyState message="No agent activity in this window" />
            ) : (
              <table className="w-full font-mono text-xs" data-testid="agent-table">
                <thead>
                  <tr className="border-b border-edge text-left text-ink-4">
                    {['Agent', 'Sessions', 'Calls', 'Reads', 'Writes', 'Commands', 'MCP', 'Errors', 'Blocked', 'Tokens', 'Cost'].map((h) => (
                      <th key={h} className="px-2 py-1.5">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {stats.byAgent.map((a) => (
                    <tr key={a.agent} className="table-row">
                      <td className="px-2 py-1.5">
                        <AgentBadge agent={a.agent} />
                      </td>
                      <td className="px-2 py-1.5">{a.sessions}</td>
                      <td className="px-2 py-1.5">{a.events}</td>
                      <td className="px-2 py-1.5">{a.reads}</td>
                      <td className="px-2 py-1.5">{a.writes}</td>
                      <td className="px-2 py-1.5">{a.commands}</td>
                      <td className="px-2 py-1.5">{a.mcp}</td>
                      <td className={`px-2 py-1.5 ${a.errors ? 'text-danger' : ''}`}>{a.errors}</td>
                      <td className={`px-2 py-1.5 ${a.blocked ? 'text-danger' : ''}`}>{a.blocked}</td>
                      <td className="px-2 py-1.5">{formatTokens(a.inputTokens + a.outputTokens)}</td>
                      <td className="px-2 py-1.5">{formatCost(a.costUsd)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {(stats.byModel ?? []).length > 0 && (
            <div className="kpi-card mb-4">
              <SectionTitle>Cost by model</SectionTitle>
              <table className="w-full font-mono text-xs" data-testid="model-table">
                <thead>
                  <tr className="border-b border-edge text-left text-ink-4">
                    {['Model', 'Sessions', 'Input', 'Output', 'Cache', 'Cost'].map((h) => (
                      <th key={h} className="px-2 py-1.5">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(stats.byModel ?? []).map((m) => (
                    <tr key={m.model} className="table-row">
                      <td className="px-2 py-1.5 text-ink-2">{m.model}</td>
                      <td className="px-2 py-1.5">{m.sessions}</td>
                      <td className="px-2 py-1.5">{formatTokens(m.inputTokens)}</td>
                      <td className="px-2 py-1.5">{formatTokens(m.outputTokens)}</td>
                      <td className="px-2 py-1.5">{formatTokens(m.cacheTokens)}</td>
                      <td className="px-2 py-1.5">{formatCost(m.costUsd)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="grid gap-4 md:grid-cols-3">
            <TopList title="Top files" items={stats.topFiles} />
            <TopList title="Top commands" items={stats.topCommands} />
            <TopList title="Top tools" items={stats.topTools} />
          </div>

          <div className="mt-4 text-[12px] text-ink-5">
            By action:{' '}
            {Object.entries(stats.byAction)
              .sort((a, b) => b[1] - a[1])
              .map(([k, v]) => `${k} ${v}`)
              .join(' · ') || '—'}
            {' · '}
            <a href="/mnemonic/sessions" className="text-accent hover:underline">
              open the live timeline
            </a>
          </div>
        </>
      )}
    </div>
  )
}

function TopList({ title, items }: { title: string; items: { name: string; count: number; mcp?: boolean }[] }) {
  const max = Math.max(1, ...items.map((i) => i.count))
  return (
    <div className="kpi-card">
      <SectionTitle>{title}</SectionTitle>
      {items.length === 0 ? (
        <EmptyState message="None" />
      ) : (
        <ul className="space-y-1 font-mono text-xs">
          {items.map((i) => (
            <li key={i.name} className="relative flex items-center gap-2 overflow-hidden rounded px-1.5 py-1">
              <span className="absolute inset-y-0 left-0 bg-accent/10" style={{ width: `${(i.count / max) * 100}%` }} />
              <span className="relative min-w-0 flex-1 truncate text-ink-3" title={i.name}>
                {i.name}
              </span>
              {i.mcp && (
                <span className="relative">
                  <McpBadge />
                </span>
              )}
              <span className="relative tabular-nums text-ink-4">{i.count}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
