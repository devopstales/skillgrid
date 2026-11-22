// @ts-nocheck
// opencode router plugin — deterministic router bootstrap injection (change
// 2026-11-17-skillgrid-host-plugins, TICKET-04).
//
// V2 entry (opencode >= 2.0): two mechanisms, selected at setup:
//   (a) durable session context entry keyed `skillgrid-router` —
//       `ctx.session.instructions.entry.put({ sessionID, key, value })` renders as
//       a `<context key="skillgrid-router">` block in the session's instructions.
//       Available on opencode >= 2.2 (PR anomalyco/opencode#56550, merged 2026-10-21).
//       Fired on the V2 `session.created` event; main sessions only; fail-open.
//   (b) "context" session-hook fallback for opencode 2.0.x/2.1.x (no
//       `ctx.session.instructions` namespace there): the hook fires on every LLM
//       request path with `{messages, system, tools}` — the same pattern as the
//       skillgrid.context plugin (verified against the v2.0.28 binary). The
//       bootstrap text is appended to the system prompt of each request as a
//       `{type:"text",text}` part. Subagent filtering uses the event's `agent`
//       id (same semantics as session.created properties: "general"/"explore"/
//       custom ids are rejected). Residual edge: a subagent session carrying a
//       primary-style agent id ("build"/"main"/"") still receives it (no
//       parent id in the event).
// Both paths are project-conditional and fail-open.
//
// V1 entry (opencode 1.x): the named-export hooks object — `event` → `session.created`.
// Dual export from day one (ADR-0045 / commit 78c78f5c precedent): the V1 named export is
// kept verbatim for 1.x harnesses, the V2 `Plugin.define({ id, setup })` default for the
// V2 loader.
//
// Dynamic import is the plugin-loading seam: the specifier is runtime-selected
// by the host (`@opencode/plugin`) with a local `Plugin.define` stub fallback for
// plain Node.

import * as fs from "node:fs"
import * as path from "node:path"

const CONTEXT_KEY = "skillgrid-router"

const STATIC_BOOTSTRAP =
  "skillgrid: router bootstrap fallback - first action: read `.agents/skills/using-skillgrid/SKILL.md`; announce `Using skillgrid:using-skillgrid to {purpose}`."

const MAX_WALK_UP = 12

// Session-aware start dir. The persistent `opencode serve --service` daemon
// (what `opencode run` connects to) has a cwd unrelated to the session's
// project, and ctx.directory is undefined in that mode — so resolve the
// session's own location.directory (ctx.directory first for --stdio servers,
// then the session record, then process.cwd()). T7 live smoke, 2026-11-22:
// without this, in-repo bootstrap injection silently failed for `opencode
// run` sessions.
async function resolveStartDir(ctx, session, ev) {
  const d = ctx ? ctx.directory : null
  if (typeof d === "string" && d.trim()) return d
  const loc = ev ? ev.location || (ev.properties && ev.properties.location) : null
  if (loc && typeof loc.directory === "string" && loc.directory.trim()) return loc.directory
  const sessionID = ev ? ev.sessionID || (ev.properties && ev.properties.sessionID) : null
  if (sessionID && session && typeof session.get === "function") {
    try {
      const rec = await session.get({ sessionID })
      if (rec && rec.location && typeof rec.location.directory === "string" && rec.location.directory.trim()) {
        return rec.location.directory
      }
    } catch {
      // fail-open: fall through to process.cwd()
    }
  }
  return process.cwd()
}

// Project-conditional: `.skillgrid/config.yaml` at or above the start dir
// (bounded walk-up). No file means "not a skillgrid project".
function isSkillgridProject(start) {
  let dir = path.resolve(start)
  for (let i = 0; i < MAX_WALK_UP; i++) {
    if (fs.existsSync(path.join(dir, ".skillgrid", "config.yaml"))) return true
    const parent = path.dirname(dir)
    if (parent === dir) break
    dir = parent
  }
  return false
}

// Repo-clone leg of the asset resolution: walk up from the start dir to the
// working-tree root (dir containing `.git`) and read the shared asset from it.
function repoCloneRoot(start) {
  let dir = path.resolve(start)
  for (let i = 0; i < MAX_WALK_UP; i++) {
    if (fs.existsSync(path.join(dir, ".git"))) return dir
    const parent = path.dirname(dir)
    if (parent === dir) break
    dir = parent
  }
  return null
}

// Live asset resolution, shared by V1 + V2: `~/.skillgrid/router-bootstrap.md`
// (installer-managed) -> repo clone asset -> static fallback (R1/R4).
function resolveBootstrap(start) {
  const home = process.env.HOME || ""
  if (home) {
    const installed = path.join(home, ".skillgrid", "router-bootstrap.md")
    try {
      const raw = fs.readFileSync(installed, "utf8")
      if (raw && raw.trim()) return raw
    } catch { /* not installed here — fall through to the clone leg */ }
  }
  const clone = repoCloneRoot(start)
  if (clone) {
    const asset = path.join(
      clone,
      ".agents",
      "skills",
      "_shared",
      "agent-config",
      "router-bootstrap.md",
    )
    try {
      const raw = fs.readFileSync(asset, "utf8")
      if (raw && raw.trim()) return raw
    } catch { /* fall through to the static fallback */ }
  }
  return STATIC_BOOTSTRAP
}

// Main-agent only: subagent/child sessions are spawned with a parent session
// marker (and/or a subagent agent id); the top-level agent session has neither.
// Primary ids observed on the live v2.0.28 install: "" / "main" / "build".
function isMainAgent(properties) {
  const parentID = properties ? properties.parentID : null
  if (parentID) return false
  const agent = properties ? properties.agent : null
  if (typeof agent === "string") {
    const a = agent.trim()
    if (a && a !== "main" && a !== "build") return false
  }
  return true
}

// Shared injection core. Fail-open: any failure degrades to "no injection" and
// never blocks session start.
export async function injectRouter(ctx, event) {
  try {
    const properties = (event && event.properties) || {}
    const sessionID = properties.sessionID || (event ? event.sessionID : "") || ""
    if (!isMainAgent(properties)) return
    const directory = await resolveStartDir(ctx, ctx && ctx.session, event)
    if (!isSkillgridProject(directory)) return
    const bootstrap = resolveBootstrap(directory)
    const entry = ctx ? ctx.session : null
    const put =
      entry && entry.instructions && entry.instructions.entry
        ? entry.instructions.entry.put
        : null
    if (!put || typeof put !== "function") return
    await put({ sessionID, key: CONTEXT_KEY, value: bootstrap })
  } catch {
    // fail-open: never block session start on injector errors
  }
}

// V1 entry (opencode 1.x): named-export hooks object; `event` keyed by the
// session lifecycle event. Kept for 1.x harnesses alongside the V2 default below.
export const SkillgridRouter = async (input) => ({
  event: async ({ event }) => {
    await injectRouter(input, event)
  },
})

async function importPlugin() {
  try {
    // Dynamic import is the plugin-loading seam: the specifier is runtime-selected
    // by the host (`@opencode/plugin`) with a local `Plugin.define` stub fallback
    // for plain Node.
    return await import("@opencode/plugin")
  } catch {
    return { Plugin: { define: (def) => def } }
  }
}

const { Plugin } = await importPlugin()

// V2 entry (opencode >= 2.0). Mechanism (a) when the durable instruction-entry
// API is present (opencode >= 2.2), else the (b) "context" session-hook
// fallback (opencode 2.0.x/2.1.x). The V1 named export above is retained for
// 1.x users (dual export, day one).
export default Plugin.define({
  id: "skillgrid.router",
  async setup(ctx) {
    const session = ctx ? ctx.session : null
    const hasInstructionsEntry = !!(
      session &&
      session.instructions &&
      session.instructions.entry &&
      typeof session.instructions.entry.put === "function"
    )

    if (hasInstructionsEntry) {
      // (a) durable <context key="skillgrid-router"> entry on session.created
      try {
        const stream = ctx.event.subscribe({ types: ["session.created"] })
        if (stream && typeof stream[Symbol.asyncIterator] === "function") {
          for await (const event of stream) {
            await injectRouter(ctx, event)
          }
        }
      } catch {
        // fail-open: a broken event stream must never block session start
      }
      return
    }

    // (b) "context" session-hook fallback: fires on every LLM request path
    // with {messages, system, tools}. Registered ONCE here — ctx.session is
    // the global domain, so per-session-created registration would accumulate
    // duplicate handlers.
    try {
      if (session && typeof session.hook === "function") {
        session.hook("context", async (event) => {
          try {
            const ev = event || {}
            // main-agent filter: the event's agent id carries the same
            // semantics as session.created properties ("build"/"main"/"" =
            // primary; "general"/"explore"/custom = subagent)
            if (!isMainAgent({ agent: ev.agent })) return
            // Session-aware start dir: the persistent --service daemon has a
            // cwd unrelated to the session's project (T7 live smoke,
            // 2026-11-22), so resolve it from the session record.
            const directory = await resolveStartDir(ctx, session, ev)
            if (!isSkillgridProject(directory)) return
            const bootstrap = resolveBootstrap(directory)
            if (Array.isArray(ev.system)) {
              // idempotent within one request assembly. Marker is "router
              // bootstrap" (present in every bootstrap variant, absent from
              // project content like AGENTS.md) — NOT "using-skillgrid", which
              // AGENTS.md itself contains and false-positives on (T7 live
              // smoke, 2026-11-22: bootstrap silently skipped in-repo).
              if (ev.system.some((t) => t && typeof t.text === "string" && t.text.toLowerCase().includes("router bootstrap"))) return
              ev.system.push({ type: "text", text: bootstrap })
              return
            }
            if (Array.isArray(ev.messages)) {
              ev.messages.unshift({ role: "user", content: [{ type: "text", text: bootstrap }] })
            }
          } catch {
            // fail-open: never block the request
          }
        })
      }
    } catch {
      // fail-open: a broken hook registration must never block session start
    }
  },
})
