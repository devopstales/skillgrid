import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { test } from "node:test"

const source = readFileSync(new URL("./mnemonic.ts", import.meta.url), "utf8")

let runtimeImport = 0
const PROJECT_ID = "project-1"
const MODEL_SESSION_ID = "model-invented"
const RESOLUTION_ERROR = /could not resolve an authoritative session for/
const REGISTRATION_ERROR = /could not confirm session registration for/
const CHILD_SESSIONS = new Map([
  ["leaf", session("leaf", "root")],
  ["root", session("root")],
])

function session(id, parentID, projectID = PROJECT_ID) {
  return { id, ...(parentID === undefined ? {} : { parentID }), projectID }
}

function sdkResult(data, { error, status = 200 } = {}) {
  return { data, error, response: { status } }
}

function missingSDKResult() {
  return sdkResult(undefined, { error: { name: "NotFound" }, status: 404 })
}

function sdkLookup(sessions) {
  return ({ path }) => sdkResult(sessions.get(path.id))
}

function httpResponse(data, ok = true) {
  const payload = data ?? { created: true, project_id: PROJECT_ID, session_id: "s" }
  return { ok, async json() { return payload } }
}

function deferredEvent() {
  let emit
  const event = new Promise((resolve) => { emit = resolve })
  return { event, emit }
}

function deferredResponse() {
  const started = deferredEvent()
  const response = deferredEvent()
  return {
    handler() {
      started.emit()
      return response.event
    },
    started: started.event,
    resolve: response.emit,
  }
}

function toolOutput(...sessionIDs) {
  const sessionID = sessionIDs.length === 0 ? MODEL_SESSION_ID : sessionIDs[0]
  return { args: sessionID === undefined ? {} : { session_id: sessionID } }
}

async function assertRejection(pending, output, matcher) {
  assert.notEqual(output.args.session_id, MODEL_SESSION_ID)
  await assert.rejects(pending, matcher)
}

function assertNoRegistration(runtime, message) {
  assert.deepEqual(runtime.registeredIDs, [], message)
}

async function createRuntime(t, {
  sessionGet = async ({ path }) => sdkResult(session(path.id)),
  registrationResponse,
  contextResponse,
} = {}) {
  const originalFetch = globalThis.fetch
  const originalBun = globalThis.Bun
  const registeredIDs = []
  const sessionGetIDs = []
  const requests = []
  globalThis.Bun = {
    spawnSync(args) {
      if (args.includes("remote")) return { exitCode: 1, stdout: Buffer.from("") }
      return { exitCode: 0, stdout: Buffer.from("/work/mnemonic\n") }
    },
    spawn() {},
    file() { return { async exists() { return false } } },
  }
  globalThis.fetch = async (url, init) => {
    const path = new URL(url).pathname
    if (path === "/health") return { ok: true, async json() { return { status: "ok" } } }
    const body = init?.body ? JSON.parse(init.body) : undefined
    requests.push({ path, url: String(url), body })
    if (path === "/sessions") {
      registeredIDs.push(body.id)
      if (registrationResponse) return registrationResponse(registeredIDs.length)
      return httpResponse()
    }
    if (path === "/context/compaction" && contextResponse) return contextResponse()
    return httpResponse({})
  }

  t.after(() => {
    globalThis.fetch = originalFetch
    globalThis.Bun = originalBun
  })
  runtimeImport += 1
  const moduleURL = new URL(`./mnemonic.ts?mnemonic-runtime=${runtimeImport}`, import.meta.url)
  const { Mnemonic } = await import(moduleURL.href)
  const plugin = await Mnemonic({
    directory: "/work/mnemonic",
    project: { id: PROJECT_ID },
    client: {
      session: {
        async get(request) {
          sessionGetIDs.push(request.path.id)
          return sessionGet(request)
        },
      },
    },
  })
  return {
    plugin,
    event: (type, info) => plugin.event({ event: { type, properties: { info } } }),
    before: plugin["tool.execute.before"],
    chat: plugin["chat.message"],
    after: plugin["tool.execute.after"],
    compact: plugin["experimental.session.compacting"],
    transform: plugin["experimental.chat.system.transform"],
    registeredIDs,
    sessionGetIDs,
    requests,
  }
}

test("protocol documents passive capture, privacy, and compaction recovery", () => {
  assert.match(source, /## Passive Capture/)
  assert.match(source, /## Privacy/)
  assert.match(source, /## After Compaction/)
  assert.match(source, /mem_session_summary/)
  assert.match(source, /Key Learnings/)
  assert.match(source, /<private>\u2026<\/private>|<private>/)
})

test("write hook binds only the four attributed mem_* writes to session_id", () => {
  assert.match(source, /SUB_AGENT_TOOLS = new Set\(\[[\s\S]*"mem_save"[\s\S]*"mem_save_prompt"[\s\S]*"mem_session_summary"[\s\S]*"mem_capture_passive"/)
  assert.match(source, /"tool.execute.before"/)
  assert.match(source, /output\.args\.session_id = sessionId/)
  assert.match(source, /RESOLUTION_ERROR_PREFIX/)
  assert.match(source, /REGISTRATION_ERROR_PREFIX/)
})

test("registration enters the cache only after a successful acknowledgement", async (t) => {
  const runtime = await createRuntime(t, {
    registrationResponse: (attempt) => attempt === 1
      ? httpResponse({ error: "unavailable" }, false)
      : httpResponse(),
  })
  await runtime.event("session.created", session("runtime"))
  assert.deepEqual(runtime.registeredIDs, ["runtime"])

  // First write after a failed ack retries → attempt 2; the following
  // write must not re-POST because the id is now in the known cache.
  for (const expectedRegistrations of [2, 2]) {
    const output = toolOutput(undefined)
    await runtime.before({ tool: "mem_save", sessionID: "runtime" }, output)
    assert.equal(output.args.session_id, "runtime")
    assert.equal(runtime.registeredIDs.length, expectedRegistrations)
  }
})

test("write hook is fail-closed when the authoritative session is unresolved", async (t) => {
  const runtime = await createRuntime(t, {
    sessionGet: async ({ path }) => (path.id === "orphan" ? missingSDKResult() : sdkResult(session(path.id))),
  })
  const output = toolOutput()
  await assert.rejects(
    runtime.before({ tool: "mem_capture_passive", sessionID: "orphan" }, output),
    RESOLUTION_ERROR,
  )
  assert.equal(output.args.session_id, MODEL_SESSION_ID, "failed resolution must not forward MCP arguments")
  assertNoRegistration(runtime)
})

test("a fresh plugin resolves a top-level session and registers it", async (t) => {
  const runtime = await createRuntime(t)
  const output = toolOutput(undefined)
  await runtime.before({ tool: "mem_save", sessionID: "persisted-root" }, output)

  assert.equal(output.args.session_id, "persisted-root")
  assert.deepEqual(runtime.registeredIDs, ["persisted-root"])
})

test("a fresh plugin follows an unobserved child to its root without registering the child", async (t) => {
  const runtime = await createRuntime(t, { sessionGet: sdkLookup(CHILD_SESSIONS) })
  const output = toolOutput(undefined)
  await runtime.before({ tool: "mem_session_summary", sessionID: "leaf" }, output)

  assert.equal(output.args.session_id, "root")
  assert.deepEqual(runtime.sessionGetIDs, ["leaf", "root"])
  assert.deepEqual(runtime.registeredIDs, ["root"], "the leaf child must never be registered")
})

test("SDK lookup failures remain fail-closed and retryable", async (t) => {
  let attempt = 0
  const runtime = await createRuntime(t, {
    sessionGet: async ({ path }) => {
      attempt += 1
      if (attempt === 1) return missingSDKResult()
      if (attempt === 2) throw new Error("SDK unavailable")
      return sdkResult(session(path.id))
    },
  })
  const first = toolOutput()
  await assert.rejects(runtime.before({ tool: "mem_save_prompt", sessionID: "retry-root" }, first), RESOLUTION_ERROR)
  assert.equal(runtime.sessionGetIDs.length, 1)
  assertNoRegistration(runtime)

  const second = toolOutput()
  await assert.rejects(runtime.before({ tool: "mem_save_prompt", sessionID: "retry-root" }, second), RESOLUTION_ERROR)
  assert.equal(runtime.sessionGetIDs.length, 2)
  assertNoRegistration(runtime)

  const recovered = toolOutput(undefined)
  await runtime.before({ tool: "mem_save_prompt", sessionID: "retry-root" }, recovered)
  assert.equal(recovered.args.session_id, "retry-root")
  assert.deepEqual(runtime.registeredIDs, ["retry-root"])
})

test("missing ancestors abort without registering the observed child", async (t) => {
  let ancestorExists = false
  const runtime = await createRuntime(t, {
    sessionGet: async ({ path }) => path.id === "child"
      ? sdkResult(session("child", "missing"))
      : ancestorExists
        ? sdkResult(session("missing"))
        : missingSDKResult(),
  })
  const output = toolOutput()
  await assert.rejects(runtime.before({ tool: "mem_capture_passive", sessionID: "child" }, output), RESOLUTION_ERROR)
  assert.deepEqual(runtime.sessionGetIDs, ["child", "missing"])
  assertNoRegistration(runtime)

  ancestorExists = true
  const recovered = toolOutput(undefined)
  await runtime.before({ tool: "mem_capture_passive", sessionID: "child" }, recovered)
  assert.equal(recovered.args.session_id, "missing")
  assert.deepEqual(runtime.registeredIDs, ["missing"], "a failed chain must not cache or register its leaf")
})

test("invalid, cyclic, and mismatched SDK ownership aborts without registration", async (t) => {
  for (const scenario of [
    { name: "invalid session shape", start: "malformed", sessions: new Map([["malformed", { id: 42, projectID: PROJECT_ID }]]), expectedLookups: ["malformed"] },
    { name: "cyclic parent chain", start: "a", sessions: new Map([["a", session("a", "b")], ["b", session("b", "a")]]), expectedLookups: ["a", "b"] },
    { name: "self-parent chain", start: "self", sessions: new Map([["self", session("self", "self")]]), expectedLookups: ["self"] },
    { name: "cross-project mismatch", start: "foreign", sessions: new Map([["foreign", session("foreign", undefined, "project-2")]]), expectedLookups: ["foreign"] },
    { name: "missing project ID", start: "unscoped", sessions: new Map([["unscoped", { id: "unscoped" }]]), expectedLookups: ["unscoped"] },
  ]) {
    await t.test(scenario.name, async (t) => {
      const runtime = await createRuntime(t, { sessionGet: sdkLookup(scenario.sessions) })
      const output = toolOutput()
      await assert.rejects(runtime.before({ tool: "mem_save", sessionID: scenario.start }, output), RESOLUTION_ERROR)
      assert.deepEqual(runtime.sessionGetIDs, scenario.expectedLookups)
      assertNoRegistration(runtime)
    })
  }
})

test("session.updated reparents a known leaf while deletion tombstones dominate", async (t) => {
  const runtime = await createRuntime(t, {
    sessionGet: async () => { throw new Error("event-cached sessions must not query the SDK") },
  })
  await runtime.event("session.created", session("old-root"))
  await runtime.event("session.created", session("new-root"))
  await runtime.event("session.created", session("leaf", "old-root"))

  const beforeUpdate = toolOutput(undefined)
  await runtime.before({ tool: "mem_save", sessionID: "leaf" }, beforeUpdate)
  assert.equal(beforeUpdate.args.session_id, "old-root")

  await runtime.event("session.updated", session("leaf", "new-root"))
  const afterUpdate = toolOutput(undefined)
  await runtime.before({ tool: "mem_save", sessionID: "leaf" }, afterUpdate)
  assert.equal(afterUpdate.args.session_id, "new-root")
  assert.deepEqual(runtime.registeredIDs, ["old-root", "new-root"])

  await runtime.event("session.deleted", { id: "new-root" })
  const deleted = toolOutput()
  await assert.rejects(runtime.before({ tool: "mem_save", sessionID: "leaf" }, deleted), RESOLUTION_ERROR)
  assert.deepEqual(runtime.sessionGetIDs, [])
  assert.deepEqual(runtime.registeredIDs, ["old-root", "new-root"], "deleted descendants must never revive")
})

test("deleting a leaf during its SDK lookup aborts without mutation or registration", async (t) => {
  const lookup = deferredResponse()
  const runtime = await createRuntime(t, { sessionGet: lookup.handler })
  const output = toolOutput()
  const pending = runtime.before({ tool: "mem_save", sessionID: "leaf" }, output)
  await lookup.started

  await runtime.event("session.deleted", { id: "leaf" })
  lookup.resolve(sdkResult(session("leaf")))

  await assert.rejects(pending, RESOLUTION_ERROR)
  assertNoRegistration(runtime)
})

test("deleting a staged ancestor tombstones its pending descendants across retries", async (t) => {
  const leafLookup = deferredResponse()
  let leafAttempts = 0
  const runtime = await createRuntime(t, {
    sessionGet: async ({ path }) => {
      assert.equal(path.id, "leaf")
      leafAttempts += 1
      if (leafAttempts === 1) return leafLookup.handler()
      return sdkResult(session("leaf"))
    },
  })
  const first = toolOutput()
  const pending = runtime.before({ tool: "mem_save", sessionID: "leaf" }, first)
  await leafLookup.started

  await runtime.event("session.deleted", { id: "root" })
  leafLookup.resolve(sdkResult(session("leaf", "root")))
  await assert.rejects(pending, RESOLUTION_ERROR)

  const retry = toolOutput()
  await assert.rejects(runtime.before({ tool: "mem_save", sessionID: "leaf" }, retry), RESOLUTION_ERROR)
  assert.deepEqual(runtime.sessionGetIDs, ["leaf"], "a tombstoned staged leaf must not query the SDK again")
  assertNoRegistration(runtime)
})

test("deleting an already-staged ancestor tombstones descendants across retries", async (t) => {
  const rootLookup = deferredResponse()
  let leafAttempts = 0
  const runtime = await createRuntime(t, {
    sessionGet: async ({ path }) => {
      if (path.id === "leaf") {
        leafAttempts += 1
        return leafAttempts === 1
          ? sdkResult(session("leaf", "ancestor"))
          : sdkResult(session("leaf"))
      }
      if (path.id === "ancestor") return sdkResult(session("ancestor", "old-root"))
      assert.equal(path.id, "old-root")
      return rootLookup.handler()
    },
  })
  const first = toolOutput()
  const pending = runtime.before({ tool: "mem_save", sessionID: "leaf" }, first)
  await rootLookup.started

  await runtime.event("session.deleted", { id: "ancestor" })
  rootLookup.resolve(sdkResult(session("old-root")))
  await assert.rejects(pending, RESOLUTION_ERROR)

  const retry = toolOutput()
  await assert.rejects(runtime.before({ tool: "mem_save", sessionID: "leaf" }, retry), RESOLUTION_ERROR)
  assert.deepEqual(runtime.sessionGetIDs, ["leaf", "ancestor", "old-root"], "a descendant of the deleted staged ancestor must not query the SDK again")
  assertNoRegistration(runtime)
})

test("a reparent event during leaf lookup overrides the stale SDK parent", async (t) => {
  const lookup = deferredResponse()
  const runtime = await createRuntime(t, { sessionGet: lookup.handler })
  const output = toolOutput(undefined)
  const pending = runtime.before({ tool: "mem_session_summary", sessionID: "leaf" }, output)
  await lookup.started

  await runtime.event("session.updated", session("new-root"))
  await runtime.event("session.updated", session("leaf", "new-root"))
  lookup.resolve(sdkResult(session("leaf", "old-root")))
  await pending

  assert.equal(output.args.session_id, "new-root")
  assert.deepEqual(runtime.sessionGetIDs, ["leaf"])
  assert.deepEqual(runtime.registeredIDs, ["new-root"])
})

test("write tool hook revalidates ownership after registration", async (t) => {
  for (const scenario of [
    { name: "leaf reparented", mutate: async (r) => { await r.event("session.updated", session("new-root")); await r.event("session.updated", session("leaf", "new-root")) } },
    { name: "ancestor reparented", mutate: async (r) => { await r.event("session.updated", session("new-root")); await r.event("session.updated", session("ancestor", "new-root")) } },
    { name: "leaf deleted", mutate: (r) => r.event("session.deleted", { id: "leaf" }) },
    { name: "root ancestor deleted", mutate: (r) => r.event("session.deleted", { id: "old-root" }) },
  ]) {
    await t.test(scenario.name, async (t) => {
      const registration = deferredResponse()
      const runtime = await createRuntime(t, {
        sessionGet: async () => { throw new Error("event-cached ownership must not query the SDK") },
        registrationResponse: registration.handler,
      })
      await runtime.event("session.updated", session("old-root"))
      await runtime.event("session.updated", session("ancestor", "old-root"))
      await runtime.event("session.updated", session("leaf", "ancestor"))

      const output = toolOutput()
      const pending = runtime.before({ tool: "mem_save", sessionID: "leaf" }, output)
      await registration.started
      await scenario.mutate(runtime)
      registration.resolve(httpResponse())

      await assert.rejects(pending, RESOLUTION_ERROR)
      assert.deepEqual(runtime.registeredIDs, ["old-root"])
      assert.deepEqual(runtime.sessionGetIDs, [])
    })
  }
})

test("chat.message resolves an unobserved child and skips its prompt", async (t) => {
  const runtime = await createRuntime(t, { sessionGet: sdkLookup(CHILD_SESSIONS) })
  await runtime.chat(
    { sessionID: "leaf" },
    { message: {}, parts: [{ type: "text", text: "A sufficiently long child prompt" }] },
  )

  assert.deepEqual(runtime.sessionGetIDs, ["leaf", "root"])
  assertNoRegistration(runtime, "an unobserved child prompt must not register the child or root")
  assert.equal(runtime.requests.some(({ path }) => path === "/prompts"), false)
})

test("short chat prompts are not captured", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.chat(
    { sessionID: "runtime" },
    { message: {}, parts: [{ type: "text", text: "ok" }] },
  )
  assert.equal(runtime.requests.some(({ path }) => path === "/prompts"), false)
})

test("private tags are stripped from captured prompts", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.chat(
    { sessionID: "runtime" },
    { message: {}, parts: [{ type: "text", text: "Deploy with the <private>sk-1234567890abcdef</private> token, please" }] },
  )
  const prompt = runtime.requests.find(({ path }) => path === "/prompts")
  assert.ok(prompt, "a prompt POST is expected")
  assert.equal(prompt.body.session_id, "runtime")
  assert.doesNotMatch(prompt.body.content, /sk-1234567890abcdef/)
  assert.match(prompt.body.content, /\[REDACTED\]/)
})

test("Task passive capture resolves an unobserved child and attributes the root", async (t) => {
  const runtime = await createRuntime(t, { sessionGet: sdkLookup(CHILD_SESSIONS) })
  await runtime.after({ tool: "Task", sessionID: "leaf" }, "A".repeat(60))

  assert.deepEqual(runtime.sessionGetIDs, ["leaf", "root"])
  assert.deepEqual(runtime.registeredIDs, ["root"], "the child must never be registered")
  const passive = runtime.requests.find(({ path }) => path === "/observations/passive")
  assert.equal(passive?.body.session_id, "root")
  assert.equal(passive?.body.source, "task-complete")
})

test("short Task output below the passive threshold is not captured", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.after({ tool: "Task", sessionID: "runtime" }, "done, nothing else")
  assert.equal(runtime.requests.some(({ path }) => path === "/observations/passive"), false)
})

test("compaction resolves an unobserved child and registers only its root", async (t) => {
  const runtime = await createRuntime(t, {
    sessionGet: sdkLookup(CHILD_SESSIONS),
    contextResponse: () => httpResponse({ context: "root-only context" }),
  })
  const output = { context: [] }
  await runtime.compact({ sessionID: "leaf" }, output)

  assert.deepEqual(runtime.sessionGetIDs, ["leaf", "root"])
  assert.deepEqual(runtime.registeredIDs, ["root"], "compaction must never register the child")
  const compactionContext = runtime.requests.find(({ path }) => path === "/context/compaction")
  assert.equal(new URL(compactionContext?.url).searchParams.get("session_id"), "root")
  assert.equal(runtime.requests.some(({ path }) => path === "/context"), false)
  assert.ok(output.context.includes("root-only context"))
  assert.match(output.context.at(-1), /FIRST ACTION REQUIRED/)
})

test("compaction skips invalid or missing sessions and still injects the recovery rule", async (t) => {
  for (const scenario of [
    { name: "invalid", prepare: (r) => r.event("session.deleted", { id: "runtime" }), sessionGet: async () => { throw new Error("tombstoned sessions must not query the SDK") } },
    { name: "missing", prepare: () => Promise.resolve(), sessionGet: async () => missingSDKResult() },
  ]) {
    await t.test(scenario.name, async (t) => {
      const runtime = await createRuntime(t, { sessionGet: scenario.sessionGet })
      await scenario.prepare(runtime)
      const output = { context: [] }
      await runtime.compact({ sessionID: "runtime" }, output)

      assertNoRegistration(runtime)
      assert.equal(runtime.requests.some(({ path }) => path === "/context/compaction" || path === "/context"), false)
      assert.match(output.context.at(-1), /FIRST ACTION REQUIRED/)
    })
  }
})

test("compaction omits server context when the endpoint is unavailable but still injects the rule", async (t) => {
  const runtime = await createRuntime(t, {
    contextResponse: () => httpResponse({ context: "must not inject" }, false),
  })
  const output = { context: [] }
  await runtime.compact({ sessionID: "runtime" }, output)

  assert.equal(runtime.requests.filter(({ path }) => path === "/context/compaction").length, 1)
  assert.equal(runtime.requests.some(({ path }) => path === "/context"), false)
  assert.equal(output.context.includes("must not inject"), false)
  assert.match(output.context.at(-1), /FIRST ACTION REQUIRED/)
})

test("automatic hooks omit writes when ownership changes during registration", async (t) => {
  for (const scenario of [
    {
      name: "chat.message",
      invoke: ({ chat }) => chat(
        { sessionID: "runtime" },
        { message: {}, parts: [{ type: "text", text: "A sufficiently long root prompt" }] },
      ),
      forbiddenPath: "/prompts",
    },
    {
      name: "Task passive capture",
      invoke: ({ after }) => after({ tool: "Task", sessionID: "runtime" }, "A".repeat(60)),
      forbiddenPath: "/observations/passive",
    },
  ]) {
    await t.test(scenario.name, async (t) => {
      const registration = deferredResponse()
      const runtime = await createRuntime(t, { registrationResponse: registration.handler })
      await runtime.event("session.updated", session("runtime"))
      const pending = scenario.invoke(runtime)
      await registration.started
      await runtime.event("session.updated", session("new-root"))
      await runtime.event("session.updated", session("runtime", "new-root"))
      registration.resolve(httpResponse())
      await pending

      assert.deepEqual(runtime.registeredIDs, ["runtime"])
      assert.equal(runtime.requests.some(({ path }) => path === scenario.forbiddenPath), false)
    })
  }
})

test("the write hook retries registration and binds children to their parent", async (t) => {
  const runtime = await createRuntime(t, {
    sessionGet: async ({ path }) => ["unresolved", "orphan"].includes(path.id)
      ? missingSDKResult()
      : sdkResult(session(path.id)),
    registrationResponse: (attempt) => attempt === 1
      ? httpResponse({ error: "unavailable" }, false)
      : httpResponse(),
  })
  const first = toolOutput()
  let registrationErrorMessage = ""
  await assert.rejects(
    runtime.before({ tool: "mem_save", sessionID: "runtime" }, first),
    (error) => {
      registrationErrorMessage = error.message
      assert.match(error.message, /could not confirm session registration for/)
      assert.match(error.message, /verify that the Mnemonic server is available and retry/)
      return true
    },
  )
  assert.equal(first.args.session_id, MODEL_SESSION_ID, "failed registration must not forward MCP arguments")

  const second = toolOutput()
  await runtime.before({ tool: "mem_save", sessionID: "runtime" }, second)
  assert.equal(second.args.session_id, "runtime")
  assert.deepEqual(runtime.registeredIDs, ["runtime", "runtime"])

  await runtime.event("session.created", session("sub", "runtime"))
  const subagent = toolOutput("sub")
  await runtime.before({ tool: "mem_session_summary", sessionID: "sub" }, subagent)
  assert.equal(subagent.args.session_id, "runtime")
  assert.equal(runtime.registeredIDs.length, 2, "child must reuse the confirmed parent, not register itself")

  const unresolved = toolOutput()
  let resolutionErrorMessage = ""
  await assert.rejects(
    runtime.before({ tool: "mem_capture_passive", sessionID: "unresolved" }, unresolved),
    (error) => {
      resolutionErrorMessage = error.message
      assert.match(error.message, RESOLUTION_ERROR)
      return true
    },
  )
  assert.notEqual(resolutionErrorMessage, registrationErrorMessage)
  assert.equal(unresolved.args.session_id, MODEL_SESSION_ID, "failed resolution must not forward MCP arguments")
  assert.equal(runtime.registeredIDs.length, 2)

  await runtime.event("session.created", { id: "orphan", parentID: "" })
  const orphan = toolOutput(undefined)
  await assert.rejects(runtime.before({ tool: "mem_capture_passive", sessionID: "orphan" }, orphan), RESOLUTION_ERROR)
  await runtime.event("session.updated", session("orphan", "runtime"))
  await runtime.before({ tool: "mem_capture_passive", sessionID: "orphan" }, orphan)
  assert.equal(orphan.args.session_id, "runtime", "a later authoritative mapping must remain retryable")
  assert.equal(runtime.registeredIDs.length, 2)
})

test("a non-mem_* tool is left untouched by the write hook", async (t) => {
  const runtime = await createRuntime(t)
  const output = { args: {} }
  await runtime.before({ tool: "Bash", sessionID: "runtime" }, output)
  assert.equal(output.args.session_id, undefined)
  assertNoRegistration(runtime)
})

test("a title-only session.created event registers an authoritative root", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.event("session.created", { ...session("legitimate-root"), title: "Task (legitimate subagent)" })

  const output = toolOutput(undefined)
  await runtime.before({ tool: "mem_capture_passive", sessionID: "legitimate-root" }, output)
  assert.equal(output.args.session_id, "legitimate-root")
  assert.deepEqual(runtime.registeredIDs, ["legitimate-root"])
  assert.deepEqual(runtime.sessionGetIDs, [], "event-cached roots must not query the SDK")
})

test("deleting a parent invalidates descendants and prevents later writes or re-registration", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.event("session.created", session("parent"))
  await runtime.event("session.created", session("child", "parent"))
  await runtime.event("session.created", session("grandchild", "child"))

  const confirmed = toolOutput(undefined)
  await runtime.before({ tool: "mem_save", sessionID: "grandchild" }, confirmed)
  assert.equal(confirmed.args.session_id, "parent")
  await runtime.event("session.deleted", { id: "parent" })

  for (const sessionID of ["child", "grandchild"]) {
    const output = toolOutput()
    await assert.rejects(runtime.before({ tool: "mem_save_prompt", sessionID }, output), RESOLUTION_ERROR)
    await runtime.event("session.created", session(sessionID))
    await assert.rejects(runtime.before({ tool: "mem_session_summary", sessionID }, toolOutput(undefined)), RESOLUTION_ERROR)
  }

  assert.deepEqual(runtime.registeredIDs, ["parent"], "invalid descendants must never re-register as top-level sessions")
})
