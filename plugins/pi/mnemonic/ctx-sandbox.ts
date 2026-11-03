// @ts-nocheck
// Context-harness sandbox tools + tool_result overflow handler for the Pi
// extension (ADR-0044). pi-context lesson: oversized tool output is stored
// server-side and the model sees a compact receipt with the retrieval path,
// not the raw flood.
//
//   - tool_result overflow: text content above CTX_THRESHOLD replaces the
//     model-visible content with a receipt (POST /ctx/capture → sandbox_id).
//     Fail-open: any error (dead server, no session) keeps the original.
//   - ctx_search / ctx_get / ctx_stats / ctx_purge tools over the new HTTP
//     routes (GET /ctx/search, GET /observations-style fetch, GET /ctx/stats,
//     DELETE /ctx/sessions/{id}).
//
// Skip list: never index our own ctx_* retrieval tools (recursion, pi-context
// convention) and never re-index a receipt (marker guard).

import { Type } from "./api.js"
import { AGENT, api, bestEffort, redactPrivateTags, result, sessionIDOf } from "./api.js"

// Mirrors context_harness.SandboxThreshold (4096 bytes). A client-side gate
// that keeps small outputs off the wire entirely; the server gate is a
// second, authoritative pass reported in the capture response.
const CTX_THRESHOLD = 4096
const MAX_CAPTURE_BYTES = 200000 // hard cap: never ship >200KB per result
const RECEIPT_MARKER = "[skillgrid-ctx]"
const SKIP_TOOLS = new Set(["ctx_search", "ctx_get", "ctx_stats", "ctx_purge"])

function joinText(content) {
  return (content || [])
    .filter((c) => c && c.type === "text" && typeof c.text === "string")
    .map((c) => c.text)
    .join("\n")
}

// toolInputSummary builds the receipt's input summary (command/path/query),
// mirroring pi-context's summarize_tool_input (500-char cap).
function toolInputSummary(input) {
  try {
    const json = JSON.stringify(input || {})
    return json.length > 500 ? json.slice(0, 497) + "..." : json
  } catch {
    return null
  }
}

function receipt(toolName, sandboxId, sizeBytes, input) {
  const lines = [
    `${RECEIPT_MARKER} ${toolName} output stored server-side (${sizeBytes} bytes, sandbox_id ${sandboxId}).`,
    "Retrieve with ctx_search (snippet-first FTS, e.g. ctx_search {query: \"<term>\"}) or ctx_get {sandbox_id: " +
      sandboxId + "} for the full text.",
  ]
  const inputSummary = toolInputSummary(input)
  if (inputSummary) lines.push(`Input: ${inputSummary}`)
  return lines.join("\n")
}

// registerToolResultOverflow wires the tool_result handler. Returns the
// handler function so the smoke test can exercise it directly.
export function registerToolResultOverflow(pi) {
  const handler = async (event, ctx) => {
    try {
      const sid = sessionIDOf(ctx)
      if (!sid) return
      const toolName = String(event?.toolName || "")
      if (SKIP_TOOLS.has(toolName)) return
      const text = joinText(event?.content)
      if (!text || text.includes(RECEIPT_MARKER)) return
      if (text.length <= CTX_THRESHOLD) return
      const body = text.length > MAX_CAPTURE_BYTES ? text.slice(0, MAX_CAPTURE_BYTES) : text
      const data = await api(
        "POST",
        "/ctx/capture?directory=" + encodeURIComponent(ctx?.cwd || process.cwd()),
        {
          session_id: sid,
          agent: AGENT,
          tool_name: toolName,
          content: redactPrivateTags(body),
        },
        5000,
      )
      if (!data || typeof data.sandbox_id !== "number") return
      return {
        content: [{ type: "text", text: receipt(toolName, data.sandbox_id, text.length, event?.input) }],
      }
    } catch {
      // fail-open: a capture failure keeps the original content
      return undefined
    }
  }
  pi.on("tool_result", handler)
  return handler
}

export const ctxSearch = {
  name: "ctx_search",
  label: "Context: search",
  description:
    "Snippet-first FTS over oversized tool output stored in the Mnemonic sandbox (plus indexed code chunks, fused by RRF). Use after a tool result was replaced by a [skillgrid-ctx] receipt to recall what that output said.",
  promptSnippet:
    "ctx_search — FTS over stored oversized tool output + indexed code (snippet-first; use ctx_get for full text)",
  parameters: Type.Object({
    query: Type.String({ description: "Search keywords" }),
    limit: Type.Optional(Type.Number({ description: "Max results (default 20)" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const params = new URLSearchParams()
      params.set("query", args.query)
      if (args.limit) params.set("limit", String(args.limit))
      params.set("directory", ctx?.cwd || process.cwd())
      const data = await api("GET", `/ctx/search?${params}`, undefined, 10000)
      return result(JSON.stringify(data))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const ctxGet = {
  name: "ctx_get",
  label: "Context: get",
  description:
    "Fetch the full stored text of one sandboxed tool output by sandbox_id (from a [skillgrid-ctx] receipt). Prefer ctx_search first; use this only after a hit narrows the target.",
  promptSnippet:
    "ctx_get — full text of one stored tool output by sandbox_id (after ctx_search)",
  parameters: Type.Object({
    sandbox_id: Type.Number({ description: "Sandbox id from a receipt or ctx_search" }),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const data = await api(
        "GET",
        `/ctx/get/${args.sandbox_id}?directory=${encodeURIComponent(ctx?.cwd || process.cwd())}`,
        undefined,
        10000,
      )
      return result(JSON.stringify(data))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const ctxStats = {
  name: "ctx_stats",
  label: "Context: stats",
  description:
    "Sandbox storage stats for the current project: stored row count, total bytes, indexed chunk count.",
  promptSnippet: "ctx_stats — sandbox storage stats (rows, bytes)",
  parameters: Type.Object({}),
  async execute(_toolCallId, _args, _signal, _onUpdate, ctx) {
    try {
      const data = await api(
        "GET",
        `/ctx/stats?directory=${encodeURIComponent(ctx?.cwd || process.cwd())}`,
        undefined,
        10000,
      )
      return result(JSON.stringify(data))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const ctxPurge = {
  name: "ctx_purge",
  label: "Context: purge",
  description:
    "Delete all sandboxed tool outputs for the current session (session-scoped). Returns the row count purged.",
  promptSnippet: "ctx_purge — delete this session's stored tool output",
  parameters: Type.Object({}),
  async execute(_toolCallId, _args, _signal, _onUpdate, ctx) {
    try {
      const sid = sessionIDOf(ctx)
      if (!sid) return result(JSON.stringify({ error: "no active session" }))
      const data = await api(
        "DELETE",
        `/ctx/sessions/${encodeURIComponent(sid)}?directory=${encodeURIComponent(ctx?.cwd || process.cwd())}`,
        undefined,
        10000,
      )
      return result(JSON.stringify(data))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}
