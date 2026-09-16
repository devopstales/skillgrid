import { useLocation } from '@tanstack/react-router'
import { Stub } from './Stub'

export function StubPage({ view, phase }: { view: string; phase: string }) {
  const location = useLocation()
  return (
    <div className="flex h-full flex-col gap-3 p-8">
      <div className="rounded-lg border border-edge bg-card p-6">
        <div className="text-xs font-medium uppercase tracking-widest text-zinc-500">
          {location.pathname}
        </div>
        <h1 className="mt-2 text-2xl font-semibold text-zinc-100">{view}</h1>
        <p className="mt-3 text-sm text-zinc-400">{phase}</p>
        <button
          type="button"
          disabled
          className="mt-5 cursor-not-allowed rounded-md bg-accent/20 px-4 py-2 text-sm font-medium text-zinc-500"
        >
          Open {view}
        </button>
      </div>
      <Stub title={`${view} is not built yet`} note={phase} />
    </div>
  )
}
