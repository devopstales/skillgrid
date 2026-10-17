// @ts-nocheck
// Shared helpers for the skillgrid-mnemonic Pi extension (ADR-0043):
// Mnemonic HTTP transport, private-block redaction, session-id resolution,
// and the TypeBox import (with a stub fallback for the repo smoke test).

// TypeBox is a bundled skill-pi runtime dependency (ADR-0042); in Pi it
// resolves from the agent's node_modules. In the repo smoke test it is
// mocked via globalThis.__typebox (scripts/plugin-smoke.mjs), and when
// neither is available we fall back to a local stub so the tool schemas
// still build (the runtime validates, the stub only shapes).
export let Type
try {
  ;({ Type } = await import("typebox"))
} catch {
  Type = globalThis.__typebox || {
    Object: (props) => ({ kind: "object", properties: props }),
    String: (d) => ({ kind: "string", ...d }),
    Number: (d) => ({ kind: "number", ...d }),
    Boolean: (d) => ({ kind: "boolean", ...d }),
    Optional: (t) => ({ ...t, optional: true }),
  }
}

export const BASE_URL = (
  process.env.SKILLGRID_MNEMONIC_HTTP_URL || "http://127.0.0.1:7438"
).replace(/\/+$/, "")
export const TOKEN = process.env.SKILLGRID_HTTP_TOKEN || ""
export const AGENT = process.env.SKILLGRID_AGENT || "pi"

// ── HTTP ─────────────────────────────────────────────────────────────────────

function headers(json = false) {
  const h = {}
  if (json) h["Content-Type"] = "application/json"
  if (TOKEN) h.Authorization = `Bearer ${TOKEN}`
  return Object.keys(h).length ? h : undefined
}

export async function api(method, path, body, timeoutMs = 5000) {
  const res = await fetch(BASE_URL + path, {
    method,
    headers: headers(body !== undefined),
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(timeoutMs),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`)
  }
  return data
}

// Fire-and-forget: swallow every error (fail-open capture).
export async function bestEffort(path, body, timeoutMs = 5000) {
  try {
    await api("POST", path, body, timeoutMs)
  } catch {
    // fail-open: a capture failure must never surface to the model
  }
}

// ── Private-block redaction (mirrors Engram private-redaction) ───────────────

export function redactPrivateTags(text) {
  if (typeof text !== "string" || !text) return text
  return text.replace(/<private>[\s\S]*?<\/private>/g, "[REDACTED]")
}

export function redactValue(value) {
  if (typeof value === "string") return redactPrivateTags(value)
  if (Array.isArray(value)) return value.map(redactValue)
  if (value && typeof value === "object") {
    const out = {}
    for (const [k, v] of Object.entries(value)) out[k] = redactValue(v)
    return out
  }
  return value
}

// ── Session id ───────────────────────────────────────────────────────────────

export function sessionIDOf(ctx) {
  try {
    return ctx?.sessionManager?.getSessionId?.() || ""
  } catch {
    return ""
  }
}

// ── Tool helpers ─────────────────────────────────────────────────────────────

// Run a write tool call: resolve the host session id first (idempotent
// registration) so the capture lands under the right session row. Returns
// the JSON string the model sees; on error the caller returns an error
// string (never throws — the Go server is the source of truth for routing).
export async function callTool(ctx, path, body, timeoutMs = 5000) {
  const sid = sessionIDOf(ctx)
  if (sid) {
    await bestEffort("/sessions", {
      id: sid,
      agent: AGENT,
      directory: ctx?.cwd || process.cwd(),
    })
  }
  const data = await api("POST", path, body, timeoutMs)
  return JSON.stringify(data)
}

export async function getTool(ctx, path) {
  const data = await api("GET", path, undefined, 10000)
  return JSON.stringify(data)
}

export function result(text) {
  return { content: [{ type: "text", text }] }
}
