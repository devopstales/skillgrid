// memory-prefetch plugin behavior test (change
// 2026-10-24-global-llm-wiki, TICKET-05, blueprint Step 1b).
//
// Plain-Node smoke test (no test framework, no @opencode/plugin dependency):
// runs the exported hooks against a stub event and asserts the frozen-snapshot
// contract:
//   - the context hook fires once per session ID (memoize: the fetch runs
//     once; later requests reuse the cached block)
//   - a second call with the same session ID returns the cached result
//   - a new session ID renders fresh
//   - fetch failure / empty block → no injection (fail-open)
//   - the block is injected at most once per system entry (idempotent)
//
// Run: node plugins/opencode/test/memory-prefetch.test.ts (or the package
// `test` script).
import assert from "node:assert/strict"

const mod = await import("../src/memory-prefetch.ts")

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

function makeEvent(systemText, sessionID) {
  return {
    sessionID,
    system: [{ text: systemText }],
  }
}

// Capture the hook handler out of the plugin definition (the build is
// per-test so each scenario gets a fresh memoization cache).
async function setupHandler(fetchBlock) {
  let handler = null
  const ctx = {
    session: {
      hook: (name, h) => {
        if (name === "context") handler = h
      },
    },
  }
  const built = mod.buildMemoryPrefetchPlugin({ fetchBlock })
  await built.setup(ctx)
  if (!handler) throw new Error("the context hook was not registered")
  return handler
}

await (async () => {
  // 1. Memoize: fetch runs once per session ID, second request returns the
  // cached block (mid-session "adds" to the source are invisible).
  let fetches = 0
  const handler = await setupHandler(async () => {
    fetches++
    return "GLOBAL MEMORY [1% — 22/2,200 chars]\nfact one"
  })
  const event1 = makeEvent("SYSTEM", "sess-a")
  await handler(event1)
  assert.equal(fetches, 1, "fetch must run exactly once for the first request")
  assert.ok(
    event1.system[0].text.includes("fact one"),
    "first request injects the frozen block"
  )
  await handler(event1)
  assert.equal(fetches, 1, "second request for the same session must not re-fetch")
  assert.ok(
    event1.system[0].text.includes("fact one"),
    "cached block is returned verbatim"
  )
  // Idempotent: the block is injected exactly once even if the handler runs
  // again on the same event.
  const occurrences = event1.system[0].text.split("fact one").length - 1
  assert.equal(occurrences, 1, "the block must be injected exactly once")
  check("memoize by session ID (fetch once, cached on later requests)", () => {})
  check("second call with the same session ID returns the cached result", () => {})

  // 2. A different session ID renders fresh.
  const event2 = makeEvent("SYSTEM", "sess-b")
  await handler(event2)
  assert.equal(fetches, 2, "a new session ID must render fresh")
  check("a new session ID renders fresh (per-session cache)", () => {})

  // 3. Fail-open: fetch failure → no injection, no throw.
  const failHandler = await setupHandler(async () => {
    throw new Error("boom")
  })
  const event3 = makeEvent("SYSTEM", "sess-c")
  await failHandler(event3)
  assert.equal(event3.system[0].text, "SYSTEM", "fetch failure must leave the prompt untouched")
  check("fetch error → no injection (fail-open)", () => {})

  // 4. Fail-open: empty block → nothing injected, cached as empty.
  let emptyFetches = 0
  const emptyHandler = await setupHandler(async () => {
    emptyFetches++
    return ""
  })
  const event4 = makeEvent("SYSTEM", "sess-d")
  await emptyHandler(event4)
  await emptyHandler(event4)
  assert.equal(emptyFetches, 1, "an empty block is cached (no re-fetch)")
  assert.equal(event4.system[0].text, "SYSTEM", "empty block must inject nothing")
  check("empty block → no injection, cached (no re-fetch)", () => {})

  // 5. resolveSessionID honors both spellings + falls back.
  assert.equal(mod.resolveSessionID({ sessionID: "x" }), "x")
  assert.equal(mod.resolveSessionID({ sessionId: "y" }), "y")
  assert.equal(mod.resolveSessionID({}), "unknown")
  check("resolveSessionID honors sessionID/sessionId, falls back to unknown", () => {})

  console.log(`\n${passed} checks passed`)
})()
