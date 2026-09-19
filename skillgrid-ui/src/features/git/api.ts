// Shared fetch + type contracts for the Phase 6 Git view. The git bridge
// reads the repo from the server CWD / SKILLGRID_DOCS_CWD, so these are plain
// relative GETs (no ?project=). Not-a-repo → 503 (GitError status 503).

export class GitError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = (await res.json()) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      /* keep statusText */
    }
    throw new GitError(msg, res.status)
  }
  return (await res.json()) as T
}

// ---------------------------------------------------------------------------
// types
// ---------------------------------------------------------------------------

export interface GitFileStat {
  path: string
  additions: number
  deletions: number
}

export interface GitCommit {
  sha: string
  author: string
  date: string
  message: string
  body?: string
  additions: number
  deletions: number
  files?: GitFileStat[]
}

export interface GitBlameLine {
  line: number
  sha: string
  author: string
  summary: string
  text: string
}

// ---------------------------------------------------------------------------
// fetchers
// ---------------------------------------------------------------------------

export async function fetchCommits(limit = 50): Promise<{ commits: GitCommit[]; limit: number }> {
  return get(`/git/commits?limit=${limit}`)
}

export async function fetchCommit(sha: string): Promise<GitCommit> {
  return get(`/git/commits/${sha}`)
}

export async function fetchDiff(sha: string): Promise<{ sha: string; diff: string }> {
  return get(`/git/diff/${sha}`)
}

export async function fetchFileHistory(path: string): Promise<{
  path: string
  history: GitCommit[]
}> {
  return get(`/git/file-history?path=${encodeURIComponent(path)}`)
}

export async function fetchBlame(path: string): Promise<{ path: string; lines: GitBlameLine[] }> {
  return get(`/git/blame?path=${encodeURIComponent(path)}`)
}
