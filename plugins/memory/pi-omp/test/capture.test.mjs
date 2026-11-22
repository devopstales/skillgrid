import test from "node:test"
import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { mkdtemp } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { createCapture } from "../src/capture.js"

test("capture: blank prompt and missing session are no-ops (no spawn)", async () => {
  const calls = []
  const c = createCapture({ bin: "mnemonic", session: { sessionId: () => "s1" }, spawn: async (...a) => calls.push(a) })
  await c.capture("/repo", "   ")
  await c.capture("/repo", undefined)
  const c2 = createCapture({ bin: "mnemonic", session: { sessionId: () => null }, spawn: async () => { throw new Error("must not spawn") } })
  await c2.capture("/repo", "hello")
  assert.equal(calls.length, 0)
})

test("capture: prompt travels on stdin with the adapter's session id", async () => {
  const calls = []
  const c = createCapture({ bin: "mnemonic", session: { sessionId: () => "s1" }, spawn: async (...a) => calls.push(a), dataDir: () => tmpdir(), projectOf: () => "p" })
  await c.capture("/repo", "hello world")
  assert.deepEqual(calls[0][0], ["mnemonic", "memory", "save-prompt", "--id", "s1"])
  assert.equal(calls[0][1], "hello world")
  assert.equal(calls[0][2], "/repo")
})

test("capture: failing spawn increments the drop counter, never throws", async () => {
  const dataDir = await mkdtemp(join(tmpdir(), "mnemonic-drops-"))
  const c = createCapture({ bin: "mnemonic", session: { sessionId: () => "s1" }, spawn: async () => { throw new Error("exit 1") }, dataDir: () => dataDir, projectOf: () => "p" })
  await c.capture("/repo", "hello")
  await c.capture("/repo", "again")
  const rec = JSON.parse(readFileSync(join(dataDir, "p", ".capture-drops.json"), "utf8"))
  assert.equal(rec.n, 2)
  assert.ok(rec.lastAt)
})
