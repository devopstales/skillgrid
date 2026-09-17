import type {
  DocsContent,
  DocsRoot,
  DocsSearchResponse,
  DocsTreeResponse,
} from './types'

export class DocsError extends Error {
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
    throw new DocsError(msg, res.status)
  }
  return (await res.json()) as T
}

export function fetchTree(root: DocsRoot): Promise<DocsTreeResponse> {
  const q = root === 'all' ? '' : `?root=${encodeURIComponent(root)}`
  return get<DocsTreeResponse>(`/docs/tree${q}`)
}

export function fetchContent(path: string): Promise<DocsContent> {
  return get<DocsContent>(`/docs/content?path=${encodeURIComponent(path)}`)
}

export function fetchSearch(q: string): Promise<DocsSearchResponse> {
  return get<DocsSearchResponse>(`/docs/search?q=${encodeURIComponent(q)}`)
}
