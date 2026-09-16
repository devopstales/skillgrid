import { useEffect, useState } from 'react'
import {
  fetchProjects,
  readStoredProject,
  storeProject,
  type ProjectsState,
} from '../../lib/projects'

export function ProjectSelector() {
  const [state, setState] = useState<ProjectsState>({
    projects: [],
    status: 'loading',
  })
  const [selected, setSelected] = useState<string | null>(() => readStoredProject())

  useEffect(() => {
    let cancelled = false
    fetchProjects()
      .then((projects) => {
        if (cancelled) return
        setState({ projects, status: 'ready' })
        const stored = readStoredProject()
        if (stored !== null && !projects.includes(stored)) {
          setSelected(projects[0] ?? null)
        } else if (stored === null && projects.length > 0) {
          setSelected(projects[0] ?? null)
        }
      })
      .catch(() => {
        if (!cancelled) setState({ projects: [], status: 'unavailable' })
      })
    return () => {
      cancelled = true
    }
  }, [])

  const disabled = state.status !== 'ready'
  const placeholder =
    state.status === 'unavailable' ? 'no server' : 'loading projects…'

  return (
    <label className="flex items-center gap-2 text-xs text-zinc-500">
      <span className="hidden sm:inline">Project</span>
      <select
        aria-label="Project"
        className="h-8 max-w-56 rounded-md border border-edge bg-bg px-2 text-sm text-zinc-200 outline-none focus:border-accent disabled:cursor-not-allowed disabled:opacity-50"
        value={selected ?? ''}
        disabled={disabled}
        onChange={(event) => {
          const value = event.target.value
          setSelected(value)
          storeProject(value)
        }}
      >
        {disabled ? (
          <option value="">{placeholder}</option>
        ) : (
          state.projects.map((project) => (
            <option key={project} value={project}>
              {project}
            </option>
          ))
        )}
      </select>
    </label>
  )
}
