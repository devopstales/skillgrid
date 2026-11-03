// @ts-nocheck
// Mnemonic memory tools for the Pi extension (ADR-0043): mem_save,
// mem_search, mem_get_observation, mem_update, mem_session_summary,
// mem_context, mem_current_project, mem_doctor, mem_capture_passive.
// HTTP + redaction helpers come from ./api.js (shared with facts/code).

import { Type } from "./api.js"
import { AGENT, bestEffort, callTool, getTool, result, redactValue, sessionIDOf } from "./api.js"

export const memSave = {
  name: "mem_save",
  label: "Memory: save",
  description:
    "Save a curated observation to persistent memory. Use structured content with What/Why/Where/Learned sections. Reuse topic_key to upsert evolving topics.",
  promptSnippet:
    "mem_save — persist a decision/fix/discovery (What/Why/Where/Learned; topic_key upserts)",
  parameters: Type.Object({
    title: Type.String({ description: "Short searchable title (verb + what)" }),
    type: Type.String({
      description:
        "Observation type: decision, architecture, bugfix, pattern, config, discovery, learning, preference, convention",
    }),
    content: Type.String({
      description: "Structured body with What/Why/Where/Learned sections",
    }),
    topic_key: Type.Optional(
      Type.String({
        description: "Stable key for upserts, e.g. architecture/auth-model",
      })
    ),
    scope: Type.Optional(
      Type.String({ description: "Visibility scope: project (default), user, or global" })
    ),
    capture_prompt: Type.Optional(
      Type.Boolean({
        description:
          "Link the session's latest user prompt (default true). Pass false for automated saves.",
      })
    ),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const sid = sessionIDOf(ctx)
      const body = redactValue({
        title: args.title,
        type: args.type,
        content: args.content,
        topic_key: args.topic_key || undefined,
        scope: args.scope || undefined,
        session_id: sid || undefined,
        capture_prompt: args.capture_prompt ?? true,
        tool_name: "mem_save",
      })
      const data = await callTool(ctx, "/observations", body)
      return result(data)
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memSearch = {
  name: "mem_search",
  label: "Memory: search",
  description:
    "Full-text search over saved observations using FTS5. Project-scoped by default; pass project to override.",
  promptSnippet:
    "mem_search — FTS over saved observations (truncated previews; mem_get_observation for full text)",
  parameters: Type.Object({
    query: Type.String({ description: "Search keywords" }),
    project: Type.Optional(Type.String({ description: "Optional project name to search under" })),
    match_mode: Type.Optional(
      Type.String({ description: "Term matching: any (default) or all" })
    ),
    limit: Type.Optional(Type.Number({ description: "Maximum results (default 20)" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const params = new URLSearchParams()
      params.set("query", args.query)
      if (args.project) params.set("project", args.project)
      if (args.match_mode) params.set("match_mode", args.match_mode)
      if (args.limit) params.set("limit", String(args.limit))
      return result(await getTool(ctx, `/search?${params}`))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memGetObservation = {
  name: "mem_get_observation",
  label: "Memory: get",
  description:
    "Fetch full untruncated observation content by ID. Search previews are truncated — always fetch before relying on a hit as a constraint.",
  promptSnippet:
    "mem_get_observation — full untruncated content by id (before relying on a mem_search hit)",
  parameters: Type.Object({
    id: Type.Number({ description: "Observation ID from mem_search" }),
    project: Type.Optional(Type.String({ description: "Optional project name" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const params = args.project ? `?project=${encodeURIComponent(args.project)}` : ""
      return result(await getTool(ctx, `/observations/${args.id}${params}`))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memUpdate = {
  name: "mem_update",
  label: "Memory: update",
  description:
    "Update an existing observation in place (title, content, type, scope, topic_key). Empty fields are left untouched. Use for evolving topics: keep the topic_key, preserve still-valid facts, rewrite only what changed.",
  promptSnippet:
    "mem_update — update an observation in place (keep topic_key; rewrite only what changed)",
  parameters: Type.Object({
    id: Type.Number({ description: "Observation ID" }),
    title: Type.Optional(Type.String({ description: "New title (optional)" })),
    content: Type.Optional(Type.String({ description: "New full content (optional)" })),
    type: Type.Optional(Type.String({ description: "New type (optional)" })),
    topic_key: Type.Optional(Type.String({ description: "New topic_key (optional)" })),
    scope: Type.Optional(Type.String({ description: "New scope (optional)" })),
    project: Type.Optional(Type.String({ description: "Optional project name" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const { id, ...body } = args
      const bodyOut = redactValue(body)
      for (const k of Object.keys(bodyOut)) {
        if (bodyOut[k] === undefined || bodyOut[k] === "") delete bodyOut[k]
      }
      const qs = args.project ? `?project=${encodeURIComponent(args.project)}` : ""
      return result(await callTool(ctx, `/memory/observations/${id}${qs}`, bodyOut))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memSessionSummary = {
  name: "mem_session_summary",
  label: "Memory: session summary",
  description:
    "Persist a structured end-of-session summary. Required before explicitly closing or handing off a session. Body must have the six sections: Goal, Instructions, Discoveries, Accomplished, Next Steps, Relevant Files.",
  promptSnippet:
    "mem_session_summary — six-section end-of-session summary (Goal/Instructions/Discoveries/Accomplished/Next Steps/Files)",
  parameters: Type.Object({
    summary: Type.String({
      description:
        "Structured session summary (Goal, Instructions, Discoveries, Accomplished, Next Steps, Relevant Files)",
    }),
    project: Type.Optional(Type.String({ description: "Optional project name" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const sid = sessionIDOf(ctx)
      if (!sid) return result(JSON.stringify({ error: "no active session" }))
      const data = await callTool(
        ctx,
        "/sessions",
        { id: sid, agent: AGENT, title: args.summary.slice(0, 120) },
      )
      // Best-effort: the session end route owns the row summary but also ends
      // the session, so the summary text is persisted as a typed observation.
      await bestEffort(
        "/observations",
        redactValue({
          title: `session summary: ${sid}`,
          type: "learning",
          content: args.summary,
          session_id: sid,
          capture_prompt: false,
          tool_name: "mem_session_summary",
        }),
      )
      return result(JSON.stringify({ session_id: sid, saved: true, ...data }))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memContext = {
  name: "mem_context",
  label: "Memory: context",
  description:
    "Recent session summaries for the current project — the fast first recall step. Returns the newest sessions with titles and summaries.",
  promptSnippet:
    "mem_context — recent session summaries (first recall step)",
  parameters: Type.Object({
    limit: Type.Optional(Type.Number({ description: "Max sessions (default 5)" })),
    project: Type.Optional(Type.String({ description: "Optional project name" })),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const params = new URLSearchParams()
      if (args.limit) params.set("limit", String(args.limit))
      if (args.project) params.set("project", args.project)
      const qs = params.toString()
      return result(await getTool(ctx, `/context${qs ? "?" + qs : ""}`))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memCurrentProject = {
  name: "mem_current_project",
  label: "Memory: current project",
  description: "Report the project the server resolved for the current working directory.",
  promptSnippet: "mem_current_project — report the resolved project for this cwd",
  parameters: Type.Object({}),
  async execute(_toolCallId, _args, _signal, _onUpdate, ctx) {
    try {
      return result(await getTool(ctx, "/project/current?directory=" + encodeURIComponent(ctx?.cwd || process.cwd())))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memDoctor = {
  name: "mem_doctor",
  label: "Memory: doctor",
  description: "Health diagnostics for the Mnemonic server and the current project store.",
  promptSnippet: "mem_doctor — Mnemonic server + store health diagnostics",
  parameters: Type.Object({}),
  async execute(_toolCallId, _args, _signal, _onUpdate, ctx) {
    try {
      return result(await getTool(ctx, "/memory/doctor"))
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}

export const memCapturePassive = {
  name: "mem_capture_passive",
  label: "Memory: passive capture",
  description:
    "Send raw text (an assistant reply, tool output) to the server's passive extractor, which persists only recognized structured learnings. Use for significant learnings in a final response before replying.",
  promptSnippet:
    "mem_capture_passive — send raw text to the server's passive learning extractor",
  parameters: Type.Object({
    content: Type.String({ description: "Raw text to scan for learnings" }),
    source: Type.Optional(
      Type.String({ description: "Provenance label (default assistant)" })
    ),
  }),
  async execute(_toolCallId, args, _signal, _onUpdate, ctx) {
    try {
      const sid = sessionIDOf(ctx)
      const data = await callTool(
        ctx,
        "/observations/passive",
        redactValue({
          content: args.content,
          session_id: sid || undefined,
          source: args.source || "assistant",
        }),
      )
      return result(data)
    } catch (err) {
      return result(JSON.stringify({ error: String(err?.message || err) }))
    }
  },
}
