import { useEffect, useMemo, useRef, useState } from 'react'
import DOMPurify from 'dompurify'

// mermaid is loaded lazily (dynamic import) so its ~700 kB bundle is only
// fetched when a diagram actually renders, keeping the initial app payload
// small. The promise is memoized at module scope so the lib is initialized
// exactly once across all blocks.
let mermaidPromise: Promise<typeof import('mermaid').default> | null = null
function loadMermaid(): Promise<typeof import('mermaid').default> {
  if (!mermaidPromise) {
    mermaidPromise = import('mermaid').then((m) => {
      m.default.initialize({
        startOnLoad: false,
        securityLevel: 'strict', // forbids raw HTML inside node labels
        theme: 'dark',
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      })
      return m.default
    })
  }
  return mermaidPromise
}

// FNV-1a — a cheap, dependency-free content hash for the SVG cache key.
function hash(s: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return (h >>> 0).toString(16)
}

const svgCache = new Map<string, string>()
let idSeq = 0

export function MermaidBlock({ code, securityLevel }: { code: string; securityLevel?: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const [error, setError] = useState('')
  const key = useMemo(() => hash(code), [code])
  const cached = svgCache.get(key)
  void securityLevel // reserved for future per-block overrides

  useEffect(() => {
    let cancelled = false
    async function render() {
      try {
        let svg: string
        const hit = svgCache.get(key)
        if (hit) {
          svg = hit
        } else {
          const mermaid = await loadMermaid()
          const id = `mmd-${++idSeq}`
          const out = await mermaid.render(id, code)
          svg = out.svg
          svgCache.set(key, svg)
        }
        if (cancelled || !ref.current) return
        // sanitize: only svg is allowed; strip script/event handlers/attrs.
        const clean = DOMPurify.sanitize(svg, {
          USE_PROFILES: { svg: true, svgFilters: true },
        })
        ref.current.innerHTML = clean
        setError('')
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'mermaid render failed')
      }
    }
    void render()
    return () => {
      cancelled = true
    }
  }, [code, key])

  return (
    <div className="my-3 overflow-x-auto rounded border border-edge bg-black/30 p-3">
      {cached === undefined && !error && (
        <span className="text-[12px] text-ink-5">rendering diagram…</span>
      )}
      {error && (
        <div className="space-y-2">
          <pre className="whitespace-pre-wrap text-[12px] text-danger">{error}</pre>
          <pre className="whitespace-pre-wrap text-[12px] text-ink-4">{code}</pre>
        </div>
      )}
      <div ref={ref} className="flex justify-center [&>svg]:max-w-full" />
    </div>
  )
}
