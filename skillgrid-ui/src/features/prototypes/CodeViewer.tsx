import { useState } from 'react'

// CodeViewer — shows a prototype's HTML source in a scrollable, monospace
// panel. A single source is shown (the prototype is a self-contained HTML file
// with inline CSS/JS), so there's no separate CSS/JS file to split.
export function CodeViewer({ html, id }: { html: string; id: string }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(html)
      setCopied(true)
      setTimeout(() => setCopied(false), 1200)
    } catch {
      /* clipboard unavailable */
    }
  }
  return (
    <div className="rounded-lg border border-edge bg-[#0a0a0d]">
      <div className="flex items-center justify-between border-b border-edge px-3 py-2">
        <span className="truncate text-xs text-zinc-500" title={id}>
          {id}
        </span>
        <div className="flex items-center gap-2">
          <a
            href={`/prototypes/${id}`}
            download={id.split('/').pop() ?? 'prototype.html'}
            className="rounded-md border border-edge px-2 py-1 text-xs text-zinc-300 hover:text-accent"
          >
            Download
          </a>
          <button
            type="button"
            onClick={copy}
            className="rounded-md border border-edge px-2 py-1 text-xs text-zinc-300 hover:text-accent"
          >
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
      </div>
      <pre className="max-h-[640px] overflow-auto p-3 text-xs leading-relaxed text-zinc-300">
        {html}
      </pre>
    </div>
  )
}
