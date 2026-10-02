import { useEffect, useState } from 'react'
import { apiGet } from '../../lib/api'
import { EmptyState, ErrorState, LoadingState, PageHeader } from '../../components/ui/Badges'

interface Prototype {
  name: string
  date?: string
  venue?: string
  hypothesis?: string
  change?: string
  files?: string[]
}

// previewHref is the sandboxed file route for a prototype's generated
// index.html. Opening it in a new tab loads the preview (and its relative assets).
function previewHref(name: string): string {
  return `/prototypes/${encodeURIComponent(name)}/index.html`
}

export function PrototypesPage() {
  const [prototypes, setPrototypes] = useState<Prototype[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    apiGet<{ prototypes?: Prototype[] }>('/prototypes', {}, { project: false })
      .then((d) => setPrototypes(d.prototypes || []))
      .catch((err: Error) => setError(err.message))
  }, [])

  if (error) {
    return (
      <div className="p-6">
        <ErrorState error={error} />
      </div>
    )
  }
  if (!prototypes) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  return (
    <div className="p-6">
      <PageHeader
        title="Prototypes"
        subtitle={`${prototypes.length} throwaway feasibility experiments — verdicts in prototype.md`}
      />
      {prototypes.length > 0 ? (
        <div className="space-y-2">
          {prototypes.map((s) => (
            <div key={s.name} className="kpi-card">
              <div className="flex items-start justify-between gap-3">
                {s.change ? (
                  <a
                    href={`/plans?change=${encodeURIComponent(s.change)}`}
                    className="font-mono text-sm text-accent hover:underline"
                  >
                    {s.name}
                  </a>
                ) : (
                  <span className="font-mono text-sm text-ink">{s.name}</span>
                )}
                {s.date && <span className="shrink-0 text-xs text-ink-4">{s.date}</span>}
              </div>
              {s.venue && <div className="mt-1 text-xs text-accent">{s.venue}</div>}
              {s.hypothesis && (
                <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-ink-3">{s.hypothesis}</p>
              )}
              {s.files && s.files.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {s.files.map((f) =>
                    f === 'index.html' ? (
                      <a
                        key={f}
                        href={previewHref(s.name)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="rounded bg-accent/15 px-1.5 py-0.5 font-mono text-[10px] text-accent hover:underline"
                      >
                        {f}
                      </a>
                    ) : (
                      <span
                        key={f}
                        className="rounded bg-surface-700 px-1.5 py-0.5 font-mono text-[10px] text-ink-4"
                      >
                        {f}
                      </span>
                    ),
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="kpi-card">
          <EmptyState message="No prototypes found under .skillgrid/prototypes/" />
        </div>
      )}
    </div>
  )
}
