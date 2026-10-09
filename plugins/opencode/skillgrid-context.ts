// @ts-nocheck
// Context Harness CLM request path (ADR-0027). Renders the Go-owned mirror and
// withholds the oldest tool messages when the calibrated estimate exceeds
// budget - reserve. Off by default: with CLM disabled the request path is
// byte-identical. Every step is fail-open; a read/parse error never blocks.
import * as fs from "node:fs"
import * as path from "node:path"

const BASE_FACTOR = 4.0

function readMirror(directory) {
  try {
    const p = path.join(directory, "clm-mirror.txt")
    return fs.readFileSync(p, "utf8")
  } catch {
    return ""
  }
}

// parseCLMConfig reads the clm block from .skillgrid/config.yaml (searching up
// from the start dir) or honours the SKILLGRID_CTX_CLM env (1/true on). It
// returns { enabled, budget, reserve } with the Go defaults for the budget
// (0) and reserve (2048). Any failure yields enabled=false (byte-identical
// path).
function parseCLMConfig(start) {
  try {
    const env = process.env.SKILLGRID_CTX_CLM
    if (env === "1" || env.toLowerCase() === "true") {
      return { enabled: true, budget: 0, reserve: 2048 }
    }
    let dir = start
    for (let i = 0; i < 10; i++) {
      const file = path.join(dir, ".skillgrid", "config.yaml")
      let raw
      try {
        raw = fs.readFileSync(file, "utf8")
      } catch {
        const parent = path.dirname(dir)
        if (parent === dir) break
        dir = parent
        continue
      }
      const m = raw.match(/clm:\s*\n([\s\S]*?)(?=\n\S|\n\n|$)/)
      if (!m) return { enabled: false, budget: 0, reserve: 2048 }
      const block = m[1]
      const en = block.match(/enabled:\s*(\S+)/)
      if (!en || (en[1] !== "true" && en[1] !== "True")) {
        return { enabled: false, budget: 0, reserve: 2048 }
      }
      const b = block.match(/budget:\s*(\d+)/)
      const r = block.match(/reserve:\s*(\d+)/)
      return {
        enabled: true,
        budget: b ? parseInt(b[1], 10) : 0,
        reserve: r ? parseInt(r[1], 10) : 2048,
      }
    }
    return { enabled: false, budget: 0, reserve: 2048 }
  } catch {
    return { enabled: false, budget: 0, reserve: 2048 }
  }
}

function messageText(entry) {
  try {
    const parts = (entry && entry.parts) || []
    let s = ""
    for (const p of parts) {
      if (p && p.type === "text" && typeof p.text === "string") s += p.text
    }
    return s
  } catch {
    return ""
  }
}

// withholdOldestToolMessages removes the text parts of the oldest tool
// messages in place (and drops a message left with no parts). Fail-open.
function withholdOldestToolMessages(messages) {
  try {
    const toolIdx = []
    for (let i = 0; i < messages.length; i++) {
      if (messages[i] && messages[i].info && messages[i].info.role === "tool") {
        toolIdx.push(i)
      }
    }
    for (const i of toolIdx) {
      const entry = messages[i]
      entry.parts = (entry.parts || []).filter(
        (p) => !(p && p.type === "text")
      )
    }
    for (let i = messages.length - 1; i >= 0; i--) {
      if (!messages[i].parts || messages[i].parts.length === 0) {
        messages.splice(i, 1)
      }
    }
  } catch {
    // fail-open: leave messages untouched
  }
}

export const SkillgridContext = async (ctx) => ({
  "experimental.chat.messages.transform": async (input, output) => {
    try {
      const messages = (output && output.messages) || null
      if (!messages || messages.length === 0) return
      const directory = (ctx && ctx.directory) || process.cwd()
      const cfg = parseCLMConfig(directory)
      if (!cfg.enabled) return
      readMirror(directory)
      let chars = 0
      for (const entry of messages) {
        chars += messageText(entry).length
      }
      const estimate = Math.floor(chars / BASE_FACTOR)
      if (!cfg.budget) return
      if (estimate + cfg.reserve <= cfg.budget) return
      withholdOldestToolMessages(messages)
    } catch {
      // fail-open: never block the request
    }
  },
})
