// @ts-nocheck
// mnemonic-memory — pi/omp extension (ADR-0067). v0 (door-check scope):
// session registration + per-prompt capture through the mnemonic binary.
// No tool registration — the mem_* tools come from the mnemonic-memory
// MCP entry. Protocol injection, the session-start probe, and compaction
// recovery land in Task 6. Every path fail-open.
// <BIN> is replaced with the absolute binary path by the installer (Task 3).
import { spawn, spawnSync } from "node:child_process"
import { createSession, resolveProject } from "./src/session.js"
import { createCapture } from "./src/capture.js"

const BIN = "<BIN>"

// Production spawn: absolute binary + fixed verb words only (no shell),
// 5 s hard timeout, the payload on stdin; resolves only on exit 0.
function makeSpawn() {
  return (args, stdin, cwd) => {
    const { promise, resolve, reject } = Promise.withResolvers()
    const child = spawn(args[0], args.slice(1), {
      cwd: cwd || process.cwd(),
      stdio: ["pipe", "ignore", "ignore"],
    })
    const timer = setTimeout(() => {
      child.kill("SIGKILL")
      reject(new Error("mnemonic spawn timeout (5s)"))
    }, 5000)
    child.on("error", (e) => { clearTimeout(timer); reject(e) })
    child.on("close", (code) => {
      clearTimeout(timer)
      code === 0 ? resolve() : reject(new Error("mnemonic exit " + code))
    })
    child.stdin.end(stdin ?? "")
    return promise
  }
}

// Subagent guard (oh-my-pi ExtensionAgentIdentity): guard on kind, not
// depth. skill-pi has no identity field — absent reads as main.
function isSubagent(ctx) {
  const kind = ctx?.agent?.kind
  return kind !== undefined && kind !== "main"
}

export default function mnemonicMemoryPiomp(pi) {
  const spawn = makeSpawn()
  const session = createSession({ bin: BIN, spawn })
  const capture = createCapture({ bin: BIN, session, spawn, projectOf: (cwd) => resolveProject(cwd, { gitToplevel: (c) => { const r = spawnSync("git", ["rev-parse", "--show-toplevel"], { cwd: c, encoding: "utf8", timeout: 2000 }); return r.status === 0 ? r.stdout.trim() : null }}) })

  pi.on("session_start", async (_e, ctx) => {
    if (isSubagent(ctx)) return
    await session.start(ctx?.cwd)
  })
  pi.on("before_agent_start", async (event, ctx) => {
    if (isSubagent(ctx)) return
    // omp (oh-my-pi): reason "prompt" (user prompt) vs "revive" (parked
    // subagent revival — discarded, empty prompt); skill-pi (v1 surface):
    // messageType "user". Exactly one host matches per event.
    if (event?.reason !== "prompt" && event?.messageType !== "user") return
    await capture.capture(ctx?.cwd, event?.prompt)
  })
  pi.on("session_shutdown", async (event, ctx) => {
    // omp (oh-my-pi): SessionShutdownEvent carries no reason (fires on
    // process exit — always the conversation end); skill-pi: "quit" only.
    if ((event?.reason !== undefined && event?.reason !== "quit") || isSubagent(ctx)) return
    await session.end(ctx?.cwd)
  })
}
