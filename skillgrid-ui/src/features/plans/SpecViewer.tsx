import { MarkdownView } from '../docs/MarkdownView'

// SpecViewer — renders a spec/plan markdown file, reusing the Docs
// MarkdownView (+ its MermaidBlock for ```mermaid fences). This is the
// "spec viewer (reuses Docs)" requirement.
export function SpecViewer({
  path,
  content,
  onBack,
}: {
  path: string
  content: string
  onBack: () => void
}) {
  return (
    <div className="flex h-full flex-col">
      <div className="mb-4 flex items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded border border-edge px-2.5 py-1 text-[12px] text-ink-4 hover:text-ink-2"
        >
          ← Back
        </button>
        <span className="truncate text-[12px] text-ink-5" title={path}>
          {path}
        </span>
      </div>
      <div className="flex-1 overflow-y-auto rounded-md border border-edge bg-card p-6">
        {content.trim() ? (
          <MarkdownView body={content} />
        ) : (
          <p className="text-[13px] text-ink-5">Empty file.</p>
        )}
      </div>
    </div>
  )
}
