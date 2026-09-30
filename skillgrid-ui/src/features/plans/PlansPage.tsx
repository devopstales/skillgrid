import { useEffect, useState } from 'react'
import {
  fetchPlans,
  fetchPlan,
  fetchSpecContent,
  PlansError,
  type PlanSummary,
  type PlanDetail,
} from './api'
import { PlanCard } from './PlanCard'
import { PlanDetailPanel } from './PlanDetail'
import { SpecViewer } from './SpecViewer'
import { PlanDependencyGraph } from './PlanDependencyGraph'

export function PlansPage() {
  const [plans, setPlans] = useState<PlanSummary[]>([])
  const [error, setError] = useState('')
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
    fetchPlans()
      .then((r) => {
        if (cancelled) return
        setPlans(r.plans ?? [])
        const change = new URLSearchParams(window.location.search).get('change')
        if (change && r.plans?.some((p) => p.name === change)) {
          setSelected(change)
        }
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof PlansError ? e.message : 'failed to load plans')
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

  // Spec viewer takes over the whole view when a spec file is open.
  if (specPath) {
    return (
      <div className="h-full overflow-y-auto p-8">
        <div className="mx-auto max-w-4xl">
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
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-8">
      <div>
        <h1 className="text-[14px] font-semibold text-ink">Plans</h1>
        <p className="mt-1 text-[13px] text-ink-5">SDD plans &amp; specs from .skillgrid/</p>
      </div>

      {error ? (
        <div className="rounded border border-danger-ink bg-danger/10 p-4 text-[13px] text-danger">
          {error}
        </div>
      ) : (
        <div className="grid flex-1 gap-6 lg:grid-cols-[1fr_360px]">
          <div className="space-y-6">
            <section>
              <h2 className="mb-3 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
                All plans
              </h2>
              {plans.length === 0 ? (
                <div className="rounded border border-edge bg-card p-8 text-center text-[13px] text-ink-5">
                  No plans found in .skillgrid/specs/.
                </div>
              ) : (
                <div className="grid gap-3 sm:grid-cols-2">
                  {plans.map((p) => (
                    <PlanCard
                      key={p.name}
                      plan={p}
                      onOpen={(name) => {
                        setSelected(name)
                        const url = new URL(window.location.href)
                        url.searchParams.set('change', name)
                        window.history.replaceState({}, '', url)
                      }}
                    />
                  ))}
                </div>
              )}
            </section>

            <section>
              <h2 className="mb-3 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
                Pipeline
              </h2>
              <div className="rounded-md border border-edge bg-card p-4">
                <PlanDependencyGraph plans={plans} />
              </div>
            </section>
          </div>

          <aside className="h-fit rounded-md border border-edge bg-card p-4">
            <h2 className="mb-3 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
              Selected plan
            </h2>
            {!selected ? (
              <p className="text-[13px] text-ink-5">Select a plan to see its ledger and files.</p>
            ) : detailError ? (
              <div className="rounded border border-danger-ink bg-danger/10 p-3 text-[13px] text-danger">
                {detailError}
              </div>
            ) : detail ? (
              <PlanDetailPanel plan={detail} onOpenSpec={(path) => setSpecPath(path)} />
            ) : (
              <p className="text-[13px] text-ink-5">Loading…</p>
            )}
          </aside>
        </div>
      )}
    </div>
  )
}
