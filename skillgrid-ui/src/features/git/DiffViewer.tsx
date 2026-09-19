// DiffViewer — renders a unified diff with per-line +/- coloring. A simple,
// dependency-free parser: color added (green), removed (red), hunk headers
// (cyan), and file headers (bold). No syntax highlighting beyond that.
export function DiffViewer({ diff }: { diff: string }) {
  if (!diff.trim()) {
    return <p className="text-sm text-zinc-500">No diff for this commit.</p>
  }
  const lines = diff.split('\n')
  return (
    <pre className="overflow-x-auto rounded-md border border-edge bg-[#0a0a0d] p-3 text-xs leading-relaxed">
      {lines.map((line, i) => {
        let cls = 'text-zinc-400'
        let prefix = ' '
        if (line.startsWith('+++') || line.startsWith('---')) cls = 'text-zinc-500'
        else if (line.startsWith('diff ') || line.startsWith('index ')) cls = 'text-zinc-600'
        else if (line.startsWith('@@')) {
          cls = 'text-sky-400'
          prefix = ''
        } else if (line.startsWith('+')) {
          cls = 'text-emerald-400 bg-emerald-950/30'
          prefix = '+'
        } else if (line.startsWith('-')) {
          cls = 'text-red-400 bg-red-950/30'
          prefix = '-'
        } else {
          prefix = ' '
        }
        return (
          <div key={i} className={`whitespace-pre ${cls}`}>
            <span className="inline-block w-3 select-none text-zinc-600">{prefix}</span>
            {line}
          </div>
        )
      })}
    </pre>
  )
}
