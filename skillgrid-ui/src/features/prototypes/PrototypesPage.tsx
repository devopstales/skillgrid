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
        <h1 className="text-2xl font-semibold text-zinc-100">Prototypes</h1>
        <p className="mt-1 text-sm text-zinc-500">Sandboxed .stitch/ design-prototype gallery</p>
      </div>

      {error ? (
        <div className="rounded-md border border-red-900/60 bg-red-950/30 p-4 text-sm text-red-300">
          {error}
        </div>
      ) : prototypes.length === 0 ? (
        <div className="rounded-md border border-edge bg-card p-8 text-center text-sm text-zinc-500">
          No prototypes found in .stitch/ (add self-contained .html files to the gallery).
        </div>
      ) : (
        <div className="grid flex-1 gap-6 lg:grid-cols-[240px_1fr]">
          {/* gallery list */}
          <aside className="h-fit rounded-lg border border-edge bg-card p-2">
            <h2 className="px-2 py-1.5 text-xs font-medium uppercase tracking-widest text-zinc-500">
              Gallery
            </h2>
            <ul className="space-y-1">
              {prototypes.map((p) => (
                <li key={p}>
                  <button
                    type="button"
                    onClick={() => setSelected(p)}
                    className={`w-full truncate rounded-md px-3 py-2 text-left text-xs ${
                      p === selected ? 'bg-accent/15 text-accent' : 'text-zinc-300 hover:text-accent'
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
              <div className="flex rounded-md border border-edge p-0.5" role="tablist" aria-label="Preview or code">
                {(['preview', 'code'] as const).map((v) => (
                  <button
                    key={v}
                    type="button"
                    role="tab"
                    aria-selected={view === v}
                    onClick={() => setView(v)}
                    className={`rounded px-3 py-1 text-xs ${
                      view === v ? 'bg-accent text-white' : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    {v}
                  </button>
                ))}
              </div>
              {view === 'preview' && (
                <div className="flex rounded-md border border-edge p-0.5" role="tablist" aria-label="Device viewport">
                  {(Object.keys(DEVICE_WIDTHS) as Device[]).map((d) => (
                    <button
                      key={d}
                      type="button"
                      role="tab"
                      aria-selected={device === d}
                      onClick={() => setDevice(d)}
                      className={`rounded px-3 py-1 text-xs ${
                        device === d ? 'bg-accent text-white' : 'text-zinc-400 hover:text-zinc-200'
                      }`}
                    >
                      {d}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {loading ? (
              <p className="p-8 text-center text-sm text-zinc-500">Loading prototype…</p>
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
