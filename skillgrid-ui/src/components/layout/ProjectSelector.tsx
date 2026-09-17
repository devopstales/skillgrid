import { useEffect, useState } from 'react'
import { currentProjectName } from '../../lib/projects'

// ProjectBadge shows the single project the server is running in (read-only).
// The dashboard does not switch between projects — it reflects the active one.
export function ProjectBadge() {
  const [name, setName] = useState<string>('…')

  useEffect(() => {
    let cancelled = false
    currentProjectName().then((n) => {
      if (!cancelled) setName(n)
    })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <span
      className="flex items-center gap-2 rounded-md border border-edge bg-bg px-2.5 py-1.5 text-xs text-zinc-400"
      title="Current project (read-only)"
    >
      <span className="h-1.5 w-1.5 rounded-full bg-accent" />
      <span className="font-mono text-zinc-300">{name}</span>
    </span>
  )
}

// ProjectSelector is retained as an alias so existing imports keep compiling;
// it now renders the read-only badge.
export const ProjectSelector = ProjectBadge
