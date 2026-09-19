// Shared fetch + type contracts for the Phase 7 Prototypes view. Reads the
// sandboxed .stitch/ gallery (repo root, no ?project=). All fetches send
// Accept: application/json so the shellOrJSON-wrapped /prototypes route returns
// JSON (not the SPA shell).

export class PrototypesError extends Error {
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
    throw new PrototypesError(msg, res.status)
  }
  return (await res.json()) as T
}

export async function fetchPrototypes(): Promise<{ prototypes: string[] }> {
  return get('/prototypes')
}

// fetchPrototypeHTML fetches the raw HTML of a prototype (the server serves it
// as text/html; we read it as text for the code viewer + iframe srcDoc).
export async function fetchPrototypeHTML(id: string): Promise<string> {
  const res = await fetch(`/prototypes/${id}`, { headers: { Accept: 'text/html' } })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const j = (await res.json()) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      /* keep statusText */
    }
    throw new PrototypesError(msg, res.status)
  }
  return res.text()
}
