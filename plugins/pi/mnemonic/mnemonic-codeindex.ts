// @ts-nocheck
// Mnemonic code-index tools for the Pi extension (ADR-0043): code_status,
// code_index, code_search, code_read. HTTP + redaction helpers come from
// ./api.js (shared with memory/facts).

import { Type } from "./api.js"
import { callTool, getTool, result } from "./api.js"

export const codeStatus = {
  name: "code_status",
  label: "Code index: status",
  description:
    "Check code index health before searching. If stale=true, run code_index before code_search.",
  promptSnippet: "code_status — code index freshness check (stale → code_index)",
  parameters: Type.Object({
    project: Type.Optional(Type.String({ description: "Optional project id" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const qs = args.project ? `?project=${encodeURIComponent(args.project)}` : ""
      return result(await getTool(ctx, `/code/status${qs}`))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const codeIndex = {
  name: "code_index",
  label: "Code index: run",
  description:
    "Run the incremental code index for the current working directory. Call after clone or when code_status reports stale.",
  promptSnippet: "code_index — run the incremental code index for this cwd",
  parameters: Type.Object({
    project: Type.Optional(Type.String({ description: "Optional project id" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const body = { dir: ctx?.cwd || process.cwd() }
      if (args.project) body.project = args.project
      return result(await callTool(ctx, "/code/index", body, 60000))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const codeSearch = {
  name: "code_search",
  label: "Code index: search",
  description:
    "BM25 full-text search over indexed code chunks. Prefer this over grep/rg when exploring unknown areas of a large repo; use code_read only after search narrows the location.",
  promptSnippet:
    "code_search — BM25 over indexed code chunks (prefer over grep in large repos)",
  parameters: Type.Object({
    query: Type.String({ description: "Search terms (FTS5)" }),
    limit: Type.Optional(Type.Number({ description: "Maximum hits (default 20)" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const params = new URLSearchParams()
      params.set("query", args.query)
      if (args.limit) params.set("limit", String(args.limit))
      return result(await getTool(ctx, `/code/search?${params}`))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const codeRead = {
  name: "code_read",
  label: "Code index: read",
  description:
    "Fetch indexed source for a path (and optional line range) after code_search narrows the location. Do not read whole files speculatively.",
  promptSnippet:
    "code_read — indexed source by path + line range (after code_search)",
  parameters: Type.Object({
    path: Type.String({ description: "Repo-relative file path from code_search" }),
    start_line: Type.Optional(Type.Number({ description: "Start line (1-based)" })),
    end_line: Type.Optional(Type.Number({ description: "End line (1-based)" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const params = new URLSearchParams()
      params.set("path", args.path)
      if (args.start_line) params.set("start_line", String(args.start_line))
      if (args.end_line) params.set("end_line", String(args.end_line))
      return result(await getTool(ctx, `/code/read?${params}`))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}
