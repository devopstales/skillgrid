// currentProjectName resolves the project the server is running in via
// GET /project/current (derived server-side from the git remote's repo name).
// Read-only: the dashboard shows the active project, it does not switch
// between several.
export async function currentProjectName(): Promise<string> {
  try {
    const res = await fetch('/project/current', {
      headers: { Accept: 'application/json' },
    })
    if (res.ok) {
      const data = (await res.json()) as { project?: unknown }
      if (typeof data.project === 'string' && data.project) {
        return data.project
      }
    }
  } catch {
    // fall through
  }
  const title = document.title.trim()
  return title && title.toLowerCase() !== 'untitled' ? title : 'project'
}
