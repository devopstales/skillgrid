// opencode router plugin behavior test (change 2026-11-17-skillgrid-host-plugins, TICKET-04).
//
// Plain-Node smoke test (no test framework, no @opencode/plugin dependency):
// runs the module's V2 default entry and V1 named export against stubbed contexts
// and asserts the router bootstrap contract:
//   - V2: session.created -> instructions.entry.put({sessionID, key: "skillgrid-router", value})
//   - V2: in-project cwd -> put called with the bootstrap (live asset / static fallback)
//   - V2: non-project cwd -> not called
//   - V2: subagent session -> not called
//   - V2: missing asset -> static fallback value
//   - V1: SkillgridRouter() -> hooks.event -> session.created -> same guards
//
// Run: node --experimental-strip-types plugins/opencode/test/skillgrid-router.test.ts

import assert from "node:assert/strict"
import { fileURLToPath } from "node:url"
import * as fs from "node:fs"
import * as os from "node:os"
import * as path from "node:path"

const mod = await import("../skillgrid-router.ts")

let passed = 0
const checkPromises = []
function check(name, fn) {
  // async-aware: the pass count reflects completed assertions, and a failed
  // assertion is reported (not masked as an unhandled rejection at exit)
  checkPromises.push(
    Promise.resolve()
      .then(() => fn())
      .then(
        () => {
          passed++
          console.log("ok -", name)
        },
        (err) => {
          console.error("FAIL -", name)
          console.error(err)
          process.exitCode = 1
        },
      ),
  )
}

function makeStream(events) {
  const gen = async function* () {
    for (const ev of events) yield ev
  }
  return gen()
}

function makeV2Ctx(directory, events) {
  const puts = []
  const ctx = {
    directory,
    session: {
      instructions: {
        entry: {
          put: async (entry) => {
            puts.push(entry)
          },
        },
      },
    },
    event: {
      subscribe: () => makeStream(events),
    },
  }
  return { ctx, puts }
}

// Derived from this file's location so the suite passes from any cwd.
const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..", "..")
const NOT_PROJECT = path.join(os.tmpdir(), "skillgrid-router-not-a-project")
const FAKE_PROJECT = path.join(os.tmpdir(), "skillgrid-router-fake-project")
fs.mkdirSync(path.join(FAKE_PROJECT, ".skillgrid"), { recursive: true })
fs.writeFileSync(path.join(FAKE_PROJECT, ".skillgrid", "config.yaml"), "ticketing:\n  enabled: true\n")

// The path-(a) listener runs in the background (setup resolves immediately),
// so give it a bounded number of turns to process the queued event before
// asserting on puts.
const drainPuts = async (puts: unknown[], want: number, maxTurns = 50): Promise<void> => {
  for (let i = 0; i < maxTurns && puts.length !== want; i++) {
    const { promise, resolve } = Promise.withResolvers<void>()
    setImmediate(resolve)
    await promise
  }
}

// ── V2 entry shape ────────────────────────────────────────────────────────────
check("v2 default export is Plugin.define-shaped ({id, setup})", () => {
  assert.ok(mod.default, "default export present")
  assert.equal(mod.default.id, "skillgrid.router")
  assert.equal(typeof mod.default.setup, "function")
})

// ── V2: in-project session.created -> durable context entry ────────────────────
check("v2 in-project: session.created puts the bootstrap context entry", async () => {
  const events = [{ type: "session.created", properties: { sessionID: "s1", agent: "main" } }]
  const { ctx, puts } = makeV2Ctx(REPO_ROOT, events)
  await mod.default.setup(ctx)
  await drainPuts(puts, 1)
  assert.equal(puts.length, 1, "in-project session.created put exactly one entry")
  assert.equal(puts[0].key, "skillgrid-router")
  assert.equal(puts[0].sessionID, "s1")
  assert.ok(
    typeof puts[0].value === "string" && puts[0].value.length > 0,
    "entry value carries the bootstrap",
  )
})

// ── V2: walk-up — config above the session start dir ───────────────────────────
check("v2 walk-up: session start dir below the project root puts the bootstrap", async () => {
  const sub = path.join(FAKE_PROJECT, "sub", "dir")
  fs.mkdirSync(sub, { recursive: true })
  const events = [{ type: "session.created", properties: { sessionID: "s7", agent: "main" } }]
  const { ctx, puts } = makeV2Ctx(sub, events)
  await mod.default.setup(ctx)
  await drainPuts(puts, 1)
  assert.equal(puts.length, 1, "walk-up found the config above the start dir; one entry put")
  assert.equal(puts[0].key, "skillgrid-router")
  assert.equal(puts[0].sessionID, "s7")
  assert.ok(
    typeof puts[0].value === "string" && puts[0].value.length > 0,
    "entry value carries the bootstrap",
  )
})

// ── V2: non-project cwd -> no entry ────────────────────────────────────────────
check("v2 non-project: session.created puts nothing", async () => {
  const events = [{ type: "session.created", properties: { sessionID: "s2", agent: "main" } }]
  const { ctx, puts } = makeV2Ctx(NOT_PROJECT, events)
  await mod.default.setup(ctx)
  assert.equal(puts.length, 0, "non-project cwd put nothing")
})

// ── V2: subagent session -> no entry ───────────────────────────────────────────
check("v2 subagent: session.created puts nothing", async () => {
  const events = [{ type: "session.created", properties: { sessionID: "s3", parentID: "parent-1" } }]
  const { ctx, puts } = makeV2Ctx(REPO_ROOT, events)
  await mod.default.setup(ctx)
  assert.equal(puts.length, 0, "subagent session put nothing")
})

// ── V2: missing asset -> static fallback ───────────────────────────────────────
check("v2 missing asset: static fallback is put", async () => {
  const prevHome = process.env.HOME
  process.env.HOME = path.join(os.tmpdir(), "skillgrid-router-empty-home")
  try {
    const events = [{ type: "session.created", properties: { sessionID: "s4", agent: "main" } }]
    const { ctx, puts } = makeV2Ctx(FAKE_PROJECT, events)
    await mod.default.setup(ctx)
    await drainPuts(puts, 1)
    assert.equal(puts.length, 1)
    assert.ok(
      puts[0].value.includes("router bootstrap fallback"),
      "missing asset -> static fallback value",
    )
  } finally {
    process.env.HOME = prevHome
  }
})

// ── V1: named export loadable + same guards ────────────────────────────────────
check("v1 named export SkillgridRouter is a function", () => {
  assert.equal(typeof mod.SkillgridRouter, "function")
})

check("v1 in-project: session.created puts the bootstrap context entry", async () => {
  const puts = []
  const v1ctx = {
    directory: REPO_ROOT,
    session: { instructions: { entry: { put: async (entry) => { puts.push(entry) } } } },
  }
  const v1hooks = await mod.SkillgridRouter(v1ctx)
  assert.equal(typeof v1hooks.event, "function", "v1 hooks expose the event handler")
  await v1hooks.event({ event: { type: "session.created", properties: { sessionID: "s5", agent: "main" } } })
  assert.equal(puts.length, 1, "v1 in-project session.created put an entry")
  assert.equal(puts[0].key, "skillgrid-router")
})

check("v1 non-project: session.created puts nothing", async () => {
  const puts2 = []
  const v1ctx2 = { directory: NOT_PROJECT, session: { instructions: { entry: { put: async (e) => { puts2.push(e) } } } } }
  const v1hooks2 = await mod.SkillgridRouter(v1ctx2)
  await v1hooks2.event({ event: { type: "session.created", properties: { sessionID: "s6", agent: "main" } } })
  assert.equal(puts2.length, 0, "v1 non-project put nothing")
})

// ── V2 fallback (b): opencode 2.0.x — no instructions namespace, "context" hook ──
function makeFallbackCtx(directory) {
  const hooks = {}
  const ctx = {
    directory,
    session: {
      hook: (name, handler) => {
        hooks[name] = handler
      },
    },
    event: {
      subscribe: () => makeStream([]),
    },
  }
  return { ctx, hooks }
}

check("v2 fallback (no instructions ns): context hook registered once", async () => {
  const { ctx, hooks } = makeFallbackCtx(REPO_ROOT)
  await mod.default.setup(ctx)
  assert.equal(typeof hooks.context, "function", "context hook registered")
  assert.equal(Object.keys(hooks).length, 1, "only the context hook is registered")
})

check("v2 fallback: in-project context event appends bootstrap to system", async () => {
  const { ctx, hooks } = makeFallbackCtx(REPO_ROOT)
  await mod.default.setup(ctx)
  const system = [{ type: "text", text: "base system prompt" }]
  await hooks.context({ system, messages: [] })
  assert.equal(system.length, 2, "bootstrap appended to system")
  assert.ok(
    system[1].text && system[1].text.includes("using-skillgrid"),
    "appended part is the router bootstrap",
  )
})

check("v2 fallback: non-project context event appends nothing", async () => {
  const { ctx, hooks } = makeFallbackCtx(NOT_PROJECT)
  await mod.default.setup(ctx)
  const system = ["base system prompt"]
  await hooks.context({ system, messages: [] })
  assert.equal(system.length, 1, "non-project appended nothing")
})

check("v2 fallback: already-injected system is not duplicated", async () => {
  const { ctx, hooks } = makeFallbackCtx(REPO_ROOT)
  await mod.default.setup(ctx)
  const system = [{ type: "text", text: "# Skillgrid Router Bootstrap\n\nThis project is configured with Skillgrid." }]
  await hooks.context({ system, messages: [] })
  assert.equal(system.length, 1, "no second copy appended")
})

check("v2 fallback: AGENTS.md-style 'using-skillgrid' text is not a false positive", async () => {
  const { ctx, hooks } = makeFallbackCtx(REPO_ROOT)
  await mod.default.setup(ctx)
  // AGENTS.md contains "using-skillgrid" but is not the router bootstrap;
  // the old marker skipped injection in-repo (T7 live smoke, 2026-11-22).
  const system = [{ type: "text", text: "Entry: at the start of any skillgrid task, first read using-skillgrid." }]
  await hooks.context({ system, messages: [] })
  assert.equal(system.length, 2, "bootstrap appended despite 'using-skillgrid' in project content")
})

check("v2 fallback: daemon cwd resolves the session's own directory", async () => {
  // `opencode run` connects to the persistent --service daemon whose cwd is
  // unrelated to the session's project and whose ctx.directory is undefined;
  // the start dir must come from session.get(...).location.directory
  // (T7 live smoke, 2026-11-22: in-repo bootstrap silently skipped).
  const hooks = {}
  let lookedUp = null
  const ctx = {
    directory: undefined,
    session: {
      hook: (name, handler) => { hooks[name] = handler },
      get: async (args) => { lookedUp = args; return { location: { directory: REPO_ROOT } } },
    },
    event: { subscribe: () => makeStream([]) },
  }
  await mod.default.setup(ctx)
  const system = [{ type: "text", text: "base system prompt" }]
  await hooks.context({ sessionID: "ses_test", agent: "build", system, messages: [] })
  assert.ok(lookedUp && lookedUp.sessionID === "ses_test", "session record consulted for the start dir")
  assert.equal(system.length, 2, "bootstrap appended via the session directory")
})

// await all checks so the pass count reflects completed (not just started) assertions
await Promise.all(checkPromises)
console.log(`\n${passed} checks passed`)
