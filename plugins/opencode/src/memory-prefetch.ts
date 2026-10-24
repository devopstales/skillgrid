// @ts-nocheck
// Global memory prefetch injection (change 2026-10-24-global-llm-wiki,
// TICKET-05, ADR-0045): renders the frozen snapshot of the __global__ fact
// store into the system prompt at session start.
//
// The OpenCode V2 `context` hook fires on EVERY LLM request (primary,
// compaction, generate), so the snapshot is memoized per session ID:
// rendered once, cached, and the cached block is returned verbatim on every
// later request for the same session (blueprint M8). Mid-session fact_add
// does NOT change the system prompt until the next session.
//
// The snapshot comes from the MCP resource memory/global over the mnemonic
// HTTP surface (the same facts.RenderFrozenSnapshot the CLI and MCP server
// use). Fail-open: any error (server down, bad response, empty block) yields
// no injection — the system prompt must never break the session.
const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

// readGlobalResource fetches the frozen global memory snapshot from the
// HTTP surface (GET /memory/global → {snapshot}). Any failure (network,
// non-200, non-object body, non-string snapshot) yields "" — the fail-open
// contract.
export async function readGlobalResource(baseUrl = BASE_URL) {
  try {
    const res = await fetch(baseUrl + "/memory/global", {
      signal: AbortSignal.timeout(5000),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) return ""
    const text = data && data.snapshot
    return typeof text === "string" ? text : ""
  } catch {
    return ""
  }
}

// resolveSessionID extracts the session id from the hook event: the V2
// context event carries sessionID/sessionId; both spellings are honored,
// falling back to the literal "unknown" (still memoizable).
export function resolveSessionID(event) {
  try {
    const id =
      (event && (event.sessionID || event.sessionId)) ||
      (event && event.session && event.session.id) ||
      ""
    return typeof id === "string" && id !== "" ? id : "unknown"
  } catch {
    return "unknown"
  }
}

// injectBlock appends the frozen block to the system prompt exactly once per
// system entry: idempotent (a system string already containing the block is
// left untouched), and it never throws.
export function injectBlock(event, block) {
  try {
    if (!block) return
    const system = event && event.system
    if (!Array.isArray(system)) return
    for (let i = 0; i < system.length; i++) {
      const entry = system[i]
      if (!entry || typeof entry !== "object") continue
      if (typeof entry.text !== "string") continue
      if (entry.text.includes(block)) continue // already injected: skip
      entry.text = entry.text + "\n\n" + block
    }
  } catch {
    // fail-open: never block the request
  }
}

// buildMemoryPrefetchPlugin assembles the V2 plugin definition with the
// per-session memoization cache. Exported separately so the behavior is
// testable without @opencode/plugin (the V2 entrypoint below wraps it).
export function buildMemoryPrefetchPlugin(opts = {}) {
  const fetchBlock = opts.fetchBlock || readGlobalResource
  const cache = new Map() // sessionID -> frozen block
  return {
    id: "skillgrid.memory-prefetch",
    async setup(ctx) {
      ctx.session.hook("context", async (event) => {
        try {
          const sessionID = resolveSessionID(event)
          let block = cache.get(sessionID)
          if (block === undefined) {
            block = await fetchBlock()
            // An empty block (store empty, server down) is cached too:
            // re-fetching on every LLM request would defeat the frozen
            // contract and burn a round trip per request.
            cache.set(sessionID, block)
          }
          if (!block) return
          injectBlock(event, block)
        } catch {
          // fail-open: never block the request
        }
      })
    },
  }
}

// V1 named export (harness parity with the other skillgrid plugins).
export const MemoryPrefetch = buildMemoryPrefetchPlugin()

// V2 entrypoint (ADR-0032 V2 port / ADR-0045): Plugin.define with the
// context session hook. In the repo smoke test (plain Node) @opencode/plugin
// is absent, so fall back to a local Plugin.define stub.
async function importPlugin() {
  try {
    return await import("@opencode/plugin")
  } catch {
    return { Plugin: { define: (def) => def } }
  }
}

const { Plugin } = await importPlugin()

export default Plugin.define(buildMemoryPrefetchPlugin())
