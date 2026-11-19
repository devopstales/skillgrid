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
import * as fs from "node:fs"
import * as os from "node:os"
import * as path from "node:path"

const mod = await import("../skillgrid-router.ts")

let passed = 0
function check(name, fn) {
  try {
    fn()
    passed++
    console.log("ok -", name)
  } catch (err) {
    console.error("FAIL -", name)
    console.error(err)
    process.exitCode = 1
  }
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

const REPO_ROOT = process.cwd()
const NOT_PROJECT = path.join(os.tmpdir(), "skillgrid-router-not-a-project")
const FAKE_PROJECT = path.join(os.tmpdir(), "skillgrid-router-fake-project")
fs.mkdirSync(path.join(FAKE_PROJECT, ".skillgrid"), { recursive: true })
fs.writeFileSync(path.join(FAKE_PROJECT, ".skillgrid", "config.yaml"), "ticketing:\n  enabled: true\n")

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
  assert.equal(puts.length, 1, "in-project session.created put exactly one entry")
  assert.equal(puts[0].key, "skillgrid-router")
  assert.equal(puts[0].sessionID, "s1")
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

console.log(`\n${passed} checks passed`)
