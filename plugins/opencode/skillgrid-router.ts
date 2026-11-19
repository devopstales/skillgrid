// @ts-nocheck
// opencode router plugin — deterministic router bootstrap injection (change
// 2026-11-17-skillgrid-host-plugins, TICKET-04).
//
// V2 entry (opencode >= 2.0): the bootstrap is installed as the durable session
// context entry keyed `skillgrid-router` — `ctx.session.instructions.entry.put({
// sessionID, key, value })` renders as a `<context key="skillgrid-router">` block in
// the session's instructions. Fired on the V2 `session.created` event; main sessions
// only; project-conditional; fail-open.
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

function startDir(ctx) {
  const d = ctx ? ctx.directory : null
  return typeof d === "string" && d.trim() ? d : process.cwd()
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
    const directory = startDir(ctx)
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

// V2 entry (opencode >= 2.0): durable `<context key="skillgrid-router">` entry
// installed on the `session.created` event. The V1 named export above is retained
// for 1.x users (dual export, day one).
export default Plugin.define({
  id: "skillgrid.router",
  async setup(ctx) {
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
  },
})
