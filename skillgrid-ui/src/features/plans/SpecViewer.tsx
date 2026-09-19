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
          className="rounded-md border border-edge px-2.5 py-1 text-xs text-zinc-400 hover:text-zinc-200"
        >
          ← Back
        </button>
        <span className="truncate text-xs text-zinc-500" title={path}>
          {path}
        </span>
      </div>
      <div className="flex-1 overflow-y-auto rounded-lg border border-edge bg-card p-6">
        {content.trim() ? (
          <MarkdownView body={content} />
        ) : (
          <p className="text-sm text-zinc-500">Empty file.</p>
        )}
      </div>
    </div>
  )
}
