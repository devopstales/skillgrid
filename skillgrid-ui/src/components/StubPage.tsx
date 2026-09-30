import { useLocation } from '@tanstack/react-router'
import { Stub } from './Stub'

export function StubPage({ view, phase }: { view: string; phase: string }) {
  const location = useLocation()
  return (
    <div className="flex h-full flex-col gap-3 p-8 font-mono">
      <div className="rounded-md border border-edge bg-card p-6">
        <div className="text-[11px] font-medium uppercase tracking-[0.14em] text-ink-6">
          {location.pathname}
        </div>
        <h1 className="mt-2 text-[22px] font-semibold text-ink">{view}</h1>
        <p className="mt-3 text-[13px] text-ink-4">{phase}</p>
        <button
          type="button"
          disabled
          className="mt-5 cursor-not-allowed rounded border border-accent-ink bg-accent/10 px-4 py-2 text-[13px] font-medium text-ink-5"
        >
          Open {view}
        </button>
      </div>
      <Stub title={`${view} is not built yet`} note={phase} />
    </div>
  )
}
