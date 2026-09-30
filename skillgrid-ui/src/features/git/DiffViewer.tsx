// DiffViewer — renders a unified diff with per-line +/- coloring. A simple,
// dependency-free parser: color added (green), removed (red), hunk headers
// (cyan), and file headers (bold). No syntax highlighting beyond that.
export function DiffViewer({ diff }: { diff: string }) {
  if (!diff.trim()) {
    return <p className="text-[13px] text-ink-5">No diff for this commit.</p>
  }
  const lines = diff.split('\n')
  return (
    <pre className="overflow-x-auto rounded border border-edge bg-inset p-3 text-[12px] leading-relaxed">
      {lines.map((line, i) => {
        let cls = 'text-ink-4'
        let prefix = ' '
        if (line.startsWith('+++') || line.startsWith('---')) cls = 'text-ink-5'
        else if (line.startsWith('diff ') || line.startsWith('index ')) cls = 'text-ink-6'
        else if (line.startsWith('@@')) {
          cls = 'text-info'
          prefix = ''
        } else if (line.startsWith('+')) {
          cls = 'text-accent bg-accent/10'
          prefix = '+'
        } else if (line.startsWith('-')) {
          cls = 'text-danger bg-danger/10'
          prefix = '-'
        } else {
          prefix = ' '
        }
        return (
          <div key={i} className={`whitespace-pre ${cls}`}>
            <span className="inline-block w-3 select-none text-ink-6">{prefix}</span>
            {line}
          </div>
        )
      })}
    </pre>
  )
}
