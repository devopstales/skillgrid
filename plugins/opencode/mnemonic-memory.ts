// @ts-nocheck
const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

const AGENT = process.env.SKILLGRID_AGENT || "opencode"

async function api(method, path, body) {
  const res = await fetch(BASE_URL + path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(5000),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`)
  }
  return data
}

const memSave = {
  description:
    "Save a curated observation to persistent memory. Use structured content with What/Why/Where/Learned sections. Reuse topic_key to upsert evolving topics.",
  args: {
    title: tool
      .schema.string()
      .describe("Short searchable title (verb + what)"),
    type: tool
      .schema.string()
      .describe(
        "Observation type: decision, architecture, bugfix, pattern, config, discovery, learning, preference, convention"
      ),
    content: tool
      .schema.string()
      .describe(
        "Structured body with What/Why/Where/Learned sections"
      ),
    session_id: tool
      .schema.string()
      .describe("Active session ID from mem_session_start"),
    topic_key: tool
      .schema.string()
      .optional()
      .describe("Stable key for upserts, e.g. architecture/auth-model"),
    scope: tool
      .schema.string()
      .optional()
      .describe("Visibility scope: project (default), user, or global"),
    owner: tool
      .schema.string()
      .optional()
      .describe("Optional creating user/agent identity"),
    capture_prompt: tool
      .schema.boolean()
      .optional()
      .describe(
        "Link the session's latest user prompt (default true). Pass false for automated saves."
      ),
    infer: tool
      .schema.boolean()
      .optional()
      .describe(
        "Deterministically infer empty metadata (default false)."
      ),
    project: tool
      .schema.string()
      .optional()
      .describe("Optional explicit project name to record under"),
    tool_name: tool
      .schema.string()
      .optional()
      .describe("Optional provenance for which tool produced the save"),
  },
  async execute(args) {
    const data = await api("POST", "/memory/observations", args)
    return JSON.stringify(data)
  },
}

const memSearch = {
  description:
    "Full-text search over saved observations using FTS5. Pass project to scope, or all_projects=true to span every store.",
  args: {
    query: tool.schema.string().describe("Search keywords"),
    project: tool
      .schema.string()
      .optional()
      .describe("Optional project name to search under"),
    scope: tool
      .schema.string()
      .optional()
      .describe("Optional visibility scope filter (project|user|global)"),
    limit: tool
      .schema.number()
      .optional()
      .describe("Maximum results (default 20)"),
    all_projects: tool
      .schema.boolean()
      .optional()
      .describe(
        "Span every project store and merge results by cross-project rank"
      ),
    match_mode: tool
      .schema.string()
      .optional()
      .describe("Term matching: any (default) or all"),
    reader_agent: tool
      .schema.string()
      .optional()
      .describe("Optional reader agent id for restricted/agent ACL grants"),
    reader_owner: tool
      .schema.string()
      .optional()
      .describe(
        "Optional reader identity for per-owner visibility enforcement"
      ),
    unfold: tool
      .schema.string()
      .optional()
      .describe(
        "Comma-separated observation ids to return in full. Other hits stay a preview."
      ),
  },
  async execute(args) {
    const params = new URLSearchParams()
    if (args.query) params.set("query", args.query)
    if (args.project) params.set("project", args.project)
    if (args.scope) params.set("scope", args.scope)
    if (args.limit) params.set("limit", String(args.limit))
    if (args.all_projects) params.set("all_projects", "true")
    if (args.match_mode) params.set("match_mode", args.match_mode)
    if (args.reader_agent) params.set("reader_agent", args.reader_agent)
    if (args.reader_owner) params.set("reader_owner", args.reader_owner)
    if (args.unfold) params.set("unfold", args.unfold)
    const data = await api("GET", `/memory/observations?${params}`)
    return JSON.stringify(data)
  },
}

const memGetObservation = {
  description: "Fetch full untruncated observation content by ID.",
  args: {
    id: tool.schema.number().describe("Observation ID from mem_search"),
    reader_agent: tool
      .schema.string()
      .optional()
      .describe("Optional reader agent id for restricted/agent ACL grants"),
    reader_owner: tool
      .schema.string()
      .optional()
      .describe(
        "Optional reader identity for per-owner visibility enforcement"
      ),
  },
  async execute(args) {
    const params = new URLSearchParams()
    if (args.reader_agent) params.set("reader_agent", args.reader_agent)
    if (args.reader_owner) params.set("reader_owner", args.reader_owner)
    const qs = params.toString() ? `?${params}` : ""
    const data = await api("GET", `/memory/observations/${args.id}${qs}`)
    return JSON.stringify(data)
  },
}

const factAdd = {
  description:
    "Add a durable fact to Fact Memory. Returns the new fact id.",
  args: {
    content: tool.schema.string().describe("Fact text (required)"),
    session_id: tool
      .schema.string()
      .optional()
      .describe("Session id to attribute the trail event to"),
    project: tool
      .schema.string()
      .optional()
      .describe("Project id (defaults to CWD resolve)"),
  },
  async execute(args) {
    const data = await api("POST", "/facts", args)
    return JSON.stringify(data)
  },
}

const factSearch = {
  description:
    "Lexical FTS search over Fact Memory (soft-deleted facts excluded). Returns matching facts ranked by bm25.",
  args: {
    query: tool.schema.string().describe("Search terms (any-term recall)"),
    session_id: tool
      .schema.string()
      .optional()
      .describe("Session id to attribute the trail event to"),
    project: tool
      .schema.string()
      .optional()
      .describe("Project id (defaults to CWD resolve)"),
    limit: tool
      .schema.number()
      .optional()
      .describe("Max results (default 20)"),
    include_deleted: tool
      .schema.boolean()
      .optional()
      .describe("Include soft-deleted facts (default false)"),
  },
  async execute(args) {
    const params = new URLSearchParams()
    if (args.query) params.set("query", args.query)
    if (args.session_id) params.set("session_id", args.session_id)
    if (args.project) params.set("project", args.project)
    if (args.limit) params.set("limit", String(args.limit))
    if (args.include_deleted) params.set("include_deleted", "true")
    const data = await api("GET", `/facts/search?${params}`)
    return JSON.stringify(data)
  },
}

const factForget = {
  description:
    "Soft-delete a fact (sets deleted_at; the row survives for the audit trail and default search excludes it afterwards).",
  args: {
    fact_id: tool.schema.number().describe("Fact id to forget"),
    session_id: tool
      .schema.string()
      .optional()
      .describe("Session id to attribute the trail event to"),
    project: tool
      .schema.string()
      .optional()
      .describe("Project id (defaults to CWD resolve)"),
  },
  async execute(args) {
    const data = await api("POST", `/facts/${args.fact_id}/forget`, {
      session_id: args.session_id,
      project: args.project,
    })
    return JSON.stringify(data)
  },
}

const factDecay = {
  description:
    "Apply the AKL importance decay to one fact: importance_score *= exp(-decay_rate * age_days). Returns the new score.",
  args: {
    fact_id: tool.schema.number().describe("Fact id to decay"),
    session_id: tool
      .schema.string()
      .optional()
      .describe("Session id to attribute the trail event to"),
    project: tool
      .schema.string()
      .optional()
      .describe("Project id (defaults to CWD resolve)"),
  },
  async execute(args) {
    const data = await api("POST", `/facts/${args.fact_id}/decay`, {
      session_id: args.session_id,
      project: args.project,
    })
    return JSON.stringify(data)
  },
}

const memSessionStart = {
  description:
    "Create a new workspace session. Required before mem_save in OpenCode plugin flows. Optional title names the session.",
  args: {
    directory: tool
      .schema.string()
      .optional()
      .describe("Workspace directory (defaults to cwd)"),
    title: tool
      .schema.string()
      .optional()
      .describe("Optional human-readable session name"),
  },
  async execute(args) {
    const data = await api("POST", "/sessions", args)
    return JSON.stringify(data)
  },
}

const memSessionEnd = {
  description: "End a session with optional summary.",
  args: {
    session_id: tool.schema.string().describe("Session ID to end"),
    summary: tool
      .schema.string()
      .optional()
      .describe("Optional end-of-session summary"),
  },
  async execute(args) {
    const data = await api(
      "POST",
      `/sessions/${args.session_id}/end`,
      args.summary ? { summary: args.summary } : {}
    )
    return JSON.stringify(data)
  },
}

const memSessionSummary = {
  description: "Persist structured end-of-session summary before closing.",
  args: {
    session_id: tool.schema.string().describe("Session ID"),
    summary: tool
      .schema.string()
      .describe(
        "Structured session summary (Goal, Discoveries, Accomplished, Next Steps, Relevant Files)"
      ),
  },
  async execute(args) {
    const data = await api(
      "POST",
      `/sessions/${args.session_id}/summary`,
      args
    )
    return JSON.stringify(data)
  },
}

export const MnemonicMemory = async () => ({
  tool: {
    mem_save: tool(memSave),
    mem_search: tool(memSearch),
    mem_get_observation: tool(memGetObservation),
    fact_add: tool(factAdd),
    fact_search: tool(factSearch),
    fact_forget: tool(factForget),
    fact_decay: tool(factDecay),
    mem_session_start: tool(memSessionStart),
    mem_session_end: tool(memSessionEnd),
    mem_session_summary: tool(memSessionSummary),
  },
})
