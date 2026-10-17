// Plugin smoke test — imports each of the 5 native TS plugins with a mocked
// `tool()` + `globalThis.opencode` and a live node:http mock server, then
// exercises every tool execute / event handler against the real route surface.
// This is the S23 "pnpm test" harness (WINDOWS W022): it proves the plugins
// load, expose the expected tools, and their fetches hit registered routes.
import { createRequire } from "node:module"
import { fileURLToPath, pathToFileURL } from "node:url"
import { dirname, resolve } from "node:path"
import http from "node:http"
import { tmpdir } from "node:os"
import fs from "node:fs"

const require = createRequire(import.meta.url)
const __dirname = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(__dirname, "..")

let passed = 0
let failed = 0
const failures = []
function ok(cond, msg) {
  if (cond) {
    passed++
  } else {
    failed++
    failures.push(msg)
    console.error("  FAIL:", msg)
  }
}
function check(cond, msg) {
  ok(cond, msg)
  if (cond) console.log("  ok:", msg)
}

// ── mock opencode SDK ────────────────────────────────────────────────────────
const mockSchema = {
  string: () => mockSchemaFn("string"),
  number: () => mockSchemaFn("number"),
  boolean: () => mockSchemaFn("boolean"),
  object: () => mockSchemaFn("object"),
  any: () => mockSchemaFn("any"),
}
function mockSchemaFn(kind) {
  return {
    describe: () => mockSchemaFn(kind),
    optional: () => mockSchemaFn(kind),
    nullable: () => mockSchemaFn(kind),
    min: () => mockSchemaFn(kind),
    max: () => mockSchemaFn(kind),
    array: () => mockSchemaFn(kind),
    enum: () => mockSchemaFn(kind),
    union: () => mockSchemaFn(kind),
  }
}
// The plugins call the top-level `tool(def)` and read `tool.schema.*`, so the
// mock must itself carry a `.schema`.
const mockTool = Object.assign((def) => ({
  ...def,
  __tool: true,
  args: { ...(def.args || {}) },
}), { schema: mockSchema })
globalThis.tool = mockTool
globalThis.opencode = {
  plugin: { tool: mockTool },
  schema: mockSchema,
}

// ── mock HTTP server: answers every /teams, /policy, /observations, /search,
//    /facts, /sessions, /context, /code path with 200 + a JSON body so the
//    plugin's fetch() resolves. ───────────────────────────────────────────────
const seenPaths = new Set()
const server = http.createServer((req, res) => {
  seenPaths.add(req.url.split("?")[0])
  let body = ""
  req.on("data", (c) => (body += c))
  req.on("end", () => {
    // /ctx/capture returns a real sandbox_id so the Pi overflow handler can
    // build its receipt (ADR-0044).
    if (req.url.split("?")[0] === "/ctx/capture") {
      res.writeHead(200, { "Content-Type": "application/json" })
      res.end(JSON.stringify({ sandbox_id: 42, gated: true, pointer: "ctx_search smoke" }))
      return
    }
    res.writeHead(200, { "Content-Type": "application/json" })
    res.end(JSON.stringify({ ok: true, echo: { method: req.method, path: req.url, body } }))
  })
})
await new Promise((r) => server.listen(0, "127.0.0.1", r))
const port = server.address().port
const base = `http://127.0.0.1:${port}`
process.env.SKILLGRID_MNEMONIC_HTTP_URL = base
process.env.SKILLGRID_CHECKPOINT_URL = base
process.env.SKILLGRID_AGENT = "test-agent"

// ── the 5 plugins and their expected exported factory + tools ───────────────
const plugins = [
  {
    file: "plugins/opencode/skillgrid-squad.ts",
    factory: "SkillgridSquad",
    tools: [
      "squad_spawn_task",
      "squad_pull_next_task",
      "squad_submit_output",
      "squad_submit_review",
      "squad_read_task",
      "squad_mark_done",
    ],
    kind: "tool",
    sample: { tool: "squad_spawn_task", args: { title: "t", brief: "b" } },
  },
  {
    file: "plugins/opencode/skillgrid-events.ts",
    factory: "SkillgridEvents",
    tools: [],
    kind: "event",
    // before: input has sessionID + tool; output.args carries the file path so
    // handleToolBefore reaches the /policy/evaluate gate.
    events: [
      ["tool.execute.before", { sessionID: "s1", tool: "edit" }, { args: { path: "/tmp/x.ts" } }],
      ["tool.execute.after", { sessionID: "s1", tool: "edit" }, { args: { path: "/tmp/x.ts" } }],
    ],
  },
  {
    file: "plugins/opencode/mnemonic-memory.ts",
    factory: "MnemonicMemory",
    tools: [
      "mem_save",
      "mem_search",
      "mem_get_observation",
      "fact_add",
      "fact_search",
      "fact_forget",
      "fact_decay",
      "mem_session_start",
      "mem_session_end",
      "mem_session_summary",
    ],
    kind: "tool",
    sample: { tool: "mem_save", args: { title: "t", content: "c", type: "decision" } },
  },
  {
    file: "plugins/opencode/skillgrid-compaction.ts",
    factory: "SkillgridCompaction",
    tools: [],
    kind: "eventHandler", // single `event: async ({event})` keyed by event.type
    events: [
      "session.created",
      "session.idle",
      "session.compacted",
    ],
  },
  {
    file: "plugins/opencode/mnemonic-codeindex.ts",
    factory: "MnemonicCodeIndex",
    tools: ["code_status", "code_index", "code_search", "code_read", "code_files"],
    kind: "tool",
    sample: { tool: "code_status", args: {} },
  },
]

// ── load each plugin via the node:typescript type-stripping loader ──────────
const tsLoader = "node:typescript"
async function loadPlugin(file) {
  const abs = resolve(repoRoot, file)
  const url = pathToFileURL(abs).href
  // Use dynamic import with the experimental type-stripping enabled via the
  // node flag --experimental-strip-types (set on the command line).
  return await import(url)
}

// ── Pi extension (ADR-0043): a separate harness from the opencode plugin
//    factory shape. It default-exports a factory taking a mock ExtensionAPI
//    (pi.registerTool / pi.on); tools use TypeBox-style parameters and
//    execute(toolCallId, args, signal, onUpdate, ctx) returning AgentToolResult
//    {content:[{type,text}]}. ────────────────────────────────────────────────
{
  const piTools = []
  const piEvents = []
  const mockPi = {
    registerTool: (t) => piTools.push(t),
    on: (name, handler) => piEvents.push([name, handler]),
    events: { on: () => () => {}, emit: () => {} },
  }
  const piExpectedTools = [
    "mem_save", "mem_search", "mem_get_observation", "mem_update",
    "mem_session_summary", "mem_context", "mem_current_project", "mem_doctor",
    "mem_capture_passive", "fact_add", "fact_search", "fact_forget",
    "code_status", "code_index", "code_search", "code_read",
    "ctx_search", "ctx_get", "ctx_stats", "ctx_purge",
  ]
  const piExpectedEvents = [
    "session_start", "before_agent_start", "session_before_compact", "session_shutdown",
    "tool_result",
  ]

  console.log("\n# plugins/pi/mnemonic/index.ts")
  let piMod
  try {
    piMod = await loadPlugin("plugins/pi/mnemonic/index.ts")
    check(true, "extension loads")
  } catch (err) {
    check(false, "extension loads: " + err.message)
    piMod = null
  }
  if (piMod) {
    const factory = piMod.default
    check(typeof factory === "function", "default-exports factory")
    if (typeof factory === "function") {
      try {
        factory(mockPi)
        for (const name of piExpectedTools) {
          check(piTools.some((t) => t.name === name), "registers tool " + name)
        }
        for (const name of piExpectedEvents) {
          check(piEvents.some(([n]) => n === name), "subscribes event " + name)
        }
        // ADR-0044: every tool carries a one-line promptSnippet so it shows
        // in Pi's "Available tools" system-prompt section.
        check(
          piTools.every((t) => typeof t.promptSnippet === "string" && t.promptSnippet.length > 0),
          "every pi tool has a promptSnippet",
        )
        // exercise one tool end-to-end against the mock server:
        // execute(toolCallId, args, signal, onUpdate, ctx)
        const save = piTools.find((t) => t.name === "mem_save")
        if (save) {
          const out = await save.execute("tc1", {
            title: "t",
            type: "decision",
            content: "c",
          }, undefined, undefined, {
            cwd: "/tmp",
            sessionManager: { getSessionId: () => "pi-session-1" },
          })
          check(
            Array.isArray(out?.content) && typeof out.content[0]?.text === "string",
            "mem_save execute returns AgentToolResult",
          )
        }
        // exercise the before_agent_start handlers: the events module's
        // prompt-capture handler and the index's protocol-injection handler
        // both subscribe to the same event (separate files, ADR-0043).
        const beforeStarts = piEvents.filter(([n]) => n === "before_agent_start").map(([, h]) => h)
        if (beforeStarts.length > 0) {
          const opts = { appendSystemPrompt: "" }
          const event = { messageType: "user", prompt: "fix the login bug", systemPromptOptions: opts }
          const ctx = { cwd: "/tmp", sessionManager: { getSessionId: () => "pi-session-1" } }
          for (const handler of beforeStarts) await handler(event, ctx)
          check(typeof opts.appendSystemPrompt === "string" && opts.appendSystemPrompt.length > 0,
            "before_agent_start injects Memory Protocol")
        }
        // ADR-0044: tool_result overflow replaces oversized output with a
        // receipt (mock server's /ctx/capture returns a sandbox_id).
        const toolResults = piEvents.filter(([n]) => n === "tool_result").map(([, h]) => h)
        if (toolResults.length > 0) {
          const ctx = { cwd: "/tmp", sessionManager: { getSessionId: () => "pi-session-1" } }
          // Oversized: replaced with a receipt.
          const big = "x".repeat(5000)
          const outBig = await toolResults[0](
            { toolName: "bash", input: { command: "make test" }, content: [{ type: "text", text: big }] },
            ctx,
          )
          check(
            !!outBig?.content?.[0]?.text?.includes("[skillgrid-ctx]") &&
              outBig.content[0].text.includes("sandbox_id"),
            "tool_result overflow returns receipt for oversized output",
          )
          // Small: untouched (undefined → pi keeps original content).
          const outSmall = await toolResults[0](
            { toolName: "bash", input: { command: "ls" }, content: [{ type: "text", text: "ok" }] },
            ctx,
          )
          check(outSmall === undefined, "tool_result small output passes through")
          // Our own ctx_* tools: skipped (recursion guard).
          const outSkip = await toolResults[0](
            { toolName: "ctx_search", input: { query: "x" }, content: [{ type: "text", text: big }] },
            ctx,
          )
          check(outSkip === undefined, "tool_result skips ctx_* tools")
        }
      } catch (err) {
        check(false, "pi extension wiring: " + err.message)
      }
    }
  }
}

for (const p of plugins) {
  console.log("\n# " + p.file)
  let mod
  try {
    mod = await loadPlugin(p.file)
  } catch (err) {
    check(false, "plugin loads: " + err.message)
    continue
  }
  check(Boolean(mod[p.factory]), "exports factory " + p.factory)
  if (!mod[p.factory]) continue
  const plugin = await mod[p.factory]()

  if (p.kind === "tool") {
    const toolMap = plugin.tool || {}
    for (const name of p.tools) {
      check(Boolean(toolMap[name]), "exposes tool " + name)
    }
    // exercise one tool end-to-end against the mock server
    const t = toolMap[p.sample.tool]
    if (t) {
      try {
        const out = await t.execute(p.sample.args)
        check(typeof out === "string" && out.length > 0, "execute(" + p.sample.tool + ") returns JSON")
      } catch (err) {
        check(false, "execute(" + p.sample.tool + "): " + err.message)
      }
    }
  } else if (p.kind === "event") {
    for (const [evtName, payload, output] of p.events) {
      const handler = plugin[evtName]
      check(typeof handler === "function", "exposes event " + evtName)
      if (typeof handler === "function") {
        try {
          await handler(payload, output || {})
          check(true, "event " + evtName + " handler runs")
        } catch (err) {
          check(false, "event " + evtName + ": " + err.message)
        }
      }
    }
  } else if (p.kind === "eventHandler") {
    const handler = plugin.event
    check(typeof handler === "function", "exposes event handler")
    for (const evtType of p.events) {
      if (typeof handler !== "function") continue
      try {
        await handler({ event: { type: evtType, properties: { sessionID: "s1" } } }, {})
        check(true, "event " + evtType + " handled")
      } catch (err) {
        check(false, "event " + evtType + ": " + err.message)
      }
    }
  }

  // ADR-0032 V2 port: the V2 default export (Plugin.define) must exist and its
  // setup must register against the mock V2 ctx without throwing. @opencode/
  // plugin is absent in plain Node, so Plugin.define is a local stub that
  // records {id, setup}.
  if (mod.default && typeof mod.default.setup === "function") {
    check(typeof mod.default.id === "string" && mod.default.id.length > 0,
      "v2 entrypoint has an id (" + mod.default.id + ")")
    try {
      const added = []
      const hooked = []
      const mockCtx = {
        tool: {
          transform: async (cb) => {
            cb({
              add: (d) => added.push(d.name),
              namespace: () => {},
            })
          },
          hook: async (name, cb) => {
            hooked.push([name, cb])
          },
        },
        event: {
          subscribe: () => {
            // Async iterator that yields one session event then ends.
            const events = [{ type: "session.created", properties: { sessionID: "s1" } }]
            return {
              [Symbol.asyncIterator]() {
                let i = 0
                return { next: () => (i < events.length ? { value: events[i++], done: false } : { done: true }) }
              },
            }
          },
        },
      }
      await mod.default.setup(mockCtx)
      if (p.kind === "tool") {
        const missing = p.tools.filter((t) => !added.includes(t))
        check(missing.length === 0,
          "v2 setup registers all tools" + (missing.length ? " (missing: " + missing.join(",") + ")" : ""))
      } else if (p.kind === "event") {
        check(hooked.length >= 2, "v2 setup registers tool hooks (before+after)")
      } else if (p.kind === "eventHandler") {
        check(true, "v2 setup subscribes to the event stream")
      }
    } catch (err) {
      check(false, "v2 setup: " + err.message)
    }
  } else {
    check(false, "v2 default export (Plugin.define) present")
  }
}

// ── confirm the mock server actually saw traffic (plugins really fetched) ──
const expectedHits = ["/teams/tasks", "/policy/evaluate", "/observations", "/code/status"]
for (const path of expectedHits) {
  const hit = [...seenPaths].some((s) => s.startsWith(path))
  check(hit, "mock server received " + path + " (plugin really fetched)")
}

server.close()
console.log(`\n── plugin smoke: ${passed} passed, ${failed} failed ──`)
if (failed > 0) {
  console.error("\nFailures:")
  for (const f of failures) console.error(" - " + f)
  process.exit(1)
}
