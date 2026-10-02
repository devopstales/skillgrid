import { useEffect, useState } from 'react'
import { apiGet } from '../../lib/api'
import { EmptyState, ErrorState, LoadingState, PageHeader } from '../../components/ui/Badges'

interface Spike {
  name: string
  date?: string
  venue?: string
  hypothesis?: string
  files?: string[]
}

export function SpikesPage() {
  const [spikes, setSpikes] = useState<Spike[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    apiGet<{ spikes?: Spike[] }>('/spikes', {}, { project: false })
      .then((d) => setSpikes(d.spikes || []))
      .catch((err: Error) => setError(err.message))
  }, [])

  if (error) {
    return (
      <div className="p-6">
        <ErrorState error={error} />
      </div>
    )
  }
  if (!spikes) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  return (
    <div className="p-6">
      <PageHeader
        title="Spikes"
        subtitle={`${spikes.length} throwaway feasibility experiments — verdicts in spike.md`}
      />
      {spikes.length > 0 ? (
        <div className="space-y-2">
          {spikes.map((s) => (
            <div key={s.name} className="kpi-card">
              <div className="flex items-start justify-between gap-3">
                <span className="font-mono text-sm text-ink">{s.name}</span>
                {s.date && <span className="shrink-0 text-xs text-ink-4">{s.date}</span>}
              </div>
              {s.venue && <div className="mt-1 text-xs text-accent">{s.venue}</div>}
              {s.hypothesis && (
                <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-ink-3">{s.hypothesis}</p>
              )}
              {s.files && s.files.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {s.files.map((f) => (
                    <span
                      key={f}
                      className="rounded bg-surface-700 px-1.5 py-0.5 font-mono text-[10px] text-ink-4"
                    >
                      {f}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="kpi-card">
          <EmptyState message="No spikes found under .skillgrid/spikes/" />
        </div>
      )}
    </div>
  )
}
