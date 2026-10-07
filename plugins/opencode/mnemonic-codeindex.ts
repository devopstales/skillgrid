// @ts-nocheck
const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

async function api(method, path, body) {
  const res = await fetch(BASE_URL + path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(10000),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`)
  }
  return data
}

const codeStatus = {
  description:
    "Check code index health before searching. If stale=true, run code_index before code_search.",
  args: {
    project: tool
      .schema.string()
      .optional()
      .describe("Optional project id. Wins over cwd resolution."),
  },
  async execute(args) {
    const params = args.project ? `?project=${args.project}` : ""
    const data = await api("GET", `/code/status${params}`)
    return JSON.stringify(data)
  },
}

const codeIndex = {
  description:
    "Run incremental code index for the cwd git root (respects indexing.yaml). Call after clone or when code_status reports stale.",
  args: {
    project: tool
      .schema.string()
      .optional()
      .describe("Optional project id. Wins over cwd resolution."),
  },
  async execute(args) {
    const data = await api("POST", "/code/index", args)
    return JSON.stringify(data)
  },
}

const codeSearch = {
  description:
    "BM25 full-text search over indexed code chunks. Prefer this over grep/rg when exploring unknown areas of a large repo. Use code_read only after search narrows path and line range.",
  args: {
    query: tool.schema.string().describe("Search terms (FTS5)"),
    project: tool
      .schema.string()
      .optional()
      .describe("Optional project id. Wins over cwd resolution."),
    limit: tool
      .schema.number()
      .optional()
      .describe("Maximum hits (default 20)"),
  },
  async execute(args) {
    const params = new URLSearchParams()
    if (args.query) params.set("query", args.query)
    if (args.project) params.set("project", args.project)
    if (args.limit) params.set("limit", String(args.limit))
    const data = await api("GET", `/code/search?${params}`)
    return JSON.stringify(data)
  },
}

const codeRead = {
  description:
    "Fetch indexed source for a path (and optional line range) after code_search narrows the location. Do not read whole files speculatively.",
  args: {
    path: tool
      .schema.string()
      .describe("Repo-relative file path from code_search"),
    project: tool
      .schema.string()
      .optional()
      .describe("Optional project id. Wins over cwd resolution."),
    start_line: tool
      .schema.number()
      .optional()
      .describe("Start line (1-based); omit to read all indexed chunks"),
    end_line: tool
      .schema.number()
      .optional()
      .describe("End line (1-based); defaults to start_line"),
  },
  async execute(args) {
    const params = new URLSearchParams()
    if (args.path) params.set("path", args.path)
    if (args.project) params.set("project", args.project)
    if (args.start_line) params.set("start_line", String(args.start_line))
    if (args.end_line) params.set("end_line", String(args.end_line))
    const data = await api("GET", `/code/read?${params}`)
    return JSON.stringify(data)
  },
}

const codeFiles = {
  description:
    "List indexed files for a project. Returns file paths and metadata.",
  args: {
    project: tool
      .schema.string()
      .optional()
      .describe("Optional project id. Wins over cwd resolution."),
    limit: tool
      .schema.number()
      .optional()
      .describe("Maximum files (default 100)"),
  },
  async execute(args) {
    const params = new URLSearchParams()
    if (args.project) params.set("project", args.project)
    if (args.limit) params.set("limit", String(args.limit))
    const data = await api("GET", `/code/files?${params}`)
    return JSON.stringify(data)
  },
}

export const MnemonicCodeIndex = async () => ({
  tool: {
    code_status: tool(codeStatus),
    code_index: tool(codeIndex),
    code_search: tool(codeSearch),
    code_read: tool(codeRead),
    code_files: tool(codeFiles),
  },
})
