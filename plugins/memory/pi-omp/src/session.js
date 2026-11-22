// session.js — adapter-owned session identity (ADR-0067): the adapter
// mints the session id and is the single authority on it (the model
// never invents it). Pure logic: spawn is injected. Fail-open: a dead
// verb never blocks the host session.
import { randomBytes } from "node:crypto"
import { basename } from "node:path"

const mint = () => "mem-" + Date.now().toString(36) + "-" + randomBytes(3).toString("hex")

// Mirror of the CLI project chain (explicit > env > git toplevel >
// basename), env leg first — hooks never pass --project. Scratch-file
// placement ONLY: store resolution stays in the verb (the Go chain).
export function resolveProject(cwd, { env = process.env, gitToplevel } = {}) {
  if (env.MNEMONIC_PROJECT) return env.MNEMONIC_PROJECT
  if (typeof gitToplevel === "function") {
    const top = gitToplevel(cwd || process.cwd())
    if (top) return basename(top)
  }
  return basename(cwd || process.cwd())
}

export function createSession({ bin, spawn }) {
  let sid = null
  return {
    sessionId: () => sid,
    async start(cwd) {
      if (sid) return sid
      sid = mint()
      try { await spawn([bin, "memory", "session", "start", "--id", sid], "", cwd) }
      catch { /* fail-open: the drop counter surfaces a dead verb */ }
      return sid
    },
    async end(cwd) {
      if (!sid) return
      const id = sid
      sid = null
      try { await spawn([bin, "memory", "session", "end", "--session", id], "", cwd) }
      catch { /* fail-open */ }
    },
  }
}
