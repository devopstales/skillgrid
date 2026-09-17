import { useEffect, useMemo, useRef, useState } from 'react'
import mermaid from 'mermaid'
import DOMPurify from 'dompurify'

// mermaid is configured once with the safe defaults. securityLevel 'strict'
// (the server-declared default) forbids raw HTML inside node labels, and we
// additionally DOMPurify-sanitize the rendered SVG before injecting it.
let configured = false
function ensureConfigured(securityLevel: string) {
  if (configured) return
  configured = true
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: securityLevel === 'strict' ? 'strict' : 'strict',
    theme: 'dark',
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
  })
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
  ensureConfigured(securityLevel ?? 'strict')
  const ref = useRef<HTMLDivElement>(null)
  const [error, setError] = useState('')
  const key = useMemo(() => hash(code), [code])
  const cached = svgCache.get(key)

  useEffect(() => {
    let cancelled = false
    async function render() {
      try {
        let svg: string
        const hit = svgCache.get(key)
        if (hit) {
          svg = hit
        } else {
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
    <div className="my-3 overflow-x-auto rounded-md border border-edge bg-black/30 p-3">
      {cached === undefined && !error && (
        <span className="text-xs text-zinc-500">rendering diagram…</span>
      )}
      {error && (
        <div className="space-y-2">
          <pre className="whitespace-pre-wrap text-xs text-red-300">{error}</pre>
          <pre className="whitespace-pre-wrap text-xs text-zinc-400">{code}</pre>
        </div>
      )}
      <div ref={ref} className="flex justify-center [&>svg]:max-w-full" />
    </div>
  )
}
