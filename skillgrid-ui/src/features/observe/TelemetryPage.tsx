import { useEffect, useState } from 'react'
import { apiGet } from '../../lib/api'
import {
  EmptyState,
  ErrorState,
  KpiCard,
  LoadingState,
  PageHeader,
  SectionTitle,
  StatusBadge,
  TypeBadge,
} from '../../components/ui/Badges'
import { AgentBadge } from '../sessions/AgentBadge'

interface Session {
  id: string
  status?: string
  memories_writes?: number
  has_summary?: boolean
  last_active?: string
  agent?: string
  last_tool?: string
}

interface Event {
  id: string
  ts?: string
  type?: string
  summary?: string
  actor?: string
}

export function TelemetryPage() {
  const [sessions, setSessions] = useState<Session[] | null>(null)
  const [events, setEvents] = useState<Event[] | null>(null)
  const [sessionsError, setSessionsError] = useState<string | null>(null)
  const [eventsError, setEventsError] = useState<string | null>(null)

  useEffect(() => {
    apiGet<{ sessions?: Session[] }>('/mnemonic/sessions')
      .then((s) => setSessions(s.sessions || []))
      .catch((err: Error) => setSessionsError(err.message))
    apiGet<{ events?: Event[] }>('/activity/events', { limit: 50 })
      .then((e) => setEvents(e.events || []))
      .catch((err: Error) => setEventsError(err.message))
  }, [])

  if (!sessions && !events && !sessionsError && !eventsError) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  return (
    <div className="p-6">
      <PageHeader title="Telemetry" subtitle="Agent observation: session activity + event stream" />

      <div className="mb-4 grid grid-cols-3 gap-4">
        <KpiCard label="Total Sessions" value={sessions?.length || 0} />
        <KpiCard
          label="Active"
          value={(sessions || []).filter((s) => s.status === 'active').length}
        />
        <KpiCard label="Events (50)" value={events?.length || 0} />
      </div>

      <div className="kpi-card mb-4">
        <SectionTitle>Event Stream</SectionTitle>
        {eventsError ? (
          <ErrorState error={eventsError} />
        ) : (
        <div className="max-h-[300px] space-y-0.5 overflow-y-auto">
          {(events || []).map((ev) => (
            <div key={ev.id} className="flex items-center gap-3 px-1 py-1.5 text-xs">
              <span className="w-[130px] shrink-0 font-mono text-ink-4">
                {ev.ts?.slice(0, 19).replace('T', ' ')}
              </span>
              <TypeBadge type={ev.type} />
              <span className="flex-1 truncate text-ink">{ev.summary}</span>
              <span className="shrink-0 text-ink-4">{ev.actor}</span>
            </div>
          ))}
          {(!events || events.length === 0) && <EmptyState message="No events" />}
        </div>
        )}
      </div>

      <div className="kpi-card">
        <SectionTitle>Sessions</SectionTitle>
        {sessionsError ? (
          <ErrorState error={sessionsError} />
        ) : (
        <div className="max-h-[300px] overflow-y-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-edge text-left text-ink-4">
                <th className="px-2 py-1.5">Session</th>
                <th className="px-2 py-1.5">Agent</th>
                <th className="px-2 py-1.5">Last tool</th>
                <th className="px-2 py-1.5">Status</th>
                <th className="px-2 py-1.5">Writes</th>
                <th className="px-2 py-1.5">Summary</th>
                <th className="px-2 py-1.5">Last Active</th>
              </tr>
            </thead>
            <tbody>
              {(sessions || []).map((s) => (
                <tr key={s.id} className="table-row">
                  <td className="max-w-[180px] truncate px-2 py-1.5 font-mono text-ink-3">
                    <a
                      href={`/mnemonic/sessions?id=${encodeURIComponent(s.id)}`}
                      className="hover:text-accent hover:underline"
                    >
                      {s.id?.slice(0, 16)}…
                    </a>
                  </td>
                  <td className="px-2 py-1.5">
                    <AgentBadge agent={s.agent} />
                  </td>
                  <td className="max-w-[140px] truncate px-2 py-1.5 font-mono text-ink-4">{s.last_tool || '—'}</td>
                  <td className="px-2 py-1.5">
                    <StatusBadge status={s.status} />
                  </td>
                  <td className="px-2 py-1.5 font-mono">{s.memories_writes}</td>
                  <td className="px-2 py-1.5">{s.has_summary ? '✓' : '—'}</td>
                  <td className="px-2 py-1.5 font-mono text-ink-4">
                    {s.last_active?.slice(0, 16)?.replace('T', ' ')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        )}
      </div>
    </div>
  )
}
