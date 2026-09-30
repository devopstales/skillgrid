import assert from "node:assert/strict"
import { mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { test } from "node:test"

const PROJECT_ID = "project-1"
const scratch = mkdtempSync(join(tmpdir(), "mnemonic-capture-"))
process.chdir(scratch)

function httpResponse(data) {
  return { ok: true, async json() { return data ?? { created: true, project_id: PROJECT_ID } } }
}

async function createRuntime(t, { fetchImpl } = {}) {
  const originalFetch = globalThis.fetch
  const originalBun = globalThis.Bun
  const requests = []
  const pendingErrors = []
  globalThis.Bun = {
    spawnSync() { return { exitCode: 1, stdout: Buffer.from("") } },
    spawn() {},
  }
  globalThis.fetch = async (url, init) => {
    const path = new URL(url).pathname
    if (path === "/health") return { ok: true, async json() { return { status: "ok" } } }
    if (fetchImpl) {
      try {
        return await fetchImpl(url, init)
      } catch (error) {
        pendingErrors.push(error)
        throw error
      }
    }
    const body = init?.body ? JSON.parse(init.body) : undefined
    requests.push({ path, url: String(url), body })
    return httpResponse({})
  }
  t.after(() => {
    globalThis.fetch = originalFetch
    globalThis.Bun = originalBun
    rmSync(scratch, { recursive: true, force: true })
  })
  const moduleURL = new URL(`../mnemonic.ts?mnemonic-capture=${Date.now()}-${Math.random()}`, import.meta.url)
  const { Mnemonic } = await import(moduleURL.href)
  const plugin = await Mnemonic({
    directory: "/work/mnemonic",
    project: { id: PROJECT_ID },
    client: {
      session: {
        async get({ path }) {
          return { data: { id: path.id, projectID: PROJECT_ID }, response: { status: 200 } }
        },
      },
    },
  })
  return {
    after: plugin["tool.execute.after"],
    event: (type, info) => plugin.event({ event: { type, properties: { info } } }),
    requests,
    pendingErrors,
  }
}

// SATISFIES plugin-posts-per-tool-call
test("plugin posts per tool call to /sessions/{id}/tool-calls", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.event("session.created", { id: "cap-root", projectID: PROJECT_ID })
  await runtime.after(
    { tool: "bash", sessionID: "cap-root", args: { command: "go test ./..." } },
    { output: "ok" },
  )
  await new Promise((r) => setTimeout(r, 200))
  const call = runtime.requests.find(({ path }) => path === "/sessions/cap-root/tool-calls")
  assert.ok(call, "a tool-call POST is expected; captured: " + JSON.stringify(runtime.requests.map(({ path }) => path)))
  assert.equal(call.body.session_id, "cap-root")
  assert.equal(call.body.tool_name, "bash")
  assert.equal(call.body.command, "go test ./...")
  assert.equal(call.body.agent, "kilo")
  assert.equal(call.body.type, "shell")
  assert.equal(call.body.result_status, "success")
})

// SATISFIES plugin-is-fire-and-forget-never-blocks
test("plugin POST is fire-and-forget (never throws, never blocks)", async (t) => {
  const runtime = await createRuntime(t, {
    fetchImpl: async () => {
      throw new Error("network down")
    },
  })
  await runtime.event("session.created", { id: "ff-root", projectID: PROJECT_ID })
  const pending = runtime.after(
    { tool: "bash", sessionID: "ff-root", args: { command: "ls" } },
    { output: "ok" },
  )
  await Promise.race([
    pending,
    new Promise((_, reject) => setTimeout(() => reject(new Error("hook blocked on a failed fetch")), 500)),
  ])
  await new Promise((r) => setTimeout(r, 50))
  assert.ok(
    runtime.pendingErrors.some((e) => e instanceof Error && e.message === "network down"),
    "the network failure must be swallowed, not rethrown",
  )
})

// SATISFIES private-tools-can-be-allowed
test("SKILLGRID_MNEMONIC_PRIVATE_TOOLS scrubbing: listed private tools are captured with output redacted", async (t) => {
  const originalEnv = process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS
  process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS = "secret_tool"
  t.after(() => {
    if (originalEnv === undefined) delete process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS
    else process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS = originalEnv
  })
  const runtime = await createRuntime(t)
  await runtime.event("session.created", { id: "priv-root", projectID: PROJECT_ID })
  await runtime.after(
    { tool: "secret_tool", sessionID: "priv-root", args: { path: "/a" } },
    { output: "top-secret payload <private>sk-1234</private>" },
  )
  await new Promise((r) => setTimeout(r, 25))
  const call = runtime.requests.find(({ path }) => path === "/sessions/priv-root/tool-calls")
  assert.ok(call, "a private tool must still be captured (observed, output scrubbed)")
  assert.equal(call.body.tool_name, "secret_tool")
  assert.equal(call.body.path, "/a")
  assert.doesNotMatch(call.body.content_preview, /top-secret payload|sk-1234/)
  // Non-private tools capture their output normally.
  await runtime.after(
    { tool: "bash", sessionID: "priv-root", args: { command: "echo hi" } },
    { output: "hi" },
  )
  await new Promise((r) => setTimeout(r, 25))
  const bashCall = runtime.requests.filter(({ path }) => path === "/sessions/priv-root/tool-calls")
    .at(-1)
  assert.equal(bashCall.body.tool_name, "bash")
  assert.match(bashCall.body.content_preview, /hi/)
})

// SATISFIES plugin-posts-per-tool-call (private span stripping)
test("private spans are stripped from tool-call content before hashing/preview", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.event("session.created", { id: "strip-root", projectID: PROJECT_ID })
  await runtime.after(
    { tool: "bash", sessionID: "strip-root", args: { command: "echo done" } },
    { output: "deployed with <private>ghp_abc123</private> token" },
  )
  await new Promise((r) => setTimeout(r, 25))
  const call = runtime.requests.find(({ path }) => path === "/sessions/strip-root/tool-calls")
  assert.ok(call, "a tool-call POST is expected")
  assert.doesNotMatch(call.body.content_preview, /ghp_abc123/)
  assert.match(call.body.content_preview, /\[REDACTED\]/)
  assert.ok(call.body.content_hash.length > 0, "a hash of the stripped content is expected")
})

// SATISFIES plugin-posts-per-tool-call (result_status normalization)
test("result_status is normalized to success|error only", async (t) => {
  const runtime = await createRuntime(t)
  await runtime.event("session.created", { id: "rs-root", projectID: PROJECT_ID })
  // Non-canonical statuses must not pass through verbatim.
  await runtime.after(
    { tool: "bash", sessionID: "rs-root", args: { command: "echo 1" } },
    { output: "1", status: "done" },
  )
  await runtime.after(
    { tool: "bash", sessionID: "rs-root", args: { command: "echo 2" } },
    { output: "2", status: "failed" },
  )
  await new Promise((r) => setTimeout(r, 25))
  const calls = runtime.requests.filter(({ path }) => path === "/sessions/rs-root/tool-calls")
  assert.equal(calls.length, 2)
  assert.equal(calls[0].body.result_status, "success", "non-canonical status must normalize to success")
  assert.equal(calls[1].body.result_status, "error", "'failed' must map to error")
})
