import { useEffect, useState } from 'react'
import { fetchActivityEvents, fetchActivityStats, fetchCodeStatus, fetchPipelineState } from './api'
import type { ActivityEvent, ActivityStats, CodeStatus, PipelineState } from './types'

// OverviewPage is the `/` landing page: a Terminal Ops dashboard of project
// health. A top row of four KPI cards (active sessions, total events, code
// index, pipeline phase) sits above a two-column area — Recent Activity on the
// left, Pipeline Status on the right. All data is a one-shot static fetch on
// mount (no SSE here; the Sessions view owns the live stream).

export function OverviewPage() {
  const [stats, setStats] = useState<ActivityStats | null>(null)
  const [code, setCode] = useState<CodeStatus | null>(null)
  const [pipeline, setPipeline] = useState<PipelineState | null>(null)
  const [pipelineError, setPipelineError] = useState(false)
  const [events, setEvents] = useState<ActivityEvent[]>([])
  const [eventsError, setEventsError] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)

    void (async () => {
      const [statsR, codeR, pipelineR, eventsR] = await Promise.allSettled([
        fetchActivityStats(),
        fetchCodeStatus(),
        fetchPipelineState(),
        fetchActivityEvents(10),
      ])

      if (cancelled) return

      setStats(statsR.status === 'fulfilled' ? statsR.value : null)
      setCode(codeR.status === 'fulfilled' ? codeR.value : null)
      if (pipelineR.status === 'fulfilled' && pipelineR.value) setPipeline(pipelineR.value)
      else setPipelineError(true)
      if (eventsR.status === 'fulfilled') setEvents(eventsR.value.events ?? [])
      else setEventsError(true)
      setLoading(false)
    })()

    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 border-b border-edge px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Overview</h1>
        <span className="text-[12px] text-ink-5">project health at a glance</span>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <KpiCard
            testId="kpi-active-sessions"
            label="Active Sessions"
            value={stats ? String(stats.activeSessions) : '—'}
            valueClass="text-ink"
            sublabel="sessions"
          />
          <KpiCard
            testId="kpi-total-events"
            label="Total Events"
            value={stats ? String(stats.total) : '—'}
            valueClass="text-ink"
            sublabel="events"
          />
          <CodeIndexCard code={code} loading={loading && !code} />
          <PipelinePhaseCard pipeline={pipeline} error={pipelineError} loading={loading && !pipeline && !pipelineError} />
        </div>

        <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <ActivityPanel events={events} error={eventsError} loading={loading && events.length === 0 && !eventsError} />
          </div>
          <div className="lg:col-span-1">
            <PipelineStatusPanel pipeline={pipeline} error={pipelineError} loading={loading && !pipeline && !pipelineError} />
          </div>
        </div>
      </div>
    </div>
  )
}

function KpiCard({
  testId,
  label,
  value,
  valueClass,
  sublabel,
}: {
  testId: string
  label: string
  value: string
  valueClass: string
  sublabel: string
}) {
  return (
    <div className="flex flex-col gap-2 rounded border border-edge bg-card p-4">
      <span data-testid={`${testId}-label`} className="text-[10px] font-medium uppercase tracking-[0.14em] text-ink-5">
        {label}
      </span>
      <div className="flex items-baseline gap-2">
        <span data-testid={`${testId}-value`} className={`font-mono text-3xl font-semibold ${valueClass}`}>
          {value}
        </span>
        <span className="text-[11px] text-ink-6">{sublabel}</span>
      </div>
    </div>
  )
}

function CodeIndexCard({ code, loading }: { code: CodeStatus | null; loading: boolean }) {
  let text = '—'
  let cls = 'text-ink-4'
  if (loading) {
    text = '…'
    cls = 'text-ink-5'
  } else if (code === null) {
    text = 'not indexed'
    cls = 'text-danger'
  } else if (code.stale || code.file_count === 0) {
    text = code.file_count === 0 ? 'not indexed' : 'stale'
    cls = code.file_count === 0 ? 'text-danger' : 'text-warn'
  } else {
    text = 'fresh'
    cls = 'text-accent'
  }
  const sub = code && code.file_count > 0 ? `${code.file_count} files` : ''
  return (
    <div className="flex flex-col gap-2 rounded border border-edge bg-card p-4">
      <span data-testid="kpi-code-index-label" className="text-[10px] font-medium uppercase tracking-[0.14em] text-ink-5">
        Code Index
      </span>
      <div className="flex items-baseline gap-2">
        <span data-testid="kpi-code-index-value" className={`font-mono text-3xl font-semibold ${cls}`}>
          {text}
        </span>
        {sub && <span className="text-[11px] text-ink-6">{sub}</span>}
      </div>
    </div>
  )
}

function PipelinePhaseCard({
  pipeline,
  error,
  loading,
}: {
  pipeline: PipelineState | null
  error: boolean
  loading: boolean
}) {
  let text = '—'
  if (loading) text = '…'
  else if (error || pipeline === null) text = 'unknown'
  else {
    text = pipeline.current_change ? `${pipeline.current_phase} · ${pipeline.current_change}` : pipeline.current_phase || 'idle'
  }
  return (
    <div className="flex flex-col gap-2 rounded border border-edge bg-card p-4">
      <span data-testid="kpi-pipeline-phase-label" className="text-[10px] font-medium uppercase tracking-[0.14em] text-ink-5">
        Pipeline Phase
      </span>
      <div className="flex items-baseline gap-2">
        <span data-testid="kpi-pipeline-phase-value" className="min-w-0 truncate font-mono text-3xl font-semibold text-ink">
          {text}
        </span>
      </div>
    </div>
  )
}

function Panel({ title, testId, children }: { title: string; testId?: string; children: React.ReactNode }) {
  return (
    <div className="flex h-full flex-col rounded border border-edge bg-card">
      <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
        <h2 className="text-[12px] font-semibold uppercase tracking-[0.1em] text-ink-4">{title}</h2>
      </div>
      <div className="flex min-h-0 flex-1 flex-col" data-testid={testId}>
        {children}
      </div>
    </div>
  )
}

function ActivityPanel({ events, error, loading }: { events: ActivityEvent[]; error: boolean; loading: boolean }) {
  return (
    <Panel title="Recent Activity">
      {loading && <PanelState text="Loading…" />}
      {!loading && error && <PanelState text="Activity feed unavailable." />}
      {!loading && !error && events.length === 0 && <PanelState text="No recent activity." />}
      {!loading && !error && events.length > 0 && (
        <ul className="divide-y divide-edge-soft">
          {events.map((e) => (
            <li key={e.id} className="flex items-start gap-3 px-4 py-2">
              <span className="mt-0.5 shrink-0 font-mono text-[11px] text-ink-6">{formatTs(e.ts)}</span>
              <span className="mt-0.5 shrink-0 rounded bg-edge/40 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-ink-4">
                {e.type}
              </span>
              <span className="min-w-0 flex-1 truncate text-[13px] text-ink-2">{e.summary}</span>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  )
}

function PipelineStatusPanel({ pipeline, error, loading }: { pipeline: PipelineState | null; error: boolean; loading: boolean }) {
  return (
    <Panel title="Pipeline Status" testId="pipeline-status">
      {loading && <PanelState text="Loading…" />}
      {!loading && (error || pipeline === null) && <PanelState text="Pipeline state unavailable." />}
      {!loading && !error && pipeline !== null && (
        <div className="flex flex-col gap-2 px-4 py-3 text-[13px]">
          <Row label="phase" value={pipeline.current_phase || '—'} />
          <Row label="change" value={pipeline.current_change || '—'} />
          <Row label="status" value={pipeline.status || '—'} />
          <Row label="completed" value={String(pipeline.completed_changes)} valueTestId="pipeline-completed-value" />
        </div>
      )}
    </Panel>
  )
}

function Row({ label, value, valueTestId }: { label: string; value: string; valueTestId?: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-[11px] uppercase tracking-[0.1em] text-ink-6">{label}</span>
      <span data-testid={valueTestId} className="truncate font-mono text-ink-2">
        {value}
      </span>
    </div>
  )
}

function PanelState({ text }: { text: string }) {
  return <div className="flex flex-1 items-center justify-center px-6 py-6 text-center text-[13px] text-ink-5">{text}</div>
}

function formatTs(ts: string): string {
  if (!ts) return ''
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return ts
  return d.toLocaleString(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}
