import test from "node:test"
import assert from "node:assert/strict"
import { createSession, resolveProject } from "../src/session.js"

const calls = []
const okSpawn = async (args, stdin, cwd) => { calls.push({ args, stdin, cwd }) }

test("session: registers a minted id and ends the same id", async () => {
  const s = createSession({ bin: "mnemonic", spawn: okSpawn })
  const sid = await s.start("/repo")
  assert.match(sid, /^mem-[0-9a-z]+-[0-9a-f]+$/)
  assert.deepEqual(calls[0].args, ["mnemonic", "memory", "session", "start", "--id", sid])
  assert.equal(calls[0].cwd, "/repo")
  await s.end("/repo")
  assert.deepEqual(calls[1].args, ["mnemonic", "memory", "session", "end", "--session", sid])
})

test("session: spawn failure is fail-open (no throw, id retained for capture)", async () => {
  const s = createSession({ bin: "mnemonic", spawn: async () => { throw new Error("ENOENT") } })
  assert.ok(await s.start("/repo"))
  await s.end("/repo") // no throw
})

test("session: double start keeps one id (factory closure is per-conversation)", async () => {
  const calls = []
  const s = createSession({ bin: "mnemonic", spawn: async (args, stdin, cwd) => { calls.push({ args, stdin, cwd }) } })
  assert.equal(await s.start("/repo"), await s.start("/repo"))
  assert.equal(calls.length, 1)
})

test("resolveProject: MNEMONIC_PROJECT > git toplevel basename > cwd basename", () => {
  const git = (cwd) => (cwd === "/repo/wt" ? "/repo/wt" : null)
  assert.equal(resolveProject("/repo/wt", { env: {}, gitToplevel: git }), "wt")
  assert.equal(resolveProject("/repo/wt", { env: { MNEMONIC_PROJECT: "envp" }, gitToplevel: git }), "envp")
  assert.equal(resolveProject("/a/b/c", { env: {}, gitToplevel: () => null }), "c")
})
