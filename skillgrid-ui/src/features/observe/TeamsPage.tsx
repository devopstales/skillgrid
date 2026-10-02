import { useEffect, useState } from 'react'
import { apiGet } from '../../lib/api'
import {
  EmptyState,
  ErrorState,
  KpiCard,
  LoadingState,
  PageHeader,
  StatusBadge,
} from '../../components/ui/Badges'

// The server joins a ledger row to the Mnemonic session whose title names its
// ticket or task id; live is working, idle, or finished.
interface MemberSession {
  id: string
  project?: string
  agent?: string
  status?: string
  title?: string
  live?: string
}

interface TeamMember {
  agent?: string
  task?: string
  status?: string
  owns?: string
  test?: string
  session?: MemberSession
}

function sessionHref(id: string, project?: string): string {
  const q = new URLSearchParams({ id })
  if (project) q.set('store', project)
  return `/mnemonic/sessions?${q.toString()}`
}

interface TeamRun {
  name: string
  heading?: string
  ledger?: string
  wave_ledger?: string
  updated?: string
  in_flight?: number
  done?: number
  members?: TeamMember[]
}

interface TeamRuns {
  count?: number
  in_flight?: number
  done?: number
  runs?: TeamRun[]
}

interface LiveSession {
  id: string
  title?: string
  status?: string
  agent?: string
  last_active?: string
}

// A harness that touched a tool in this window is still working. Quieter open
// sessions are idle. An ended session is finished. Same three readings Herdr
// uses for a pane: working, idle, and done.
const WORKING_MS = 2 * 60 * 1000

export function agentLiveStatus(
  session: LiveSession,
  now = Date.now(),
): 'working' | 'idle' | 'finished' {
  const status = (session.status || '').toLowerCase()
  if (status === 'ended' || status === 'finished' || status === 'done') return 'finished'
  const last = Date.parse(session.last_active || '')
  if (!Number.isNaN(last) && now - last >= 0 && now - last <= WORKING_MS) return 'working'
  return 'idle'
}

function liveAgents(sessions: LiveSession[]): LiveSession[] {
  const rank = { working: 0, idle: 1, finished: 2 }
  return sessions
    .filter((s) => (s.agent || '').trim() !== '')
    .sort((a, b) => rank[agentLiveStatus(a)] - rank[agentLiveStatus(b)])
}

// Backlog card ids look like TASK-030.05. The Kanban board opens that card
// from /tracker?task=<id>.
const TASK_ID_RE = /TASK-\d+(?:\.\d+)?/i

function TaskCell({ task }: { task?: string }) {
  if (!task) return <>—</>
  const match = task.match(TASK_ID_RE)
  if (!match || match.index === undefined) return <>{task}</>
  const id = match[0]
  return (
    <>
      {task.slice(0, match.index)}
      <a href={`/tracker?task=${encodeURIComponent(id)}`} className="text-accent hover:underline">
        {id}
      </a>
      {task.slice(match.index + id.length)}
    </>
  )
}

export function TeamsPage() {
  const [data, setData] = useState<TeamRuns | null>(null)
  const [agents, setAgents] = useState<LiveSession[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const load = () => {
      apiGet<TeamRuns>('/sdd/runs', {}, { project: false })
        .then((runs) => {
          if (!cancelled) setData(runs)
        })
        .catch((err: Error) => {
          if (!cancelled) setError(err.message)
        })
      apiGet<{ sessions?: LiveSession[] }>('/mnemonic/sessions')
        .then((res) => {
          if (!cancelled) setAgents(res.sessions || [])
        })
        .catch(() => {
          if (!cancelled) setAgents([])
        })
    }
    load()
    const timer = setInterval(load, 15000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [])

  if (error) {
    return (
      <div className="p-6">
        <ErrorState error={error} />
      </div>
    )
  }
  if (!data) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  const runs = data.runs || []
  const agentsLive = liveAgents(agents)

  return (
    <div className="p-6">
      <PageHeader
        title="Teams"
        subtitle="Live execution ledgers under .skillgrid/sdd. The file is the record."
      />

      {agentsLive.length > 0 && (
        <section className="kpi-card mb-4">
          <div className="text-[10px] uppercase tracking-wide text-ink-4">Live agents</div>
          <ul className="mt-2 divide-y divide-edge">
            {agentsLive.map((s) => (
              <li key={s.id} className="flex items-center gap-3 py-1.5">
                <a
                  href={sessionHref(s.id)}
                  className="font-mono text-sm text-accent hover:underline"
                >
                  {s.agent}
                </a>
                <StatusBadge status={agentLiveStatus(s)} />
                {s.title && <span className="truncate text-xs text-ink-3">{s.title}</span>}
              </li>
            ))}
          </ul>
        </section>
      )}

      <div className="mb-4 grid grid-cols-3 gap-4">
        <KpiCard label="Runs" value={data.count || runs.length} />
        <KpiCard label="In flight" value={data.in_flight || 0} />
        <KpiCard label="Done" value={data.done || 0} />
      </div>

      {runs.length === 0 ? (
        <div className="kpi-card">
          <EmptyState message="No execution ledgers under .skillgrid/sdd/" />
        </div>
      ) : (
        <div className="space-y-3">
          {runs.map((run) => (
            <section key={run.name} className="kpi-card">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <a
                    href={`/plans?change=${encodeURIComponent(run.name)}`}
                    className="font-mono text-sm text-accent hover:underline"
                  >
                    {run.name}
                  </a>
                  {run.heading && <div className="mt-1 text-xs text-ink-3">{run.heading}</div>}
                </div>
                {run.updated && (
                  <span className="shrink-0 font-mono text-[10px] text-ink-4">{run.updated}</span>
                )}
              </div>
              <div className="mt-2 text-[10px] text-ink-4">
                {run.in_flight || 0} in flight · {run.done || 0} done
              </div>
              {(run.members || []).length > 0 && (
                <div className="mt-3 overflow-x-auto">
                  <table className="w-full text-left text-xs">
                    <thead className="text-[10px] uppercase tracking-wide text-ink-4">
                      <tr>
                        <th className="py-1 pr-3 font-medium">Agent</th>
                        <th className="py-1 pr-3 font-medium">Task</th>
                        <th className="py-1 pr-3 font-medium">Harness</th>
                        <th className="py-1 pr-3 font-medium">Live</th>
                        <th className="py-1 pr-3 font-medium">Status</th>
                        <th className="py-1 pr-3 font-medium">Owns</th>
                        <th className="py-1 font-medium">Test</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(run.members || []).map((m, i) => (
                        <tr key={`${m.agent}-${m.task}-${i}`} className="border-t border-edge">
                          <td className="py-1.5 pr-3 font-mono text-ink">{m.agent || '—'}</td>
                          <td className="py-1.5 pr-3 text-ink-3">
                            <TaskCell task={m.task} />
                          </td>
                          <td className="py-1.5 pr-3 font-mono">
                            {m.session ? (
                              <a
                                href={sessionHref(m.session.id, m.session.project)}
                                className="text-accent hover:underline"
                                title={m.session.title}
                              >
                                {m.session.agent || m.session.id.slice(0, 8)}
                              </a>
                            ) : (
                              <span className="text-ink-4">—</span>
                            )}
                          </td>
                          <td className="py-1.5 pr-3">
                            {m.session?.live ? (
                              <StatusBadge status={m.session.live} />
                            ) : (
                              <span className="text-ink-4">—</span>
                            )}
                          </td>
                          <td className="py-1.5 pr-3">
                            <StatusBadge status={m.status} />
                          </td>
                          <td className="max-w-[240px] truncate py-1.5 pr-3 font-mono text-[10px] text-ink-4" title={m.owns}>
                            {m.owns || '—'}
                          </td>
                          <td className="py-1.5 text-ink-4">{m.test || '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
