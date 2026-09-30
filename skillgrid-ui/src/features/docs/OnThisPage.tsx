import { useEffect, useRef, useState } from 'react'
import { slugify } from './MarkdownView'

// TOC entries extracted from markdown headings (### / ## / #).
export interface TocEntry {
  id: string
  text: string
  level: number
}

export function extractToc(body: string): TocEntry[] {
  const out: TocEntry[] = []
  for (const line of body.split('\n')) {
    const m = /^(#{1,3})\s+(.*)$/.exec(line)
    if (m) {
      const text = m[2].trim().replace(/[*_`]/g, '')
      out.push({ id: slugify(text), text, level: m[1].length })
    }
  }
  return out
}

export function OnThisPage({ entries }: { entries: TocEntry[] }) {
  const [active, setActive] = useState('')
  const raf = useRef(0)

  useEffect(() => {
    if (entries.length === 0) return
    function onScroll() {
      cancelAnimationFrame(raf.current)
      raf.current = requestAnimationFrame(() => {
        let current = ''
        for (const e of entries) {
          const el = document.getElementById(e.id)
          if (!el) continue
          if (el.getBoundingClientRect().top <= 120) current = e.id
        }
        setActive(current)
      })
    }
    window.addEventListener('scroll', onScroll, { passive: true })
    onScroll()
    return () => {
      window.removeEventListener('scroll', onScroll)
      cancelAnimationFrame(raf.current)
    }
  }, [entries])

  if (entries.length === 0) return null

  return (
    <nav className="w-56 shrink-0 border-l border-edge bg-card">
      <div className="sticky top-0 px-3 py-3">
        <h4 className="mb-2 text-[11px] font-semibold uppercase tracking-[0.1em] text-ink-5">
          On this page
        </h4>
        <ul className="space-y-1">
          {entries.map((e) => (
            <li key={e.id} style={{ paddingLeft: (e.level - 1) * 10 }}>
              <a
                href={`#${e.id}`}
                onClick={(ev) => {
                  ev.preventDefault()
                  document.getElementById(e.id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                }}
                className={[
                  'block truncate rounded px-1.5 py-0.5 text-[12px] transition-colors',
                  active === e.id
                    ? 'bg-accent/15 text-accent'
                    : 'text-ink-5 hover:text-ink-3',
                ].join(' ')}
              >
                {e.text}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
