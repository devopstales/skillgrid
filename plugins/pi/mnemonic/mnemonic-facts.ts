// @ts-nocheck
// Mnemonic fact-memory tools for the Pi extension (ADR-0043): fact_add,
// fact_search, fact_forget. HTTP + redaction helpers come from ./api.js
// (shared with memory/code).

import { Type } from "./api.js"
import { callTool, redactValue, result, sessionIDOf } from "./api.js"

export const factAdd = {
  name: "fact_add",
  label: "Facts: add",
  description: "Add a durable fact to Fact Memory. Returns the new fact id.",
  promptSnippet: "fact_add — add a durable fact (returns the fact id)",
  parameters: Type.Object({
    content: Type.String({ description: "Fact text (required)" }),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const sid = sessionIDOf(ctx)
      return result(
        await callTool(ctx, "/facts", redactValue({
          content: args.content,
          session_id: sid || undefined,
        })),
      )
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const factSearch = {
  name: "fact_search",
  label: "Facts: search",
  description:
    "Lexical FTS search over Fact Memory. Returns matching facts ranked by bm25.",
  promptSnippet: "fact_search — FTS over durable facts (bm25 ranked)",
  parameters: Type.Object({
    query: Type.String({ description: "Search terms (any-term recall)" }),
    limit: Type.Optional(Type.Number({ description: "Max results (default 20)" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const body = { query: args.query }
      if (args.limit) body.limit = args.limit
      return result(await callTool(ctx, "/facts/search", body))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const factForget = {
  name: "fact_forget",
  label: "Facts: forget",
  description:
    "Soft-delete a fact (sets deleted_at; default search excludes it afterwards).",
  promptSnippet: "fact_forget — soft-delete a fact by id",
  parameters: Type.Object({
    fact_id: Type.Number({ description: "Fact id to forget" }),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const sid = sessionIDOf(ctx)
      return result(
        await callTool(ctx, `/facts/${args.fact_id}/forget`, {
          session_id: sid || undefined,
        }),
      )
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}
