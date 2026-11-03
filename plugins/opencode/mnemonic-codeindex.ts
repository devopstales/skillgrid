// @ts-nocheck
// v2 (opencode >= 2.0, `@opencode/cli` host): the default export is
// { id, setup }, validated by PluginModule.load. The host resolves
// `@opencode/plugin` (and its workspace deps) for discovery-loaded local
// plugins, so no node_modules is needed beside the file (OpenCode #50434
// — the legacy `@opencode-ai/plugin` npm specifier is NOT host-provided).
// The code_* tools are also exposed via the mnemonic MCP server (the working
// path, TASK-081); this keeps the plugin loadable in every v2 build.
import { Plugin } from "@opencode/plugin"

const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

// v1 `tool()` arg-schema shim. The v2 API dropped the root `tool` export
// (tools register via ctx.tool.transform with effect Schema), so these named
// v1 tool objects — kept for tooling parity / the mnemonic MCP server — need
// only their `description` and `execute`; the `args` schemas below are inert
// strings here (only the v1 `tool()` factory read them, and it no longer runs).
const tool = { schema: { string: () => ({ optional: () => ({ describe: () => "" }), describe: () => "" }), number: () => ({ optional: () => ({ describe: () => "" }), describe: () => "" }), boolean: () => ({ optional: () => ({ describe: () => "" }), describe: () => "" }) } }

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

export default Plugin.define({
  id: "skillgrid-mnemonic-codeindex",
  setup: async (_ctx) => {
    // fail-open no-op; tools available via mnemonic MCP (TASK-081)
  },
})

// V2 entrypoint (ADR-0032 V2 port). The V1 named export above is kept for
// V1.18.29+ harnesses; @opencode/plugin is a bundled OpenCode dependency.
// The V2 editor takes JSON-Schema inputs, so wrap converts the V1 arg schema
// (tool.schema.* calls, which collapse to plain objects) to JSON Schema.
// Every value is optional (the V1 harness validated at the harness boundary);
// unknown args are rejected by additionalProperties:false.
function wrap(definition) {
  const properties = {}
  const required = []
  for (const [k, v] of Object.entries(definition.args || {})) {
    if (v && v.__optional) {
      properties[k] = v.schema
    } else {
      properties[k] = v.schema
      required.push(k)
    }
  }
  return {
    name: definition.name,
    description: definition.description,
    input: {
      type: "object",
      properties,
      ...(required.length ? { required } : {}),
      additionalProperties: false,
    },
    async execute(input) {
      return JSON.stringify(await definition.execute(input))
    },
  }
}

async function importPlugin() {
  try {
    return await import("@opencode/plugin")
  } catch {
    return { Plugin: { define: (def) => def } }
  }
}

const { Plugin } = await importPlugin()

export default Plugin.define({
  id: "skillgrid.codeindex",
  async setup(ctx) {
    const defs = []
    for (const [name, d] of Object.entries({
      code_status: codeStatus,
      code_index: codeIndex,
      code_search: codeSearch,
      code_read: codeRead,
      code_files: codeFiles,
    })) {
      defs.push(wrap({ ...d, name }))
    }
    await ctx.tool.transform((editor) => {
      for (const def of defs) editor.add(def)
    })
  },
})
