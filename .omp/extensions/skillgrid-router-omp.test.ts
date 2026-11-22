// @ts-nocheck
// node --experimental-strip-types .omp/extensions/skillgrid-router-omp.test.ts
// Unit checks for the omp full-router project extension guards (TICKET-07).
import assert from "node:assert"
import path from "node:path"
import { fileURLToPath } from "node:url"

import extension from "./skillgrid-router-omp.ts"

const here = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(here, "..", "..")

let pass = 0
const checkPromises = []
function check(name, fn) {
  checkPromises.push(
    fn()
      .then(() => {
        pass++
        console.log(`ok - ${name}`)
      })
      .catch((e) => {
        console.error(`FAIL - ${name}: ${e.message}`)
        process.exitCode = 1
      }),
  )
}

function makePi() {
  const handlers = {}
  const pi = { on: (name, handler) => { handlers[name] = handler } }
  return { pi, handlers }
}

check("in-project main agent: router section appended to systemPrompt", async () => {
  const { pi, handlers } = makePi()
  extension(pi)
  const systemPrompt = ["base section"]
  handlers.before_agent_start({ systemPrompt }, { cwd: REPO_ROOT, agent: { kind: "main" } })
  assert.equal(systemPrompt.length, 2, "one section appended")
  assert.ok(systemPrompt[1].includes("using-skillgrid"), "router content present")
})

check("non-project cwd (/tmp): nothing appended", async () => {
  const { pi, handlers } = makePi()
  extension(pi)
  const systemPrompt = ["base section"]
  handlers.before_agent_start({ systemPrompt }, { cwd: "/tmp", agent: { kind: "main" } })
  assert.equal(systemPrompt.length, 1, "no section appended outside a project")
})

check("non-main agent kind: nothing appended (R3 subagent skip)", async () => {
  const { pi, handlers } = makePi()
  extension(pi)
  const systemPrompt = ["base section"]
  handlers.before_agent_start({ systemPrompt }, { cwd: REPO_ROOT, agent: { kind: "subagent" } })
  assert.equal(systemPrompt.length, 1, "subagent skipped")
})

check("missing systemPrompt array: fail-open, no throw", async () => {
  const { pi, handlers } = makePi()
  extension(pi)
  assert.doesNotThrow(() => handlers.before_agent_start({}, { cwd: REPO_ROOT, agent: { kind: "main" } }))
})

await Promise.all(checkPromises)
console.log(`${pass} checks passed`)
