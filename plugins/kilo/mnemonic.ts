// Mnemonic OpenCode/Kilo plugin — session lifecycle, prompt + passive capture,
// memory-protocol injection, save nudges, and compaction recovery.
//
// Copied into the agent's config dir by `skillgrid setup <agent>`. All writes
// go to the Mnemonic HTTP API (`skillgrid serve`), which persists them. The
// MCP stdio server (`skillgrid mcp`) remains the primary tool transport; this
// plugin handles the *event-driven* capture that the agent does not do itself
// (user prompts, Task output, save reminders, compaction context).
//
// Ported to the Mnemonic engine from gentleman-ai's Engram adapter, so both
// memory systems share one session-ownership contract:
//   - writes (tool.execute.before) are FAIL-CLOSED: if the authoritative
//     session cannot be resolved or registered, the hook throws so the call
//     is retryable instead of silently mis-attributed;
//   - capture hooks (chat.message, Task passive) are FAIL-OPEN: they skip
//     silently when ownership is unresolvable;
//   - deleted session trees are tombstoned for the lifetime of the plugin —
//     late events and re-creations must not revive them;
//   - sub-agent sessions never register as top-level Mnemonic sessions;
//     their mem_* writes bind to the authoritative root.
//
// Event flow:
//   1. session.created/updated → record ownership; register roots (idempotent)
//   2. chat.message            → POST /prompts (strip <private>, truncate)
//   3. tool.execute.before     → bind session_id onto mem_* write args
//   4. tool.execute.after      → POST /observations/passive for Task output
//   5. system transform        → inject protocol + debounced save nudge
//   6. session.compacting      → inject compaction context + first-action rule
//
// Env overrides:
//   SKILLGRID_MNEMONIC_HTTP_URL   (default http://127.0.0.1:7438)
//   SKILLGRID_MNEMONIC_HTTP_TOKEN (bearer for write routes, empty = open)
//   SKILLGRID_MNEMONIC_BIN        (default "skillgrid")
//   SKILLGRID_MNEMONIC_NUDGE_COOLDOWN_SECS (default 900)

const BASE_URL = (
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ?? "http://127.0.0.1:7438"
).replace(/\/+$/, "")
const TOKEN = process.env.SKILLGRID_MNEMONIC_HTTP_TOKEN ?? ""
const BIN = process.env.SKILLGRID_MNEMONIC_BIN ?? "skillgrid"

const RESOLUTION_ERROR_PREFIX = "mnemonic could not resolve an authoritative session for"
const REGISTRATION_ERROR_PREFIX = "mnemonic could not confirm session registration for"

// ─────────────────────────── HTTP client ───────────────────────────

function req(method: string, path: string, body?: unknown): Promise<any> {
  const headers: Record<string, string> = { "Content-Type": "application/json" }
  if (TOKEN) headers["Authorization"] = "Bearer " + TOKEN
  return fetch(BASE_URL + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(3000),
  })
    .then(async (res: any) => {
      if (!res || res.ok === false) return { _status: res?.status ?? 0 }
      if (typeof res?.json === "function") {
        const data = await res.json()
        return data ?? {}
      }
      const text = typeof res?.text === "function" ? await res.text() : ""
      if (!text) return {}
      try { return JSON.parse(text) } catch { return {} }
    })
    .catch(() => ({ _status: 0 }))
}

async function health(): Promise<boolean> {
  try {
    const res = await fetch(BASE_URL + "/health", { signal: AbortSignal.timeout(500) })
    return res.ok
  } catch { return false }
}

// ─────────────────────────── Privacy strip ───────────────────────────
// Double safety: the Go engine redacts too, but sensitive spans never hit
// the wire when the caller marks them <private>…</private>.
function stripPrivateTags(str: string): string {
  if (!str) return ""
  return str.replace(/<private>[\s\S]*?<\/private>/gi, "[REDACTED]").trim()
}

const MAX_PROMPT_LENGTH = 2000
function truncate(str: string, max: number): string {
  if (!str) return ""
  return str.length > max ? str.slice(0, max) + "…" : str
}

// ─────────────────────── Project resolution ────────────────────────
// Prefer the authoritative id returned by the server (POST /sessions). Fall
// back to resolving locally the same way the Go engine does (git remote
// origin basename, else directory basename), so capture calls always have a
// project scope even before a session is registered.

let serverProjectID = ""

function resolveProjectFallback(directory: string): string {
  let remote = ""
  try {
    if ((globalThis as any).Bun) {
      const r = (globalThis as any).Bun.spawnSync(["git", "-C", directory, "remote", "get-url", "origin"])
      if (r?.exitCode === 0 && r?.stdout) remote = r.stdout.toString().trim()
    } else if ((globalThis as any).process?.execSync) {
      remote = (globalThis as any).process.execSync(
        `git -C ${JSON.stringify(directory)} remote get-url origin`,
        { stdio: ["ignore", "pipe", "ignore"] },
      ).toString().trim()
    }
  } catch { remote = "" }
  if (remote) {
    const name = remote.replace(/\.git$/, "").split(/[/:]/).pop()
    if (name) return name.toLowerCase().replace(/[-_]+/g, "-").replace(/^-|-$/g, "")
  }
  const base = (directory || "").replace(/\/+$/, "").split("/").pop()
  return (base || "unknown").toLowerCase()
}

function projectFor(directory: string): string {
  if (serverProjectID) return serverProjectID
  return resolveProjectFallback(directory || process.cwd())
}

// ─────────────────────────── Protocol ─────────────────────────────

const INLINE_PROTOCOL = `# Mnemonic Memory Protocol

Mnemonic is skillgrid's local-first memory (SQLite + FTS5): \`mem_*\`, \`code_*\`, \`web_*\` tools over MCP and HTTP.

## When to Save (mandatory)
Call \`mem_save\` immediately after: a bug fix, an architecture/design decision, a non-obvious discovery, a config change, a new pattern, a learned user preference, or a user correction.

## Save Format
- title: verb + what (short, searchable)
- type: standing | preference | convention | decision | architecture | bugfix | pattern | config | correction | discovery | learning | lesson | session_log
- scope: project (default) | user | global
- topic_key (optional): stable key for upserts, e.g. architecture/auth-model
- content: **What** / **Why** / **Where** / **Learned** sections

## Session Protocol
1. \`mem_session_start\` on open — pass \`title\` (short name of the session's goal, e.g. "Fix login race"). Shown in the dashboard session list.
2. \`mem_save\` during work (dedups by hash within 24h, upserts by topic_key)
3. \`mem_search\` before starting work that may overlap prior sessions
4. \`mem_session_summary\` + \`mem_session_end\` before closing

## Code Index Ladder
\`code_status\` → \`code_index\` (if stale) → \`code_search\` → \`code_read\`.

## Web Cache
\`web_cache_lookup\` before remote research MCPs; \`web_cache_save\` immediately after Context7/Exa/DeepWiki/WebFetch returns; \`web_cache_search\` over cached research.

## Privacy
Wrap anything sensitive (tokens, paths with secrets, PII) in \`<private>…</private>\` — it is stripped before storage.

## Passive Capture
Finishing a \`Task\`/subagent does NOT require a manual \`mem_save\` — the server extracts "Key Learnings:" sections automatically. Do include a "## Key Learnings:" section in task output so it is captured.

## After Compaction
If you see "FIRST ACTION REQUIRED" as your first prompt, call \`mem_session_summary\` with the compacted summary BEFORE anything else. The session-only compaction context has already been injected; use project-scoped \`mem_context\` only when explicitly requested.`

async function loadProtocol(): Promise<string> {
  try {
    const fs = (await import("node:fs/promises")).default
    const path = (await import("node:path")).default
    const os = (await import("node:os")).default
    const candidates = [
      path.join(os.homedir(), ".config", "kilo", "shared", "memory-protocol.md"),
      path.join(os.homedir(), ".config", "opencode", "shared", "memory-protocol.md"),
    ]
    for (const c of candidates) {
      try {
        const text = await fs.readFile(c, "utf8")
        if (text.trim()) return text
      } catch { /* try next */ }
    }
  } catch { /* use inline */ }
  return INLINE_PROTOCOL
}

// ─────────────────────── Plugin state ─────────────────────────────

const SUB_AGENT_TOOLS = new Set([
  "mem_save", "mem_save_prompt", "mem_session_summary", "mem_capture_passive",
])

async function ensureServer(): Promise<void> {
  if (await health()) return
  try {
    const cp = await import("node:child_process")
    const child = (cp as any).spawn?.(BIN, ["serve"], { stdio: "ignore", detached: true })
    child?.unref?.()
  } catch { /* binary not found — plugin no-ops */ }
}

type Plugin = (ctx: any) => any

export const Mnemonic: Plugin = async (ctx) => {
  const directory = ctx?.directory ?? process.cwd()
  const protocol = await loadProtocol()

  const parentOf = new Map<string, string | null>() // sessionId -> parentId|null (null = confirmed root)
  const subAgents = new Set<string>()               // ids known to have a parent
  const invalid = new Set<string>()                 // tombstoned trees (deleted)
  const known = new Set<string>()                   // ids confirmed registered with the server
  const lastNudge = new Map<string, number>()       // root session id -> epoch seconds

  // ── SDK lookup (plugin-reload recovery) ──
  // After the plugin reloads, session.created events are not replayed. The
  // OpenCode SDK can still answer "who owns this session?" — we use that to
  // re-derive the root without re-asking the agent. Failures are fail-closed:
  // an unresolvable session is never forwarded as its own top-level root.
  async function sdkGet(id: string): Promise<any | undefined> {
    const client = ctx?.client?.session
    if (typeof client?.get !== "function") return undefined
    try {
      const result = await client.get({ path: { id } })
      const info = result?.data
      const status = result?.response?.status
      if (
        result?.error ||
        (typeof status === "number" && status >= 400) ||
        !info ||
        typeof info.id !== "string" ||
        info.id !== id ||
        typeof info.projectID !== "string" ||
        info.projectID === "" ||
        (ctx?.project?.id && info.projectID !== ctx.project.id) ||
        // undefined parentID = root; "" = unauthorised (OpenCode quirk)
        (info.parentID !== undefined && (typeof info.parentID !== "string" || info.parentID === ""))
      ) return undefined
      return info
    } catch {
      return undefined
    }
  }

  // Walk the ownership chain from sessionId to the authoritative root.
  // Stages successful lookups locally and only publishes them (or tombstones
  // the whole staged chain) once a validity decision has been made, so events
  // landing mid-lookup correctly invalidate half-built chains.
  async function resolveRoot(startId: string): Promise<string> {
    if (!startId) return ""
    if (invalid.has(startId)) return ""
    if (subAgents.has(startId) && !parentOf.has(startId)) return ""
    const visited = new Set<string>()
    const resolved = new Map<string, string | null>()
    const publishResolvedParents = (): void => {
      for (const [id, p] of resolved) {
        if (!parentOf.has(id)) {
          parentOf.set(id, p)
          if (p) subAgents.add(id)
        }
      }
    }
    const invalidateResolvedTree = (id: string): void => {
      publishResolvedParents()
      invalidateTree(id)
    }

    let cur = startId
    for (;;) {
      if (visited.has(cur)) return "" // cycle
      if (invalid.has(cur)) {
        invalidateResolvedTree(cur)
        return ""
      }
      visited.add(cur)

      let parent: string | null
      if (parentOf.has(cur)) {
        parent = parentOf.get(cur) as string | null
      } else {
        const info = await sdkGet(cur)
        if (info === undefined) return ""
        if (invalid.has(cur)) {
          invalidateResolvedTree(cur)
          return ""
        }
        if (parentOf.has(cur)) {
          parent = parentOf.get(cur) as string | null
        } else {
          parent = info.parentID === undefined ? null : info.parentID
          resolved.set(cur, parent)
        }
      }

      if (parent === null) {
        // Reached a root: verify nothing in the staged chain was tombstoned.
        for (const [id, p] of resolved) {
          const bad = invalid.has(id) ? id : (p && invalid.has(p) ? p : "")
          if (bad) {
            invalidateResolvedTree(bad)
            return ""
          }
        }
        publishResolvedParents()
        return cur
      }
      cur = parent
    }
  }

  // Register (idempotent) a session under its own client id. Only top-level
  // roots are ever registered. Failure (5xx, network) is retryable: the known
  // set is only updated after the server acknowledges with `created: true`
  // or `created: false` (idempotent re-create), so the next attempt re-POSTs
  // and does not cache a failed registration.
  async function ensureSession(sessionId: string, title = ""): Promise<boolean> {
    if (!sessionId || invalid.has(sessionId)) return false
    if (subAgents.has(sessionId) && !parentOf.has(sessionId)) return false
    if (known.has(sessionId)) return true
    const res = await req("POST", "/sessions", { id: sessionId, directory, title })
    if (!res || typeof res !== "object" || typeof res.created !== "boolean") return false
    if (res.project_id) serverProjectID = String(res.project_id)
    known.add(sessionId)
    return true
  }

  // Record an authoritative ownership event. Mirrors the Engram contract:
  //  - missing parentID field  → confirmed root
  //  - non-empty parentID      → child
  //  - empty/odd parentID, or a project mismatch → event is unauthorised
  //    (clear any cached state for that id and skip)
  //  - tombstoned ids or ids whose new parent is tombstoned stay tombstoned.
  function recordSession(info: any): boolean {
    const id = info?.id
    if (typeof id !== "string" || id === "") return false
    const rawParent = info?.parentID
    const parent = rawParent === undefined
      ? null
      : (typeof rawParent === "string" && rawParent !== "" ? rawParent : undefined)
    const rawProject = info?.projectID
    const invalidProject =
      typeof rawProject !== "string" ||
      rawProject === "" ||
      (ctx?.project?.id && rawProject !== ctx.project.id)
    if (parent === undefined || invalidProject) {
      parentOf.delete(id)
      subAgents.delete(id)
      return false
    }
    if (invalid.has(id) || (parent && invalid.has(parent))) {
      invalidateTree(id)
      return false
    }
    parentOf.set(id, parent)
    if (parent) subAgents.add(id)
    else subAgents.delete(id)
    return true
  }

  // Tombstone a session and every cached descendant; removed from the known
  // set so it can never re-register. Late events must not revive it.
  function invalidateTree(id: string): void {
    const doomed = new Set([id])
    let grew = true
    while (grew) {
      grew = false
      for (const [child, par] of parentOf) {
        if (par && doomed.has(par) && !doomed.has(child)) {
          doomed.add(child)
          grew = true
        }
      }
    }
    for (const d of doomed) {
      invalid.add(d)
      known.delete(d)
      subAgents.delete(d)
      parentOf.delete(d)
    }
  }

  // ── Server bootstrap (once, on load) — non-fatal ──
  ensureServer()

  return {
    // ── Session lifecycle ──
    event: async ({ event }: any) => {
      if (event?.type === "session.created" || event?.type === "session.updated") {
        const info = event?.properties?.info ?? event?.properties
        if (!info?.id) return
        if (!recordSession(info)) return
        if (event.type === "session.created" && !subAgents.has(info.id)) {
          await ensureSession(info.id, String(info.title ?? ""))
        }
      } else if (event?.type === "session.deleted") {
        const info = event?.properties?.info ?? event?.properties
        if (info?.id) invalidateTree(info.id)
      }
    },

    // ── User prompt capture (fail-open) ──
    "chat.message": async (input: any, output: any) => {
      const client = String(input?.sessionID ?? "")
      const sessionId = await resolveRoot(client)
      // Child prompts skip: the sub-agent's narrative is captured via its
      // Task output, not its chat stream.
      if (!sessionId || subAgents.has(client)) return
      if (!(await ensureSession(sessionId))) return
      const confirmed = await resolveRoot(client)
      if (confirmed !== sessionId) return

      const parts: any[] = output?.parts ?? []
      const text = parts
        .filter((p) => p?.type === "text")
        .map((p) => p?.text ?? "")
        .join("\n")
        .trim()
      const summary = output?.message?.summary
      const fallback = summary ? `${summary.title ?? ""}\n${summary.body ?? ""}`.trim() : ""
      const content = text || fallback
      if (content.length <= 10) return

      await req("POST", `/prompts?project=${encodeURIComponent(projectFor(directory))}`, {
        session_id: sessionId,
        content: stripPrivateTags(truncate(content, MAX_PROMPT_LENGTH)),
        project: projectFor(directory),
      })
    },

    // ── Write hook: bind session_id onto mem_* args (fail-closed) ──
    // On any ownership/registration failure the hook throws so the tool call
    // fails loudly and is retryable — a wrong session id would mis-attribute
    // the write forever.
    "tool.execute.before": async (input: any, output: any) => {
      const tool = String(input?.tool ?? "").toLowerCase()
      if (!SUB_AGENT_TOOLS.has(tool)) return
      const toolName = String(input?.tool ?? tool)
      const client = String(input?.sessionID ?? "")

      const sessionId = await resolveRoot(client)
      if (!sessionId) {
        throw new Error(`${RESOLUTION_ERROR_PREFIX} ${toolName}`)
      }
      const registered = await ensureSession(sessionId)
      const confirmed = await resolveRoot(client)
      if (confirmed !== sessionId) {
        throw new Error(`${RESOLUTION_ERROR_PREFIX} ${toolName}`)
      }
      if (!registered) {
        throw new Error(
          `${REGISTRATION_ERROR_PREFIX} ${toolName}; verify that the Mnemonic server is available and retry`,
        )
      }
      output.args = output.args ?? {}
      output.args.session_id = sessionId
    },

    // ── Task passive capture (fail-open): attribute to the authoritative root ──
    "tool.execute.after": async (input: any, output: any) => {
      if (String(input?.tool ?? "") !== "Task" || output === undefined || output === null) return
      const client = String(input?.sessionID ?? "")
      const sessionId = await resolveRoot(client)
      if (!sessionId) return
      const text = typeof output === "string" ? output : JSON.stringify(output)
      if (text.length <= 50) return
      if (!(await ensureSession(sessionId))) return
      const confirmed = await resolveRoot(client)
      if (confirmed !== sessionId) return
      await req("POST", `/observations/passive?project=${encodeURIComponent(projectFor(directory))}`, {
        session_id: sessionId,
        content: stripPrivateTags(text),
        source: "task-complete",
      })
    },

    // ── System prompt: protocol + debounced save nudge (fail-open) ──
    "experimental.chat.system.transform": async (input: any, output: any) => {
      const sys: string[] = output?.system ?? []
      const block = "\n\n" + protocol
      if (sys.length > 0) sys[sys.length - 1] += block
      else sys.push(block)
      output.system = sys

      // Nudge: only old sessions with a stale last-save get reminded.
      try {
        const sessionID = String(input?.sessionID ?? "")
        if (!sessionID || invalid.has(sessionID) || subAgents.has(sessionID)) return
        const root = await resolveRoot(sessionID)
        if (!root || root === sessionID && subAgents.has(sessionID)) return

        const cooldown = parseInt(process.env.SKILLGRID_MNEMONIC_NUDGE_COOLDOWN_SECS ?? "900", 10)
        const nowSecs = Math.floor(Date.now() / 1000)
        const last = lastNudge.get(root)
        if (last !== undefined && nowSecs - last < cooldown) return

        const proj = projectFor(directory)
        const toEpoch = (ts: string): number => {
          if (!ts) return 0
          const norm = ts.includes("T") ? ts : ts.replace(" ", "T") + "Z"
          const ms = Date.parse(norm)
          return Number.isNaN(ms) ? 0 : Math.floor(ms / 1000)
        }

        const sess = await req("GET", `/sessions/${encodeURIComponent(root)}?project=${encodeURIComponent(proj)}`)
        const startEpoch = toEpoch(String(sess?.started_at ?? ""))
        if (startEpoch > 0 && nowSecs - startEpoch < 300) return // session too young

        const lastRes = await req("GET", `/memory/last-save-at?project=${encodeURIComponent(proj)}`)
        const lastEpoch = toEpoch(String(lastRes?.last_save_at ?? ""))
        if (!lastEpoch || nowSecs - lastEpoch < 900) return

        const nudge = "\n\nMEMORY REMINDER: It's been over 15 minutes since your last memory save. " +
          "If you've made decisions, discoveries, completed significant work, or found non-obvious things, " +
          "call mem_save now."
        if (output.system.length > 0) output.system[output.system.length - 1] += nudge
        else output.system.push(nudge)
        lastNudge.set(root, nowSecs)
      } catch { /* never let the nudge crash the transform */ }
    },

    // ── Compaction recovery: always inject the first-action rule ──
    "experimental.session.compacting": async (input: any, output: any) => {
      const client = String(input?.sessionID ?? "")
      const sessionId = await resolveRoot(client)
      if (sessionId && (await ensureSession(sessionId))) {
        const proj = projectFor(directory)
        try {
          const data = await req("GET", `/context/compaction?project=${encodeURIComponent(proj)}&session_id=${encodeURIComponent(sessionId)}&limit=5`)
          if (typeof data?.context === "string") {
            (output.context ??= []).push(data.context)
          } else if (data?.context) {
            (output.context ??= []).push(flattenCompaction(data.context))
          }
        } catch { /* context fetch is non-fatal; the first-action rule still goes out */ }
      }
      (output.context ??= []).push(
        `CRITICAL INSTRUCTION FOR COMPACTED SUMMARY:\n` +
        `FIRST ACTION REQUIRED: Call mem_session_summary with the content of this compacted summary ` +
        `(project: '${projectFor(directory)}', session_id: '${sessionId || client}'). This preserves what was accomplished before ` +
        `compaction. Do this BEFORE any other work.\n` +
        `This is NOT optional. Without it, everything done before compaction is lost from memory.`
      )
    },
  }
}

function flattenCompaction(c: any): string {
  const lines: string[] = ["## Mnemonic compaction context"]
  if (c.title) lines.push(`- Session: ${c.title}`)
  if (c.summary) lines.push(`- Last summary: ${truncate(c.summary, 1200)}`)
  if (Array.isArray(c.observations) && c.observations.length) {
    lines.push("- Recent work:")
    for (const o of c.observations) lines.push(`  - ${o}`)
  }
  if (lines.length === 1) return ""
  return lines.join("\n")
}

export default Mnemonic
