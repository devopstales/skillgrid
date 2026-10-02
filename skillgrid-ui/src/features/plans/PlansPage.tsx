import { useEffect, useState } from 'react'
import {
  fetchPlans,
  fetchPlan,
  fetchSpecContent,
  PlansError,
  type PlanSummary,
  type PlanDetail,
} from './api'
import { ChangeRow, CHANGE_COLS } from './ChangeRow'
import { PlanDetailPanel } from './PlanDetail'
import { SpecViewer } from './SpecViewer'
import { PlanDependencyGraph } from './PlanDependencyGraph'
import { chipKind } from './status'

// PlansPage — the Changes view (prototype 002 variant A "changes" page):
// a dense hairline row list on top (name · phase · progress · chip) and a
// detail pane below it. Every row is clickable and selects the change;
// the detail pane shows its ledger + files, or the clickable pipeline overview
// when nothing is selected. Opening a spec file swaps the detail pane for the
// SpecViewer (list stays visible).
export function PlansPage() {
  const [plans, setPlans] = useState<PlanSummary[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<string | null>(null)
  const [detail, setDetail] = useState<PlanDetail | null>(null)
  const [detailError, setDetailError] = useState('')
  const [specPath, setSpecPath] = useState<string | null>(null)
  const [specContent, setSpecContent] = useState<string | null>(null)
  const [specError, setSpecError] = useState('')

  // Load the plan list. On first load, honor a ?change= deep-link (from the
  // Docs "Read in Docs →" cross-link) to auto-select that plan.
  useEffect(() => {
    let cancelled = false
    setError('')
    setLoading(true)
    fetchPlans()
      .then((r) => {
        if (cancelled) return
        setPlans(r.plans ?? [])
        const change = new URLSearchParams(window.location.search).get('change')
        if (change) setSelected(change)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof PlansError ? e.message : 'failed to load plans')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Load detail when a plan is selected.
  useEffect(() => {
    if (!selected) {
      setDetail(null)
      return
    }
    let cancelled = false
    setDetailError('')
    setDetail(null)
    fetchPlan(selected)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((e) => {
        if (!cancelled) setDetailError(e instanceof PlansError ? e.message : 'failed to load plan')
      })
    return () => {
      cancelled = true
    }
  }, [selected])

  // Load spec content when a spec file is opened.
  useEffect(() => {
    if (!specPath) {
      setSpecContent(null)
      return
    }
    let cancelled = false
    setSpecError('')
    setSpecContent(null)
    fetchSpecContent(specPath)
      .then((s) => {
        if (!cancelled) setSpecContent(s.content)
      })
      .catch((e) => {
        if (!cancelled) setSpecError(e instanceof PlansError ? e.message : 'failed to load spec')
      })
    return () => {
      cancelled = true
    }
  }, [specPath])

  // select a change: update state + the ?change= deep-link, close any open spec.
  const open = (name: string) => {
    setSelected(name)
    setSpecPath(null)
    const url = new URL(window.location.href)
    url.searchParams.set('change', name)
    window.history.replaceState({}, '', url)
  }

  const activeCount = plans.filter((p) => chipKind(p.status) === 'active').length
  const shippedCount = plans.filter((p) => chipKind(p.status) === 'pass').length

  return (
    <div className="flex h-full flex-col">
      {/* Topbar: title + counts */}
      <div className="flex items-center gap-3 border-b border-edge px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Changes</h1>
        <span className="text-[12px] text-ink-5">SDD plans &amp; specs from .skillgrid/</span>
        <span className="flex-1" />
        <Pill>
          <span className={`h-[7px] w-[7px] rounded-full ${activeCount > 0 ? 'bg-warn' : 'bg-ink-6'}`} />
          {activeCount} active
        </Pill>
        <Pill>
          <span className="h-[7px] w-[7px] rounded-full bg-ok" />
          {shippedCount} shipped
        </Pill>
      </div>

      <div className="flex min-h-0 flex-1 flex-col">
        {/* List pane (top): sizes to its rows up to ~40% of the height, then scrolls */}
        <div className="flex max-h-[40%] min-h-[120px] shrink-0 flex-col border-b border-edge bg-card">
          <div className="flex items-center justify-between border-b border-edge px-3 py-2.5">
            <h2 className="text-[11px] font-semibold uppercase tracking-[0.1em] text-ink-5">changes</h2>
            <span className="font-mono text-[11px] tabular-nums text-ink-6">{plans.length}</span>
          </div>
          {error && (
            <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
              {error}
            </div>
          )}
          {loading && (
            <div className="flex flex-1 items-center justify-center text-[13px] text-ink-5">Loading…</div>
          )}
          {!loading && !error && plans.length === 0 && (
            <div className="flex flex-1 items-center justify-center px-6 text-center text-[13px] text-ink-5">
              No plans found in .skillgrid/specs/.
            </div>
          )}
          {!loading && plans.length > 0 && (
            <div className="min-h-0 flex-1 overflow-y-auto">
              <div
                className={`grid gap-2.5 border-b border-edge px-3 py-1.5 text-[10px] font-medium uppercase tracking-[0.1em] text-ink-6 ${CHANGE_COLS}`}
              >
                <span>name</span>
                <span>phase</span>
                <span className="text-right">tasks</span>
                <span className="text-right">gate</span>
              </div>
              <div role="group" aria-label="Changes">
                {plans.map((p) => (
                  <ChangeRow key={p.name} plan={p} active={selected === p.name} onOpen={open} />
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Detail pane (bottom) */}
        <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
          {specPath ? (
            <div className="flex min-h-0 flex-1 flex-col px-6 py-5">
              {specError ? (
                <div className="rounded border border-danger-ink bg-danger/10 p-4 text-[13px] text-danger">
                  {specError}
                </div>
              ) : specContent === null ? (
                <div className="p-8 text-center text-[13px] text-ink-5">Loading…</div>
              ) : (
                <SpecViewer path={specPath} content={specContent} onBack={() => setSpecPath(null)} />
              )}
            </div>
          ) : loading ? (
            <div className="flex flex-1 items-center justify-center text-[13px] text-ink-5">Loading…</div>
          ) : !selected ? (
            <div className="px-6 py-5">
              <div className="mb-3 flex flex-col gap-0.5">
                <h2 className="text-[11px] font-semibold uppercase tracking-[0.1em] text-ink-5">pipeline</h2>
                <span className="text-[12px] text-ink-6">select a change to see its ledger and files</span>
              </div>
              <PlanDependencyGraph plans={plans} selected={selected} onOpen={open} />
            </div>
          ) : detailError ? (
            <div className="flex flex-1 items-center justify-center p-8">
              <span className="rounded border border-danger-ink bg-danger/10 px-4 py-2 text-[13px] text-danger">
                {detailError}
              </span>
            </div>
          ) : detail ? (
            <div className="px-6 py-5">
              <PlanDetailPanel plan={detail} onOpenSpec={(path) => setSpecPath(path)} />
            </div>
          ) : (
            <div className="flex flex-1 items-center justify-center text-[13px] text-ink-5">Loading…</div>
          )}
        </div>
      </div>
    </div>
  )
}

function Pill({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-[4px] border border-edge-soft bg-inset px-2 py-[3px] font-mono text-[11px] tabular-nums text-ink-4">
      {children}
    </span>
  )
}
