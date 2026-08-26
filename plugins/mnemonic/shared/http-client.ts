/**
 * Skillgrid Mnemonic — HTTP client for OpenCode/Kilo plugins.
 *
 * Talks to `skillgrid serve` (default http://127.0.0.1:7438).
 * MCP tool calls remain separate (registered in opencode.json).
 */

import { spawn } from "node:child_process"

export const DEFAULT_MNEMONIC_URL = "http://127.0.0.1:7438"
export const SKILLGRID_BIN = process.env.SKILLGRID_BIN ?? "skillgrid"

export function getBaseUrl(): string {
  return (process.env.SKILLGRID_MNEMONIC_URL ?? DEFAULT_MNEMONIC_URL).replace(/\/$/, "")
}

function authHeaders(): Record<string, string> | undefined {
  const token = process.env.SKILLGRID_HTTP_TOKEN
  if (!token) return undefined
  return { Authorization: `Bearer ${token}` }
}

export async function mnemonicFetch<T = unknown>(
  path: string,
  opts: { method?: string; body?: unknown; timeoutMs?: number } = {},
): Promise<T | null> {
  const baseUrl = getBaseUrl()
  const timeoutMs = opts.timeoutMs ?? 3000

  try {
    const headers: Record<string, string> = { ...(authHeaders() ?? {}) }
    if (opts.body !== undefined) {
      headers["Content-Type"] = "application/json"
    }

    const res = await fetch(`${baseUrl}${path}`, {
      method: opts.method ?? "GET",
      headers: Object.keys(headers).length > 0 ? headers : undefined,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: AbortSignal.timeout(timeoutMs),
    })

    if (!res.ok) return null

    try {
      return (await res.json()) as T
    } catch {
      return {} as T
    }
  } catch {
    return null
  }
}

export async function isServerHealthy(baseUrl = getBaseUrl()): Promise<boolean> {
  try {
    const res = await fetch(`${baseUrl}/health`, {
      signal: AbortSignal.timeout(500),
    })
    return res.ok
  } catch {
    return false
  }
}

export async function waitForHealth(
  baseUrl: string,
  timeoutMs: number,
  intervalMs = 200,
): Promise<boolean> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (await isServerHealthy(baseUrl)) return true
    await new Promise((r) => setTimeout(r, intervalMs))
  }
  return false
}

/** Start `skillgrid serve` when GET /health fails (Engram-parity auto-start). */
export async function ensureServer(baseUrl = getBaseUrl()): Promise<boolean> {
  if (await isServerHealthy(baseUrl)) return true

  try {
    const child = spawn(SKILLGRID_BIN, ["serve"], {
      detached: true,
      stdio: "ignore",
    })
    child.unref()
  } catch {
    return false
  }

  return waitForHealth(baseUrl, 5000)
}
