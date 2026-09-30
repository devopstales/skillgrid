import { useEffect, useState } from 'react'
import { fetchPrototypes, fetchPrototypeHTML, PrototypesError } from './api'
import { SandboxPreview, DEVICE_WIDTHS, type Device } from './SandboxPreview'
import { CodeViewer } from './CodeViewer'

// PrototypesPage — the .stitch/ design-prototype gallery: a left list of
// prototypes, a right preview (sandboxed iframe with a device-viewport toggle)
// or a code view. Replaces the Phase 1 stub.
export function PrototypesPage() {
  const [prototypes, setPrototypes] = useState<string[]>([])
  const [error, setError] = useState('')
  const [selected, setSelected] = useState<string | null>(null)
  const [html, setHtml] = useState('')
  const [loading, setLoading] = useState(false)
  const [device, setDevice] = useState<Device>('desktop')
  const [view, setView] = useState<'preview' | 'code'>('preview')

  // Load the gallery list; auto-select the first prototype.
  useEffect(() => {
    let cancelled = false
    setError('')
    fetchPrototypes()
      .then((r) => {
        if (cancelled) return
        setPrototypes(r.prototypes ?? [])
        if (r.prototypes?.length) setSelected(r.prototypes[0])
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof PrototypesError ? e.message : 'failed to load prototypes')
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Load the selected prototype's HTML.
  useEffect(() => {
    if (!selected) {
      setHtml('')
      return
    }
    let cancelled = false
    setLoading(true)
    setHtml('')
    fetchPrototypeHTML(selected)
      .then((h) => {
        if (!cancelled) setHtml(h)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof PrototypesError ? e.message : 'failed to load prototype')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [selected])

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-8">
      <div>
        <h1 className="text-[14px] font-semibold text-ink">Prototypes</h1>
        <p className="mt-1 text-[13px] text-ink-5">Sandboxed .stitch/ design-prototype gallery</p>
      </div>

      {error ? (
        <div className="rounded border border-danger-ink bg-danger/10 p-4 text-[13px] text-danger">
          {error}
        </div>
      ) : prototypes.length === 0 ? (
        <div className="rounded border border-edge bg-card p-8 text-center text-[13px] text-ink-5">
          No prototypes found in .stitch/ (add self-contained .html files to the gallery).
        </div>
      ) : (
        <div className="grid flex-1 gap-6 lg:grid-cols-[240px_1fr]">
          {/* gallery list */}
          <aside className="h-fit rounded-md border border-edge bg-card p-2">
            <h2 className="px-2 py-1.5 text-[12px] font-medium uppercase tracking-[0.1em] text-ink-5">
              Gallery
            </h2>
            <ul className="space-y-1">
              {prototypes.map((p) => (
                <li key={p}>
                  <button
                    type="button"
                    onClick={() => setSelected(p)}
                    className={`w-full truncate rounded px-3 py-2 text-left text-[12px] ${
                      p === selected ? 'bg-accent/15 text-accent' : 'text-ink-3 hover:text-accent'
                    }`}
                    title={p}
                  >
                    {p}
                  </button>
                </li>
              ))}
            </ul>
          </aside>

          {/* preview / code */}
          <section className="min-w-0 space-y-4">
            <div className="flex flex-wrap items-center gap-2">
              {/* Review B5: tablist/tab semantics so the active view is announced. */}
              <div className="flex rounded border border-edge p-0.5" role="tablist" aria-label="Preview or code">
                {(['preview', 'code'] as const).map((v) => (
                  <button
                    key={v}
                    type="button"
                    role="tab"
                    aria-selected={view === v}
                    onClick={() => setView(v)}
                    className={`rounded px-3 py-1 text-[12px] ${
                      view === v ? 'bg-accent text-bg' : 'text-ink-4 hover:text-ink-2'
                    }`}
                  >
                    {v}
                  </button>
                ))}
              </div>
              {view === 'preview' && (
                <div className="flex rounded border border-edge p-0.5" role="tablist" aria-label="Device viewport">
                  {(Object.keys(DEVICE_WIDTHS) as Device[]).map((d) => (
                    <button
                      key={d}
                      type="button"
                      role="tab"
                      aria-selected={device === d}
                      onClick={() => setDevice(d)}
                      className={`rounded px-3 py-1 text-[12px] ${
                        device === d ? 'bg-accent text-bg' : 'text-ink-4 hover:text-ink-2'
                      }`}
                    >
                      {d}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {loading ? (
              <p className="p-8 text-center text-[13px] text-ink-5">Loading prototype…</p>
            ) : view === 'preview' ? (
              <SandboxPreview html={html} device={device} />
            ) : (
              <CodeViewer html={html} id={selected ?? ''} />
            )}
          </section>
        </div>
      )}
    </div>
  )
}
